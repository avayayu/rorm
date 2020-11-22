package rorm

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strconv"
	"sync"
	"time"

	redis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

var (
	ErrLockObtain  = errors.New("lock get failed")
	ErrLockNotHeld = errors.New("lock not held")
)

var (
	luaRefresh = redis.NewScript(`if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("pexpire", KEYS[1], ARGV[2]) else return 0 end`)
	luaRelease = redis.NewScript(`if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`)
	luaPTTL    = redis.NewScript(`if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("pttl", KEYS[1]) else return -3 end`)
)

//RedisMutex 基于redis实现的分布式锁
type RedisMutex struct {
	client RedisClient
	logger *zap.Logger

	tmp   []byte
	tmpMu sync.Mutex

	LockMap sync.Map
}

func New(client RedisClient, logger *zap.Logger) *RedisMutex {
	return &RedisMutex{client: client, logger: logger}
}

func (m *RedisMutex) TryObtain(ctx context.Context, key string, ttl time.Duration, retry RetryStrategy) (lock *Lock, err error) {

	// value := lib
	token, err := m.randomToken()
	if err != nil {
		return nil, err
	}

	var timer *time.Timer
	for deadline := time.Now().Add(ttl); time.Now().Before(deadline); {

		ok, err := m.obtain(key, token, ttl)
		if err != nil {
			return nil, err
		} else if ok {
			return &Lock{client: m.client, key: key, value: token}, nil
		}

		backoff := retry.NextBackoff()
		if backoff < 1 {
			break
		}

		if timer == nil {
			timer = time.NewTimer(backoff)
			defer timer.Stop()
		} else {
			timer.Reset(backoff)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	return

}

func (m *RedisMutex) obtain(key, value string, ttl time.Duration) (bool, error) {
	return m.client.SetNX(context.Background(), key, value, ttl).Result()
}

func (m *RedisMutex) randomToken() (string, error) {
	m.tmpMu.Lock()
	defer m.tmpMu.Unlock()

	if len(m.tmp) == 0 {
		m.tmp = make([]byte, 16)
	}

	if _, err := io.ReadFull(rand.Reader, m.tmp); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(m.tmp), nil
}

// //LockOptions 锁配置
// type LockOptions struct {
// 	retry RetryStrategy
// }

type Lock struct {
	client RedisClient
	key    string
	value  string
	tll    time.Duration
	Status bool
}

// func (l *Lock) Lock() {
// 	l.client.SetNX(context.Background(), l.key, l.value, l.tll)
// }

// TTL returns the remaining time-to-live. Returns 0 if the lock has expired.
func (lock *Lock) TTL() (time.Duration, error) {
	res, err := luaPTTL.Run(context.Background(), lock.client, []string{lock.key}, lock.value).Result()
	if err == redis.Nil {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	if num := res.(int64); num > 0 {
		return time.Duration(num) * time.Millisecond, nil
	}
	return 0, nil
}

// Refresh extends the lock with a new TTL.
// May return ErrNotObtained if refresh is unsuccessful.
func (lock *Lock) Refresh(ttl time.Duration) error {
	ttlVal := strconv.FormatInt(int64(ttl/time.Millisecond), 10)
	status, err := luaRefresh.Run(context.Background(), lock.client, []string{lock.key}, lock.value, ttlVal).Result()
	if err != nil {
		return err
	} else if status == int64(1) {
		return nil
	}
	return ErrLockObtain
}

// Release manually releases the lock.
// May return ErrLockNotHeld.
func (lock *Lock) Release() error {
	res, err := luaRelease.Run(context.Background(), lock.client, []string{lock.key}, lock.value).Result()
	if err == redis.Nil {
		return ErrLockNotHeld
	} else if err != nil {
		return err
	}

	if i, ok := res.(int64); !ok || i != 1 {
		return ErrLockNotHeld
	}
	return nil
}

// --------------------------------------------------------------------

// RetryStrategy allows to customise the lock retry strategy.
type RetryStrategy interface {
	// NextBackoff returns the next backoff duration.
	NextBackoff() time.Duration
}

type linearBackoff time.Duration

// LinearBackoff allows retries regularly with customized intervals
func LinearBackoff(backoff time.Duration) RetryStrategy {
	return linearBackoff(backoff)
}

// NoRetry acquire the lock only once.
func NoRetry() RetryStrategy {
	return linearBackoff(0)
}

func (r linearBackoff) NextBackoff() time.Duration {
	return time.Duration(r)
}

type limitedRetry struct {
	s RetryStrategy

	cnt, max int
}

// LimitRetry limits the number of retries to max attempts.
func LimitRetry(s RetryStrategy, max int) RetryStrategy {
	return &limitedRetry{s: s, max: max}
}

func (r *limitedRetry) NextBackoff() time.Duration {
	if r.cnt >= r.max {
		return 0
	}
	r.cnt++
	return r.s.NextBackoff()
}

type exponentialBackoff struct {
	cnt uint

	min, max time.Duration
}

// ExponentialBackoff strategy is an optimization strategy with a retry time of 2**n milliseconds (n means number of times).
// You can set a minimum and maximum value, the recommended minimum value is not less than 16ms.
func ExponentialBackoff(min, max time.Duration) RetryStrategy {
	return &exponentialBackoff{min: min, max: max}
}

func (r *exponentialBackoff) NextBackoff() time.Duration {
	r.cnt++

	ms := 2 << 25
	if r.cnt < 25 {
		ms = 2 << r.cnt
	}

	if d := time.Duration(ms) * time.Millisecond; d < r.min {
		return r.min
	} else if r.max != 0 && d > r.max {
		return r.max
	} else {
		return d
	}
}

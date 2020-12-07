package rorm

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	RormPTRNeed            = errors.New("pointer parameter need")
	RormDataNotFound       = errors.New("no data found")
	RormModelMustBeStruct  = errors.New("model must be struct type")
	RormFieldNotExist      = errors.New("field not exists in struct")
	RormPrimaryKeyNotFound = errors.New("struct need have one primary key not found")
)

// type Redis interface {
// 	Create(ctx context.Context, v interface{}) error
// 	Find(ctx context.Context, v interface{}) (err error)
// 	Update(ctx context.Context, model interface{}, fieldName string, v interface{}) (err error)
// 	Updates(ctx context.Context, model interface{}, data map[string]interface{}) (err error)
// }

type OrmQuery interface {
	Where(pattern string) *Query
	SubModel(flag bool) *Query
	Expire(d int64) *Query
}

//Loader 实现loader的结构体将自动从数据库加载数据
type RormLoader interface {
	Loader(v interface{}) error
}

type Valuer interface {
	RedisValue() string
}

type Scanner interface {
	RedisScan(src interface{}) error
}

// --------------------------------------------------------------------

// RetryStrategy allows to customise the lock retry strategy.
type RetryStrategy interface {
	// NextBackoff returns the next backoff duration.
	NextBackoff() time.Duration
}

type logger interface {
}

//封装go-redis 因为go-redis支持redis集群
//尽量提供复杂结构体的存储
//尽量提供redis存储的结构体的单个字段的更新

type Redisclient interface {
	Close() error
	Get(context.Context, string) *redis.StringCmd
	HGet(context.Context, string, string) *redis.StringCmd
	Pipeline() redis.Pipeliner
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
	HGetAll(context.Context, string) *redis.StringStringMapCmd
	SetNX(context.Context, string, interface{}, time.Duration) *redis.BoolCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ScriptExists(ctx context.Context, hashes ...string) *redis.BoolSliceCmd
	ScriptLoad(ctx context.Context, script string) *redis.StringCmd
}

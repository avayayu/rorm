package rorm

import (
	"sync"
	"time"

	redis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type BFRRedis struct {
	client  Redisclient //单个node
	logger  *zap.Logger
	tmp     []byte
	tmpMu   sync.Mutex
	LockMap sync.Map
}

type ExpireTime struct {
	Duration time.Duration
}

func NewBFRRedis(options *Options, logger *zap.Logger) *BFRRedis {

	bredis := &BFRRedis{logger: logger}

	if options.Mode == Normal {
		redisOptions := redis.Options{}
		for _, node := range options.AddressMap {
			redisOptions.Addr = node.URL + ":" + node.Port
			redisOptions.DB = node.DB
			redisOptions.Username = node.Username
			redisOptions.Password = node.Password
			break
		}

		client := redis.NewClient(&redisOptions)
		bredis.client = client
	} else {
		redisClusterOptions := redis.ClusterOptions{}
		addrList := []string{}
		for _, node := range options.AddressMap {
			addr := node.URL + ":" + node.Port
			addrList = append(addrList, addr)
			redisClusterOptions.Username = node.Username
			redisClusterOptions.Password = node.Password
		}
		redisClusterOptions.Addrs = addrList
		client := redis.NewClusterClient(&redisClusterOptions)
		bredis.client = client
	}
	return bredis
}

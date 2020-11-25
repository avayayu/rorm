package rorm

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestBFRRedis_TryObtain(t *testing.T) {
	type args struct {
		ctx   context.Context
		key   string
		ttl   time.Duration
		retry RetryStrategy
	}
	tests := []struct {
		name     string
		m        *BFRRedis
		args     args
		wantLock bool
		wantErr  bool
	}{
		// TODO: Add test cases.
		{
			name: "obtain success",
			m:    redisClient,
			args: args{
				ctx:   context.Background(),
				key:   "test250",
				ttl:   time.Second * 10,
				retry: NoRetry(),
			},
			wantErr:  false,
			wantLock: true,
		},
		{
			name: "obtain success",
			m:    redisClient,
			args: args{
				ctx:   context.Background(),
				key:   "test250",
				ttl:   time.Second * 10,
				retry: NoRetry(),
			},
			wantErr:  false,
			wantLock: false,
		},
		{
			name: "obtain success",
			m:    redisClient,
			args: args{
				ctx:   context.Background(),
				key:   "test250",
				ttl:   time.Second * 20,
				retry: LinearBackoff(10),
			},
			wantErr:  false,
			wantLock: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lock, err := tt.m.TryObtain(tt.args.ctx, tt.args.key, tt.args.ttl, tt.args.retry)
			if (err != nil) != tt.wantErr && (lock != nil) == tt.wantLock {
				t.Errorf("BFRRedis.TryObtain() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				fmt.Println("success obtain key")
			}

		})
	}
}

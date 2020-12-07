package rorm

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuery_Update(t *testing.T) {
	type args struct {
		ctx       context.Context
		model     *RedisTest
		fieldName string
		v         interface{}
	}
	testStringPtr := "testStringPtr"
	tests := []struct {
		name    string
		query   *Query
		args    args
		wantErr bool
	}{
		{
			name:  "update with key not exist in redis",
			query: redisClient.NewQuery(),
			args: args{
				ctx:       context.Background(),
				model:     &RedisTest{ID: "qweqweqwee"},
				fieldName: "TEST2",
				v:         2.8,
			},
			wantErr: true,
		},
		// TODO: Add test cases.
		{
			name:  "update single attribute float64",
			query: redisClient.NewQuery(),
			args: args{
				ctx:       context.Background(),
				model:     &RedisTest{ID: "try13"},
				fieldName: "TEST2",
				v:         2.8,
			},
			wantErr: false,
		},
		{
			name:  "update single attribute string",
			query: redisClient.NewQuery(),
			args: args{
				ctx:       context.Background(),
				model:     &RedisTest{ID: "try13"},
				fieldName: "PayLoad",
				v:         "nihao hahaha",
			},
			wantErr: false,
		},
		{
			name:  "update single attribute byte",
			query: redisClient.NewQuery(),
			args: args{
				ctx:       context.Background(),
				model:     &RedisTest{ID: "try13"},
				fieldName: "BYTES",
				v:         byte(2),
			},
			wantErr: false,
		},
		{
			name:  "update single attribute *string",
			query: redisClient.NewQuery(),
			args: args{
				ctx:       context.Background(),
				model:     &RedisTest{ID: "try13"},
				fieldName: "PTRTEST",
				v:         &testStringPtr,
			},
			wantErr: false,
		},
		{
			name:  "update single attribute SLICE",
			query: redisClient.NewQuery(),
			args: args{
				ctx:       context.Background(),
				model:     &RedisTest{ID: "try13"},
				fieldName: "SLICE",
				v:         []string{"2.22", "asdasd", "asd2112231asd"},
			},
			wantErr: false,
		},
		{
			name:  "update single attribute bool",
			query: redisClient.NewQuery(),
			args: args{
				ctx:       context.Background(),
				model:     &RedisTest{ID: "try13"},
				fieldName: "HAHA",
				v:         false,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.query.Update(tt.args.ctx, tt.args.model, tt.args.fieldName, tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("Query.Update() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				if !tt.wantErr {
					test := &RedisTest{ID: tt.args.model.ID}

					tt.query.Find(context.Background(), test)

					value := reflect.ValueOf(test).Elem()

					field := value.FieldByName(tt.args.fieldName)

					if !assert.Equal(t, field.Interface(), tt.args.v) {
						t.Errorf("Query.Update() error = %v, wantErr %v", field.Interface(), tt.args.v)
					}
				}

			}
		})
	}
}

func TestQuery_Updates(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *RedisTest
		data  map[string]interface{}
	}
	tests := []struct {
		name    string
		query   *Query
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name:  "test",
			query: redisClient.NewQuery(),
			args: args{
				ctx:   context.Background(),
				model: &RedisTest{ID: "try13"},
				data: map[string]interface{}{
					"HAHA":    true,
					"TEST":    4,
					"PayLoad": "ddddddddddd",
					"SLICE":   []string{"ddddd", "eeeee", "mmmm"},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.query.Updates(tt.args.ctx, tt.args.model, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("Query.Updates() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				test := &RedisTest{ID: tt.args.model.ID}

				tt.query.Find(context.Background(), test)

				value := reflect.ValueOf(test).Elem()

				for fieldName, valueNeed := range tt.args.data {
					field := value.FieldByName(fieldName)
					if !assert.Equal(t, field.Interface(), valueNeed) {
						t.Errorf("Query.Update() error = %v, wantErr %v", field.Interface(), valueNeed)
					}
				}

			}
		})
	}
}

func TestBFRRedis_Find(t *testing.T) {
	type args struct {
		v     *RedisTest
		query *Query
	}

	data := &RedisTest{ID: "try12"}
	tests := []struct {
		name    string
		redis   *BFRRedis
		args    args
		wantErr bool
	}{
		{
			name:  "simple Test",
			redis: redisClient,
			args: args{
				v:     data,
				query: redisClient.NewQuery().SubModel(false),
			},
			wantErr: false,
		},
		{
			name:  "simple Test.needLoadAssociation test",
			redis: redisClient,
			args: args{
				v:     &RedisTest{ID: "try13"},
				query: redisClient.NewQuery().SubModel(false),
			},
			wantErr: false,
		},
		{
			name:  "data bit found test",
			redis: redisClient,
			args: args{
				v:     &RedisTest{ID: "try14"},
				query: redisClient.NewQuery().SubModel(true),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.args.query.Find(context.Background(), tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("BFRRedis.Find() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				if err != nil {
					fmt.Println(tt.args.v)
				}
			}
		})
	}
}

func TestBFRRedis_FindSlice(t *testing.T) {
	type args struct {
		v     []RedisTest
		query *Query
	}

	tests := []struct {
		name    string
		redis   *BFRRedis
		args    args
		wantErr bool
	}{
		{
			name:  "find slice data",
			redis: redisClient,
			args: args{
				v:     []RedisTest{},
				query: redisClient.NewQuery().SubModel(true).Where("*try*"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.args.query.Find(context.Background(), &tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("BFRRedis.Find() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				assert.Equal(t, 2, len(tt.args.v))
			}
		})
	}
}

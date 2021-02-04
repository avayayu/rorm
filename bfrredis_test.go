package rorm

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	ztime "github.com/avayayu/micro/time"
)

type RedisTest struct {
	ID         string `redis:"primary"`
	TEST       int
	TEST2      float64
	TEST3      uint
	PayLoad    string
	HAHA       bool
	BYTES      byte
	PTRTEST    *string
	SLICE      []string
	TESTID     string
	TIMETEST   ztime.Time
	TestStruct *TestStruct `redis:"foreignKey:TESTID"`
}

var redisClient *BFRRedis

type TestStruct struct {
	TEST1  string `redis:"primary"`
	Haha   bool
	SLICE2 []string
	MAP    map[string]string
}

func (r *TestStruct) Key() string {
	return "testInner"
}

func (r *RedisTest) Key() string {
	return "test"
}

func TestMain(m *testing.M) {

	redisClient = NewBFRRedis(NewDefaultOptions(), nil)
	
	flag.Parse()
	exitCode := m.Run()

	redisClient = nil

	

	// 退出
	os.Exit(exitCode)
}

func TestBFRRedis_GetPrimaryKey(t *testing.T) {
	type args struct {
		v *RedisTest
	}

	haha := "ptrtest"
	tests := []struct {
		name        string
		redis       *BFRRedis
		args        args
		wantFullKey string
	}{
		// TODO: Add test cases.
		{
			name:  "simple Test",
			redis: redisClient,
			args: args{
				v: &RedisTest{
					ID:       "TEST",
					TEST:     1,
					TEST2:    1.1,
					TEST3:    1,
					PayLoad:  "asdasdasdasd",
					PTRTEST:  &haha,
					TIMETEST: ztime.Now(),
				},
			},
			wantFullKey: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// if gotFullKey, err := tt.redis.GetPrimaryKey(tt.args.v); gotFullKey != tt.wantFullKey || err != nil {
			// 	t.Errorf("BFRRedis.GetPrimaryKey() = %v, want %v", gotFullKey, tt.wantFullKey)
			// }
		})
	}
}

func TestBFRRedis_Create(t *testing.T) {
	type args struct {
		v     *RedisTest
		query *Query
	}

	haha := "ptrtest"

	tests := []struct {
		name        string
		redis       *BFRRedis
		args        args
		wantFullKey string
	}{
		// TODO: Add test cases.
		{
			name:  "simple Test",
			redis: redisClient,
			args: args{
				v: &RedisTest{
					ID:       "try15",
					TEST:     1,
					TEST2:    4.1,
					TEST3:    55,
					PayLoad:  "asdasdasdasd",
					HAHA:     true,
					BYTES:    1,
					PTRTEST:  &haha,
					TIMETEST: ztime.Now(),
					SLICE:    []string{"asd", "haha", "ddddd"},
					TESTID:   "inner3",
					TestStruct: &TestStruct{
						TEST1:  "inner3",
						Haha:   false,
						SLICE2: []string{"asd", "haha", "ddddd"},
						MAP: map[string]string{
							"aa": "dd",
							"zz": "zz",
						},
					},
				},
				query: redisClient.NewQuery(),
			},
			wantFullKey: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.args.query.Create(context.Background(), tt.args.v); err != nil {
				fmt.Println(err)
			}
		})
	}
}

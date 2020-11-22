package rorm

import (
	"fmt"
	"testing"
)

type redisTest struct {
	ID         string `redis:"primary"`
	TEST       int
	TEST2      float64
	TEST3      uint
	PayLoad    string
	HAHA       bool
	BYTES      byte
	PTRTEST    *string
	SLICE      []string
	TestStruct *TestStruct
}

type TestStruct struct {
	TEST1  string `redis:"primary"`
	Haha   bool
	SLICE2 []string
	MAP    map[string]string
}

func (r *TestStruct) Key() string {
	return "testInner"
}

func (r *redisTest) Key() string {
	return "test"
}

func TestBFRRedis_GetPrimaryKey(t *testing.T) {
	type args struct {
		v *redisTest
	}

	redis := NewBFRRedis(NewDefaultOptions(), nil)
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
			redis: redis,
			args: args{
				v: &redisTest{
					ID:      "TEST",
					TEST:    1,
					TEST2:   1.1,
					TEST3:   1,
					PayLoad: "asdasdasdasd",
					PTRTEST: &haha,
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

func TestBFRRedis_SaveSimpleStructObject(t *testing.T) {
	type args struct {
		v *redisTest
	}

	haha := "ptrtest"

	redis := NewBFRRedis(NewDefaultOptions(), nil)
	tests := []struct {
		name        string
		redis       *BFRRedis
		args        args
		wantFullKey string
	}{
		// TODO: Add test cases.
		{
			name:  "simple Test",
			redis: redis,
			args: args{
				v: &redisTest{
					ID:      "try11",
					TEST:    1,
					TEST2:   4.1,
					TEST3:   55,
					PayLoad: "asdasdasdasd",
					HAHA:    true,
					BYTES:   1,
					PTRTEST: &haha,
					SLICE:   []string{"asd", "haha", "ddddd"},
					TestStruct: &TestStruct{
						TEST1:  "inner",
						Haha:   false,
						SLICE2: []string{"asd", "haha", "ddddd"},
						MAP: map[string]string{
							"aa": "dd",
							"zz": "zz",
						},
					},
				},
			},
			wantFullKey: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.redis.SaveSimpleStructObject(tt.args.v); err != nil {
				fmt.Println(err)
			}
		})
	}
}

func TestBFRRedis_RetrieveData(t *testing.T) {
	type args struct {
		v *redisTest
	}
	redis := NewBFRRedis(NewDefaultOptions(), nil)
	data := &redisTest{ID: "try11"}
	tests := []struct {
		name    string
		redis   *BFRRedis
		args    args
		wantErr bool
	}{
		{
			name:  "simple Test",
			redis: redis,
			args: args{
				v: data,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.redis.RetrieveData(tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("BFRRedis.RetrieveData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

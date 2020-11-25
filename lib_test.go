package rorm

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestConvertStructToMap(t *testing.T) {
	haha := "asdasd"

	slice, _ := json.Marshal([]string{"asd", "haha", "ddddd"})

	mapData := map[string]string{
		"ID":      "try13",
		"TEST":    "1",
		"TEST2":   "4.1",
		"TEST3":   "55",
		"PayLoad": "asdasdasdasd",
		"HAHA":    "1",
		"PTRTEST": "asdasd",
		"SLICE":   string(slice),
		"TESTID":  "inner2",
		"BYTES":   "1",
	}
	data := &RedisTest{
		ID:      "try13",
		TEST:    1,
		TEST2:   4.1,
		TEST3:   55,
		PayLoad: "asdasdasdasd",
		HAHA:    true,
		BYTES:   1,
		PTRTEST: &haha,
		SLICE:   []string{"asd", "haha", "ddddd"},
		TESTID:  "inner2",
		TestStruct: &TestStruct{
			TEST1:  "inner2",
			Haha:   false,
			SLICE2: []string{"asd", "haha", "ddddd"},
			MAP: map[string]string{
				"aa": "dd",
				"zz": "zz",
			},
		},
	}

	type args struct {
		v interface{}
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		// TODO: Add test cases.
		{
			name: "test 1",
			args: args{
				v: data,
			},
			want: mapData,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertStructToMap(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertStructToMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

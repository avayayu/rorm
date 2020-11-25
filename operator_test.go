package rorm

import (
	"reflect"
	"testing"
)

func TestQuery_scanPatternKeys(t *testing.T) {
	type args struct {
		pattern string
	}
	tests := []struct {
		name    string
		r       *Query
		args    args
		want    []string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "try keys test",
			r:    redisClient.NewQuery().Where("*try*"),
			args: args{
				pattern: "*try*",
			},
			want:    []string{"rorm.avayayu.com/RedisTest/ID/try13", "rorm.avayayu.com/RedisTest/ID/try12"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.scanPatternKeys(tt.args.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("Query.scanPatternKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Query.scanPatternKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

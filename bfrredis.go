package rorm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	redis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

//封装go-redis 因为go-redis支持redis集群
//尽量提供复杂结构体的存储
//尽量提供redis存储的结构体的单个字段的更新

type RedisClient interface {
	Close() error
	Get(context.Context, string) *redis.StringCmd
	Pipeline() redis.Pipeliner
	HGetAll(context.Context, string) *redis.StringStringMapCmd
	SetNX(context.Context, string, interface{}, time.Duration) *redis.BoolCmd

	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ScriptExists(ctx context.Context, hashes ...string) *redis.BoolSliceCmd
	ScriptLoad(ctx context.Context, script string) *redis.StringCmd
}

type BFRRedis struct {
	Client RedisClient //单个node
	logger *zap.Logger
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
		bredis.Client = client
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
		bredis.Client = client
	}
	return bredis
}

func (r *BFRRedis) SaveSimpleStructObject(v interface{}, options ...interface{}) (err error) {
	pipe := r.Client.Pipeline()

	key, err := r.getPrimaryKey(v)

	if err != nil {
		return
	}
	typ := reflect.TypeOf(v)
	val := reflect.ValueOf(v).Elem()
	num := val.NumField()

	ctx := context.Background()

	for i := 0; i < num; i++ {
		fmt.Printf("Field %d:值=%v\n", i, val.Field(i))
		//获取到struct标签，需要通过reflect.Type来获取tag标签的值
		fieldName := typ.Elem().Field(i).Name

		//如果该字段有tag标签就显示，否则就不显示
		field := val.Field(i)
		if field.Kind() == reflect.Ptr {
			field = field.Elem()
		}
		Value := field.Interface()
		switch field.Kind() {
		// case reflect.Struct:
		// 	structValue2 := val.Field(i).Interface()
		// 	r.SaveSimpleStructObject(structValue2)
		case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct:
			json, _ := json.Marshal(Value)
			pipe.HSet(ctx, key, fieldName, string(json))
		default:
			pipe.HSet(ctx, key, fieldName, Value)
		}

	}

	pipe.Expire(ctx, key, time.Hour)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return err
	}
	return

}

//GetPrimaryKey 得到类型名 + primary key
func (r *BFRRedis) getPrimaryKey(v interface{}) (fullKey string, err error) {

	if v == nil {
		err = errors.New("v is nil")
		return
	}

	typ := reflect.TypeOf(v)
	if typ.Kind() != reflect.Ptr {
		err = errors.New("v must be a ptr")
		return
	}

	if typ.Elem().Kind() != reflect.Struct {
		err = errors.New("expect struct")
		return
	}
	val := reflect.ValueOf(v).Elem()

	num := val.NumField()

	//遍历结构体的所有字段
	for i := 0; i < num; i++ {
		fmt.Printf("Field %d:值=%v\n", i, val.Field(i))
		//获取到struct标签，需要通过reflect.Type来获取tag标签的值
		tagVal := typ.Elem().Field(i).Tag.Get("redis")
		//如果该字段有tag标签就显示，否则就不显示
		if tagVal != "" && strings.Contains(tagVal, "primary") {
			fullKey = fullKey + "/" + typ.Elem().Field(i).Name + "/" + fmt.Sprintf("%v", val.Field(i).Interface())
		}
	}
	if fullKey == "" {
		err = errors.New("primary key not found")
		return
	}
	fullKey = fmt.Sprintf("%s%v", GetTypeFullName(v), fullKey)
	return

}

func (r *BFRRedis) RetrieveData(v interface{}) (err error) {
	if v == nil {
		err = errors.New("v can not be nil")
		return
	}

	if reflect.TypeOf(v).Kind() != reflect.Ptr {
		err = errors.New("v must be ptr")
		return
	}

	if reflect.TypeOf(v).Elem().Kind() != reflect.Struct {
		err = errors.New("v must be a ptr of struct")
		return
	}

	key, err := r.getPrimaryKey(v)
	if err != nil {
		return
	}

	data, err := r.Client.HGetAll(context.Background(), key).Result()
	if err != nil {
		return
	}

	val := reflect.ValueOf(v).Elem()
	typ := reflect.TypeOf(v)

	num := val.NumField()

	for i := 0; i < num; i++ {
		key := typ.Elem().Field(i).Name
		value, _ := data[key]

		field := val.FieldByName(key)

		fmt.Println("key is " + key)

		if !field.IsValid() {
			err = errors.New("field is not valid")
			return
		}
		if !field.CanSet() {
			err = errors.New("field can not be set")
			return
		}

		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
			innerValue, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			field.SetInt(int64(innerValue))
		case reflect.Float32, reflect.Float64:
			innerValue, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return err
			}
			field.SetFloat(innerValue)
		case reflect.String:
			field.SetString(value)
		case reflect.Bool:
			if value == "1" {
				field.SetBool(true)
			} else {
				field.SetBool(false)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
			innerValue, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}
			field.SetUint(innerValue)

		case reflect.Slice, reflect.Array:
			elemType := field.Type()
			slice := reflect.MakeSlice(elemType, 1, 1)
			// Create a pointer to a slice value and set it to the slice
			slicePointer := reflect.New(slice.Type()).Interface()
			err = json.Unmarshal([]byte(value), slicePointer)
			if err != nil {
				fmt.Println(err)
				return err
			}
			if field.Kind() == reflect.Ptr {
				field.Set(reflect.ValueOf(slicePointer))
			} else {
				field.Set(reflect.ValueOf(slicePointer).Elem())
			}

		case reflect.Map, reflect.Struct:
			elemType := reflect.TypeOf(field.Elem())
			mapType := reflect.MakeMap(elemType).Addr().Interface()

			err = json.Unmarshal([]byte(value), mapType)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(mapType))
		// case reflect.Struct:
		// 	//RedisObject类型
		// 	ptr := reflect.New(reflect.TypeOf(field))
		// 	r.RetrieveData(ptr.Interface())
		// 	field.Set(ptr.Elem())
		case reflect.Ptr:
			typ := reflect.TypeOf(field.Interface())
			ptr := reflect.New(typ.Elem()).Interface()
			fmt.Println(ptr)
			r.reflectData(value, ptr)

			// fmt.Println(ptr.String())
			field.Set(reflect.ValueOf(ptr))
		default:
			fmt.Println("not get the type. key is  " + key)
			fmt.Println("field type kind is:" + field.Kind().String())
			fmt.Println("field elem  is:" + field.Elem().String())
			fmt.Println("field elem type kind is:" + field.Elem().Kind().String())

		}

	}

	return
}

func (r *BFRRedis) reflectData(data string, v interface{}) (err error) {

	if reflect.ValueOf(v).Kind() != reflect.Ptr {
		err = errors.New("not a ptr")
		return
	}

	// value := reflect.ValueOf(v)

	switch reflect.TypeOf(v).Elem().Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Struct:
		err = json.Unmarshal([]byte(data), v)
		if err != nil {
			return err
		}
	// case reflect.Struct:

	// 	//RedisObject类型
	// 	r.RetrieveData(v)
	default:
		reflect.ValueOf(v).Elem().Set(reflect.ValueOf(data))
	}
	return

}

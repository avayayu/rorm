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

type Query struct {
	// where       string
	// value       interface{}
	Pattern       string //支持正则查询Key
	Association   bool
	SelectValues  []string
	ExpireTime    time.Duration
	logger        *zap.Logger
	client        Redisclient
	AutomaticLoad bool
}

func (r *BFRRedis) NewQuery() *Query {
	return &Query{
		client:       r.client,
		logger:       r.logger,
		SelectValues: []string{},
	}
}

func (query *Query) Where(pattern string) *Query {
	query.Pattern = pattern
	return query
}

func (query *Query) SubModel(flag bool) *Query {
	query.Association = flag
	return query
}

func (query *Query) Select(field ...string) *Query {
	query.SelectValues = append(query.SelectValues, field...)
	return query
}

func (query *Query) Expire(d int64) *Query {
	query.ExpireTime = time.Duration(d)
	return query
}

func (query *Query) AutoLoad(flag bool) *Query {
	query.AutomaticLoad = flag
	return query
}

func (query *Query) pipeHSet(ctx context.Context, pipe redis.Pipeliner, key string, fieldName string, field reflect.Value) (err error) {
	switch field.Kind() {
	case reflect.Ptr:
		switch field.Elem().Kind() {
		case reflect.Struct:
			if query.Association {
				if err = query.Create(ctx, (&field).Interface()); err != nil {
					return err
				}
			}
		case reflect.Ptr:
			query.logger.Warn("data use Ptr of Ptr.Give Up Save", zap.String("field", fieldName))
		case reflect.Array, reflect.Slice, reflect.Map:
			json, _ := json.Marshal(field.Interface())
			cmd := pipe.HSet(ctx, key, fieldName, string(json))
			if err = cmd.Err(); err != nil {
				return err
			}
		default:
			cmd := pipe.HSet(ctx, key, fieldName, field.Elem().Interface())
			if err = cmd.Err(); err != nil {
				return err
			}
		}
	case reflect.Struct: //对于结构体 将直接作为一个单独的HMAP存储
		structValue2 := field.Interface()
		if query.Association {
			if err := query.Create(ctx, structValue2); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array, reflect.Map: //对于Slice Array Map三种类型 直接存储其对应的JSON
		json, _ := json.Marshal(field.Interface())
		cmd := pipe.HSet(ctx, key, fieldName, string(json))
		if err = cmd.Err(); err != nil {
			return err
		}
	default:
		cmd := pipe.HSet(ctx, key, fieldName, field.Interface())
		if err = cmd.Err(); err != nil {
			return err
		}
	}
	return
}

//GetPrimaryKey 得到类型名 + primary key
func (r *Query) getPrimaryKey(v interface{}) (fullKey string, err error) {

	if v == nil {
		err = errors.New("v is nil")
		return
	}

	typ := reflect.TypeOf(v)
	if typ.Kind() != reflect.Ptr {
		err = RormPTRNeed
		return
	}

	if typ.Elem().Kind() != reflect.Struct {
		err = RormModelMustBeStruct
		return
	}
	val := reflect.ValueOf(v).Elem()

	num := val.NumField()

	//遍历结构体的所有字段
	for i := 0; i < num; i++ {
		//获取到struct标签，需要通过reflect.Type来获取tag标签的值
		tagVal := typ.Elem().Field(i).Tag.Get("redis")
		//如果该字段有tag标签就显示，否则就不显示
		if tagVal != "" && strings.Contains(tagVal, "primary") {
			fullKey = fullKey + "/" + typ.Elem().Field(i).Name + "/" + fmt.Sprintf("%v", val.Field(i).Interface())
		}
	}
	if fullKey == "" {
		err = RormPrimaryKeyNotFound
		return
	}
	fullKey = fmt.Sprintf("%s%v", GetTypeFullName(v), fullKey)
	return
}

func (r *Query) getPrimaryKeyWithNoTags(v interface{}, keySuffix string) string {
	fullKey := fmt.Sprintf("%s/%v", GetTypeFullName(v), keySuffix)
	return fullKey
}

func (r *Query) scanPatternKeys(pattern string) ([]string, error) {
	var cursor uint64
	var n int
	var keys []string
	var err error
	for {
		keys, cursor, err = r.client.Scan(context.Background(), cursor, pattern, 10).Result()
		if err != nil {
			panic(err)
		}
		n += len(keys)
		if cursor == 0 {
			break
		}
	}
	return keys, err
}

func (query *Query) getDataFromRedis(keys ...string) (map[string]map[string]string, error) {
	mapData := make(map[string]map[string]string)
	pipe := query.client.Pipeline()
	ctx := context.Background()

	var result []*redis.StringStringMapCmd

	for _, key := range keys {
		ssmCmd := pipe.HGetAll(ctx, key)
		result = append(result, ssmCmd)
	}

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	for _, cmd := range cmds {
		if ssmCmd, ok := cmd.(*redis.StringStringMapCmd); ok {
			if _, err := ssmCmd.Result(); err != nil {
				return nil, err
			}

			args := ssmCmd.Args()
			data, err := ssmCmd.Result()
			if err != nil {
				return nil, err
			}
			if len(args) < 2 {
				return nil, errors.New("no data")
			}
			mapData[args[1].(string)] = data
		}
	}
	return mapData, nil
}

func (query *Query) fetchData(ctx context.Context, v interface{}) (data map[string]string, err error) {
	if v == nil {
		err = RormPTRNeed
		return
	}

	if reflect.TypeOf(v).Kind() != reflect.Ptr {
		err = RormPTRNeed
		return
	}

	if reflect.TypeOf(v).Elem().Kind() != reflect.Struct {
		err = RormModelMustBeStruct
		return
	}

	key, err := query.getPrimaryKey(v)
	if err != nil {
		return
	}
	if len(query.SelectValues) == 0 {
		data, err = query.client.HGetAll(ctx, key).Result()
	} else {
		for _, field := range query.SelectValues {
			subdata, err := query.client.HGet(ctx, key, field).Result()
			if err != nil {
				return nil, err
			}
			data[field] = subdata
		}

	}

	if len(data) == 0 {
		if err == nil {
			//redis没有给err 但是data长度为0 此时则认为redis里没有数据
			err = RormDataNotFound
		}
		if query.AutomaticLoad {
			if loader, ok := v.(RormLoader); ok {
				if err = loader.Loader(v); err != nil {
					return
				}
				go query.Create(ctx, v)
				data = ConvertStructToMap(v)
				return data, nil
			}
		}
		return
	}
	return
}

func (query *Query) retrieveData(data map[string]string, v interface{}) (err error) {

	val := reflect.ValueOf(v).Elem()
	typ := reflect.TypeOf(v)

	num := val.NumField()

	for i := 0; i < num; i++ {
		key := typ.Elem().Field(i).Name
		value, ok := data[key]

		if !ok {
			continue
		}

		field := val.FieldByName(key)

		if !field.IsValid() {
			err = errors.New("field is not valid")
			return
		}
		if !field.CanSet() {
			err = errors.New("field can not be set")
			return
		}

		switch field.Kind() {
		case reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Func, reflect.Invalid, reflect.UnsafePointer:
			query.logger.Warn("this type can not be reflect", zap.String("fieldName", key))
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
				return err
			}
			if field.Kind() == reflect.Ptr {
				field.Set(reflect.ValueOf(slicePointer))
			} else {
				field.Set(reflect.ValueOf(slicePointer).Elem())
			}
		case reflect.Map:
			elemType := field.Type()
			mapData := reflect.MakeMap(elemType)
			mapPointer := reflect.New(mapData.Type()).Interface()
			err = json.Unmarshal([]byte(value), mapPointer)
			if err != nil {
				return err
			}
			if field.Kind() == reflect.Ptr {
				field.Set(reflect.ValueOf(mapPointer))
			} else {
				field.Set(reflect.ValueOf(mapPointer).Elem())
			}
		case reflect.Struct:
			//pass struct.
			//deal it in function loadForeignModel
		case reflect.Ptr:
			typ := reflect.TypeOf(field.Interface())
			ptr := reflect.New(typ.Elem()).Interface()
			err = query.PtrData(value, ptr)
			if err != nil {
				return err
			}
			// fmt.Println(ptr.String())
			field.Set(reflect.ValueOf(ptr))
		default:
			fmt.Println("not get the type. key is  " + key)
			fmt.Println("field type kind is:" + field.Kind().String())
			fmt.Println("field elem  is:" + field.Elem().String())
			fmt.Println("field elem type kind is:" + field.Elem().Kind().String())

		}
	}

	//加载关联struct
	if query.Association {
		if err = query.loadForeignModel(v, data); err != nil {
			return err
		}
	}
	return
}

func (query *Query) loadForeignModel(v interface{}, data map[string]string) (err error) {
	val := reflect.ValueOf(v).Elem()
	typ := reflect.TypeOf(v)

	num := val.NumField()

	for i := 0; i < num; i++ {

		fieldType := typ.Elem().Field(i)
		key := fieldType.Name
		field := val.FieldByName(key)

		if !field.IsValid() {
			err = errors.New("field is not valid")
			return
		}
		if !field.CanSet() {
			err = errors.New("field can not be set")
			return
		}

		var foreignTags string

		switch field.Kind() {
		case reflect.Struct:
			foreignTags = fieldType.Tag.Get("redis")
		case reflect.Ptr:
			if field.Elem().Kind() == reflect.Struct {
				foreignTags = fieldType.Tag.Get("redis")
			} else {
				continue
			}
		default:
			continue
		}

		var foreignFieldName string
		if foreignTags != "" && strings.Contains(foreignTags, "foreignKey") {
			arrs := strings.Split(foreignTags, ";")
			for _, arr := range arrs {
				if strings.Contains(arr, "foreignKey") {
					foreignKeys := strings.Split(arr, ":")
					if len(foreignKeys) > 1 {
						foreignFieldName = foreignKeys[1]
					}
				}
			}

			if foreignFieldName != "" {
				structType := fieldType.Type
				if fieldType.Type.Kind() == reflect.Ptr {
					structType = structType.Elem()
				}
				structPtr := reflect.New(structType).Elem()

				primaryField, err := query.getRedisPrimaryField(structPtr.Interface())
				if err != nil {
					query.logger.Error("找到主键对应的field出错")
					continue
				}
				valueOfStruct := structPtr.FieldByName(primaryField.Name)
				foreignKeyValue := val.FieldByName(foreignFieldName)
				valueOfStruct.Set(foreignKeyValue)
				ptrToStruct := structPtr.Addr().Interface()
				err = query.retrieveData(data, ptrToStruct)
				if err != nil {
					return err
				}

				field.Set(reflect.ValueOf(ptrToStruct))
			} else {
				query.logger.Warn("a struct exist,but no foreignKey found", zap.String("fieldName", key))
			}
		}
	}
	return nil
}

func (r *Query) getRedisPrimaryField(v interface{}) (field reflect.StructField, err error) {
	typ := reflect.TypeOf(v)
	val := reflect.ValueOf(v)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		val = val.Elem()
	}

	if typ.Kind() != reflect.Struct {
		err = errors.New("v is not a struct")
		return
	}

	num := val.NumField()

	//遍历结构体的所有字段
	for i := 0; i < num; i++ {
		fmt.Printf("Field %d:值=%v\n", i, val.Field(i))
		//获取到struct标签，需要通过reflect.Type来获取tag标签的值
		tagVal := typ.Field(i).Tag.Get("redis")
		//如果该字段有tag标签就显示，否则就不显示
		if tagVal != "" && strings.Contains(tagVal, "primary") {
			return typ.Field(i), nil
		}
	}
	err = errors.New("not found field with tag redis primary")
	return
}

func (r *Query) PtrData(data string, v interface{}) (err error) {

	if reflect.ValueOf(v).Kind() != reflect.Ptr {
		err = errors.New("not a ptr")
		return
	}

	switch reflect.TypeOf(v).Elem().Kind() {
	case reflect.Array, reflect.Slice, reflect.Map:
		err = json.Unmarshal([]byte(data), v)
		if err != nil {
			return err
		}
	case reflect.Struct:

	case reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Func, reflect.Invalid, reflect.UnsafePointer:
		r.logger.Warn("this type can not be reflect", zap.String("fieldName", reflect.TypeOf(v).Name()), zap.String("value", data))
	case reflect.Ptr:
		r.logger.Warn("pointer of pointer,that's not allowed", zap.String("fieldName", reflect.TypeOf(v).Name()))
	default:
		reflect.ValueOf(v).Elem().Set(reflect.ValueOf(data))
	}
	return

}

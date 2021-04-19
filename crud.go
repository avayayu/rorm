package rorm

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

func (query *Query) Create(ctx context.Context, v interface{}) (err error) {
	pipe := query.client.Pipeline()

	key, err := query.getPrimaryKey(v)

	if err != nil {
		return
	}
	typ := reflect.TypeOf(v)
	val := reflect.ValueOf(v).Elem()
	num := val.NumField()

	for i := 0; i < num; i++ {
		fmt.Printf("Field %d:值=%v\n", i, val.Field(i))
		//获取到struct标签，需要通过reflect.Type来获取tag标签的值
		fieldName := typ.Elem().Field(i).Name
		redisTag := typ.Elem().Field(i).Tag.Get("redis")

		if redisTag == "-" {
			continue
		}

		field := val.Field(i)

		// //如果field的类型为指针，则一直取指针指到不为指针为止
		// Value := field.Interface()
		err = query.pipeHSet(ctx, pipe, key, fieldName, field)
		if err != nil {
			return err
		}
	}
	if query.ExpireTime > 0 {
		_, err = pipe.Expire(ctx, key, query.ExpireTime).Result()
		if err != nil {
			return err
		}
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return err
	}
	return

}

func (query *Query) Find(ctx context.Context, v interface{}) (err error) {

	if reflect.TypeOf(v).Kind() != reflect.Ptr {
		err = errors.New("v must be ptr")
		return
	}

	switch reflect.TypeOf(v).Elem().Kind() {
	case reflect.Struct:
		//直接找到主键对应的的数据项
		data, err := query.fetchData(ctx, v)
		if err != nil {
			return err
		}
		return query.retrieveData(data, v)

	case reflect.Slice:
		if query.Pattern == "" {
			return errors.New(`Query Pattern can not be ""`)
		}
		keys, err := query.scanPatternKeys(query.Pattern)
		if err != nil {
			return err
		}

		mapData, err := query.getDataFromRedis(keys...)
		// reflect.AppendSlice(s reflect.Value, t reflect.Value)
		elementTyp := reflect.TypeOf(v).Elem().Elem()
		value := reflect.ValueOf(v).Elem()
		fmt.Println(value)
		var realTyp reflect.Type = elementTyp
		if elementTyp.Kind() == reflect.Ptr {
			realTyp = elementTyp.Elem()
		}

		for _, mapdata := range mapData {
			element := reflect.New(realTyp)

			err = query.retrieveData(mapdata, element.Interface())
			if err != nil {
				return err
			}
			if realTyp.Kind() == reflect.Ptr {
				value = reflect.Append(value, element)
			} else {
				value = reflect.Append(value, element.Elem())
			}
		}

		reflect.ValueOf(v).Elem().Set(value)

	}
	return
}

func (query *Query) Update(ctx context.Context, model interface{}, fieldName string, v interface{}) (err error) {
	hashKey, err := query.getPrimaryKey(model)
	if err != nil {
		return err
	}

	length, err := query.client.Exists(ctx, hashKey).Result()
	if err != nil || length == 0 {
		err = RormPrimaryKeyNotFound
		return
	}

	line := query.client.Pipeline()

	typ := reflect.TypeOf(model)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		err = RormModelMustBeStruct
		return
	}

	if _, ok := typ.FieldByName(fieldName); !ok {
		err = RormFieldNotExist
		return
	}

	err = query.pipeHSet(ctx, line, hashKey, fieldName, reflect.ValueOf(v))
	if err != nil {
		return err
	}

	if _, err = line.Exec(ctx); err != nil {
		return
	}
	return
}

func (query *Query) Updates(ctx context.Context, model interface{}, data map[string]interface{}) (err error) {
	hashKey, err := query.getPrimaryKey(model)
	if err != nil {
		return err
	}
	datatemp := map[string]string{}
	if datatemp, err = query.client.HGetAll(ctx, hashKey).Result(); len(datatemp) == 0 || err != nil {
		err = RormPrimaryKeyNotFound
		return
	}

	line := query.client.Pipeline()

	typ := reflect.TypeOf(model)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	for key, value := range data {

		if _, ok := typ.FieldByName(key); !ok {
			err = RormFieldNotExist
			return
		}

		err = query.pipeHSet(ctx, line, hashKey, key, reflect.ValueOf(value))
		if err != nil {
			return err
		}
	}
	if _, err = line.Exec(ctx); err != nil {
		return
	}
	return
}

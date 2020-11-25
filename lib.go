package rorm

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"time"
	"unsafe"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func RandStringBytesMaskImprSrcUnsafe(n int) string {

	var src = rand.NewSource(time.Now().UnixNano())
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}

func GetTypeFullName(myvar interface{}) string {
	if t := reflect.TypeOf(myvar); t.Kind() == reflect.Ptr {
		return t.Elem().PkgPath() + "/" + t.Elem().Name()
	} else {
		return t.PkgPath() + "/" + t.Name()
	}
}

func ConvertStructToMap(v interface{}) map[string]string {
	data := make(map[string]string)

	val := reflect.ValueOf(v).Elem()
	typ := reflect.TypeOf(v)

	num := val.NumField()

	for i := 0; i < num; i++ {
		key := typ.Elem().Field(i).Name
		field := val.FieldByName(key)
		Value := field.Interface()
		switch field.Kind() {
		case reflect.Ptr:
			switch field.Elem().Kind() {
			case reflect.Struct:
			case reflect.Ptr:
			case reflect.Bool:
				if field.Interface().(bool) {
					data[key] = "1"
				} else {
					data[key] = "0"
				}
			case reflect.Array, reflect.Slice, reflect.Map:
				json, _ := json.Marshal(Value)
				data[key] = string(json)
			default:
				data[key] = fmt.Sprintf("%v", field.Elem().Interface())
			}
		case reflect.Struct: //对于结构体 将直接作为一个单独的HMAP存储
		case reflect.Slice, reflect.Array, reflect.Map: //对于Slice Array Map三种类型 直接存储其对应的JSON
			json, _ := json.Marshal(Value)
			data[key] = string(json)
		case reflect.Bool:
			if field.Interface().(bool) {
				data[key] = "1"
			} else {
				data[key] = "0"
			}
		default:
			data[key] = fmt.Sprintf("%v", field.Interface())
		}
	}
	return data
}

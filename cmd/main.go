package main

import (
	"fmt"
	"reflect"
)

type Test struct {
	A string
	B string
}

func main() {
	c := &Test{}
	typ := reflect.TypeOf(c)
	value := reflect.New(typ.Elem()).Elem()
	fmt.Println(value.Interface())
	aData := value.FieldByName("A")
	aData.Set(reflect.ValueOf("C"))
	fmt.Println(value.Interface())
}

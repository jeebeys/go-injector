package utils

import (
	"fmt"
	"reflect"
)

const (
	AutowireTagKey = "autowire"
)

func GetFullUniqueName(alias string, instance interface{}) string {
	t := reflect.TypeOf(instance)
	return fmt.Sprintf("%s@%p", t.String(), &instance)
}

func FieldNeedToInject(f reflect.StructField) bool {
	_, ok := f.Tag.Lookup(AutowireTagKey)
	return ok
}

func CanRegeiste(instance interface{}) bool {
	t := reflect.TypeOf(instance)
	return t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct
}

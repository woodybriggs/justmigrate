package ast

import (
	"reflect"
)

func Copy(src any) any {
	if src == nil {
		return nil
	}

	og := reflect.ValueOf(src)
	clone := reflect.New(og.Type()).Elem()
	deepCopy(og, clone)

	return clone.Interface()
}

type DeepCopiable interface {
	DeepCopy() any
}

func deepCopy(src, dst reflect.Value) {
	if src.CanInterface() {
		if delegate, ok := src.Interface().(DeepCopiable); ok {
			dst.Set(reflect.ValueOf(delegate.DeepCopy()))
			return
		}
	}

	switch src.Kind() {
	case reflect.Pointer:
		originalValue := src.Elem()

		if !originalValue.IsValid() {
			return
		}
		dst.Set(reflect.New(originalValue.Type()))
		deepCopy(originalValue, dst.Elem())

	case reflect.Interface:
		if src.IsNil() {
			return
		}
		originalValue := src.Elem()

		copyValue := reflect.New(originalValue.Type()).Elem()
		deepCopy(originalValue, copyValue)
		dst.Set(copyValue)

	case reflect.Struct:
		for i := 0; i < src.NumField(); i++ {
			if src.Type().Field(i).PkgPath != "" {
				continue
			}
			deepCopy(src.Field(i), dst.Field(i))
		}

	case reflect.Slice:
		if src.IsNil() {
			return
		}
		dst.Set(reflect.MakeSlice(src.Type(), src.Len(), src.Cap()))
		for i := 0; i < src.Len(); i++ {
			deepCopy(src.Index(i), dst.Index(i))
		}

	case reflect.Map:
		if src.IsNil() {
			return
		}
		dst.Set(reflect.MakeMap(src.Type()))
		for _, key := range src.MapKeys() {
			originalValue := src.MapIndex(key)
			copyValue := reflect.New(originalValue.Type()).Elem()
			deepCopy(originalValue, copyValue)
			copyKey := Copy(key.Interface())
			dst.SetMapIndex(reflect.ValueOf(copyKey), copyValue)
		}

	default:
		dst.Set(src)
	}
}

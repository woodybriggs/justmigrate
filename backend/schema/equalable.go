package schema

import "reflect"

type Equalable interface {
	Eq(any) bool
}

func CheckPtrEq(a, b Equalable) bool {

	if a == nil && b == nil {
		return true
	}

	aIsNil := a == nil || (reflect.ValueOf(a).Kind() == reflect.Ptr && reflect.ValueOf(a).IsNil())
	bIsNil := b == nil || (reflect.ValueOf(b).Kind() == reflect.Ptr && reflect.ValueOf(b).IsNil())

	if aIsNil && bIsNil {
		return true
	}

	if aIsNil || bIsNil {
		return false
	}

	return a.Eq(b)
}

func CheckEq(a, b Equalable) bool {
	return a.Eq(b)
}

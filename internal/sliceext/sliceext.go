package sliceext

import "slices"

type Cloneable[T any] interface {
	Clone() T
}

func CloneDeep[S ~[]E, E Cloneable[T], T any](s S) []T {
	res := make([]T, len(s))

	for i, item := range s {
		res[i] = item.Clone()
	}

	return res
}

func CloneValueDeep[S ~[]E, E any, P interface {
	*E
	Clone() P
}](s S) S {
	res := make(S, len(s))
	for i := range s {
		// Use the pointer to the element to trigger the Clone method
		ptr := P(&s[i])
		res[i] = *ptr.Clone()
	}
	return res
}

func FindFunc[S ~[]E, E any](s S, find func(E) bool) (E, bool) {
	idx := slices.IndexFunc(s, find)
	if idx >= 0 {
		return s[idx], true
	}

	var nilVal E
	return nilVal, false
}

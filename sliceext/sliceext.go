package sliceext

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

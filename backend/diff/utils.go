package diff

import (
	"iter"
	"slices"
)

type pair[T any] struct {
	A T
	B T
}

type pairs[T any] []pair[T]

type Equalable interface {
	Eq(any) bool
}

func intersection[T Equalable](a, b []T) (result pairs[T]) {
	seen := make(map[int]struct{})

	result = make([]pair[T], 0, min(len(a), len(b)))

	for i, x := range a {
		if _, included := seen[i]; included {
			continue
		}

		for _, y := range b {
			if x.Eq(y) {
				result = append(result, pair[T]{
					A: x,
					B: y,
				})
				seen[i] = struct{}{}
				break
			}
		}
	}

	return result
}

func symmetricDifference[T Equalable](a, b []T) (left, right []T) {
	matchedA := make([]bool, len(a))
	matchedB := make([]bool, len(b))

	for i, x := range a {
		if matchedA[i] {
			continue
		}
		for j, y := range b {
			if matchedB[j] {
				continue
			}
			if x.Eq(y) {
				matchedA[i] = true
				matchedB[j] = true
				break
			}
		}
	}

	// Collect unmatched
	for i, x := range a {
		if !matchedA[i] {
			left = append(left, x)
		}
	}

	for j, y := range b {
		if !matchedB[j] {
			right = append(right, y)
		}
	}

	return left, right
}

func mapOver[T, U any](seq iter.Seq[T], mapfn func(T) U) iter.Seq[U] {
	return func(yield func(U) bool) {
		seq(func(v T) bool {
			return yield(mapfn(v))
		})
	}
}

type equalableDelgate[T any] struct {
	val T
	eq  func(T, any) bool
}

func (ed equalableDelgate[T]) Eq(otherAny any) bool {
	return ed.eq(ed.val, otherAny)
}

func eqWrapper[T any](predicate func(a T, b T) bool) func(a T, b any) bool {
	return func(a T, b any) bool {
		other, ok := b.(equalableDelgate[T])
		if !ok {
			return false
		}

		return predicate(a, other.val)
	}
}

func symmetricDifferenceFunc[T any](a, b []T, predicate func(a, b T) bool) ([]T, []T) {

	toDelegate := func(val T) equalableDelgate[T] {
		return equalableDelgate[T]{
			val: val,
			eq:  eqWrapper(predicate),
		}
	}

	unwrapDelegate := func(ed equalableDelgate[T]) T {
		return ed.val
	}

	aDelegates := slices.Collect(mapOver(slices.Values(a), toDelegate))
	bDelegates := slices.Collect(mapOver(slices.Values(b), toDelegate))

	l, r := symmetricDifference(aDelegates, bDelegates)

	left := slices.Collect(mapOver(slices.Values(l), unwrapDelegate))
	right := slices.Collect(mapOver(slices.Values(r), unwrapDelegate))

	return left, right
}

func intersectionFunc[T any](a, b []T, predicate func(a, b T) bool) pairs[T] {
	toDelegate := func(val T) equalableDelgate[T] {
		return equalableDelgate[T]{
			val: val,
			eq:  eqWrapper(predicate),
		}
	}

	unwrapDelegate := func(p pair[equalableDelgate[T]]) pair[T] {
		return pair[T]{
			A: p.A.val,
			B: p.B.val,
		}
	}

	aDelegates := slices.Collect(mapOver(slices.Values(a), toDelegate))
	bDelegates := slices.Collect(mapOver(slices.Values(b), toDelegate))

	intersectedPairs := intersection(aDelegates, bDelegates)

	return slices.Collect(mapOver(slices.Values(intersectedPairs), unwrapDelegate))
}

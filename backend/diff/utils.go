package diff

import "iter"

type pair[T any] struct {
	A T
	B T
}

type pairs[T any] []pair[T]

func intersection[T any](a, b []T, equal func(x, y T) bool) (result pairs[T]) {
	seen := make(map[int]struct{})

	result = make([]pair[T], 0, min(len(a), len(b)))

	for i, x := range a {
		if _, included := seen[i]; included {
			continue
		}

		for _, y := range b {
			if equal(x, y) {
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

func symmetricDifference[T any](a, b []T, predicate func(a, b T) bool) (left, right []T) {
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
			if predicate(x, y) {
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

func filter[T any](seq iter.Seq[T], predicate func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		seq(func(v T) bool {
			if !predicate(v) {
				return true // Skip this item, but tell upstream to keep going
			}
			return yield(v) // Emit item; if downstream stops, stop upstream
		})
	}
}

func filterThenMap[T any, U any](seq iter.Seq[T], filterMap func(T) (U, bool)) iter.Seq[U] {
	return func(yield func(U) bool) {
		seq(func(v T) bool {
			u, ok := filterMap(v)
			if !ok {
				return true
			}
			return yield(u)
		})
	}
}

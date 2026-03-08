package datastructures

type Queue[T any] struct {
	data  []T
	read  int
	write int
	count int
}

func NewQueue[T any](size int) *Queue[T] {
	return &Queue[T]{
		data: make([]T, size),
	}
}

func (q *Queue[T]) Enqueue(item T) {
	if q.count == len(q.data) {
		q.resize()
	}

	q.data[q.write] = item
	q.write = (q.write + 1) % len(q.data)
	q.count += 1
}

func (q *Queue[T]) Dequeue() (item T, ok bool) {
	var zero T
	if q.count == 0 {
		return zero, false
	}

	item = q.data[q.read]
	q.data[q.read] = zero

	q.read = (q.read + 1) % len(q.data)
	q.count -= 1
	return item, true
}

func (q *Queue[T]) resize() {
	newSize := len(q.data) * 2
	if newSize == 0 {
		newSize = 1
	}
	newData := make([]T, newSize)
	for i := 0; i < q.count; i++ {
		newData[i] = q.data[(q.write+i)%len(q.data)]
	}
	q.data = newData
	q.read = 0
	q.write = q.count
}

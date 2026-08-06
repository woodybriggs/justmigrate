package datastructures

type Stack[T any] struct {
	Data []T
}

func (stack *Stack[T]) Push(value T) {
	stack.Data = append(stack.Data, value)
}

func (stack *Stack[T]) Top() (T, bool) {
	var val T
	if len(stack.Data) > 0 {
		val = stack.Data[len(stack.Data)-1]
		return val, true
	}
	return val, false
}

func (stack *Stack[T]) Pop() (T, bool) {
	var val T
	if len(stack.Data) > 0 {
		val, stack.Data = stack.Data[len(stack.Data)-1], stack.Data[:len(stack.Data)-1]
		return val, true
	}
	return val, false
}

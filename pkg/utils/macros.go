package utils

import (
	"fmt"
	"golang.org/x/exp/constraints"
)

func BoolToString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func Clamp[T constraints.Integer | constraints.Float](min, value, max T) T {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func FormatASCII(ascii byte) string {
	if ascii >= 32 && ascii <= 126 {
		return string(ascii)
	}
	return "."
}

func HumanReadable[T constraints.Integer](b T) string {
	if uint64(b) < 1024 {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := 1024, 0
	for n := uint64(b) / 1024; n >= 1024; n /= 1024 {
		div *= 1024
		exp++
	}
	return fmt.Sprintf("%.0f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func RemoveIndex[T any](s []T, index int) []T {
	ret := make([]T, len(s)-1)
	copy(ret, s[:index])
	copy(ret[index:], s[index+1:])
	return ret
}

func ZeroAdjust[T constraints.Integer](v T) T {
	if v == 0 {
		return 1
	}
	return v
}

type FIFO[T any] struct {
	data           []T
	tail, head     int
	capacity, size int
}

func NewFIFO[T any](size int) *FIFO[T] {
	f := &FIFO[T]{capacity: size}
	f.data = make([]T, size)
	return f
}

func (f *FIFO[T]) Push(v T) {
	if f.size >= f.capacity {
		return // FIFO is full
	}
	f.data[f.tail] = v
	f.tail = (f.tail + 1) % f.capacity
	f.size++
}

func (f *FIFO[T]) GetIndex(i int) T {
	return f.data[(f.head+i)%f.capacity]
}

func (f *FIFO[T]) ReplaceIndex(i int, v T) {
	f.data[(f.head+i)%f.capacity] = v
}

func (f *FIFO[T]) Pop() (T, bool) {
	if f.size == 0 {
		return f.data[0], false // FIFO is empty
	}
	fe := f.data[f.head]
	f.head = (f.head + 1) % f.capacity
	f.size--

	return fe, true
}

func (f *FIFO[T]) Size() int { return f.size }
func (f *FIFO[T]) Cap() int  { return f.capacity }

func (f *FIFO[T]) Reset() { f.head, f.tail, f.size = 0, 0, 0 }

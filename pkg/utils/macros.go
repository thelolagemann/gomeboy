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

package utils

import "golang.org/x/exp/constraints"

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

func ZeroAdjust8(v uint8) uint8 {
	if v == 0 {
		return 1
	}
	return v
}

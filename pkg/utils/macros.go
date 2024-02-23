package utils

func ZeroAdjust8(v uint8) uint8 {
	if v == 0 {
		return 1
	}
	return v
}

package utils

func BoolToString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func ZeroAdjust8(v uint8) uint8 {
	if v == 0 {
		return 1
	}
	return v
}

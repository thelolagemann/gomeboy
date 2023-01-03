package utils

func SetBit(value uint8, bit uint8) uint8 {
	return value | (1 << bit)
}

func ClearBit(value uint8, bit uint8) uint8 {
	return value &^ (1 << bit)
}

// TestBit returns true if the bit is set, false otherwise.
func TestBit(value uint8, bit uint8) bool {
	return value&(1<<bit) != 0
}

// GetBit returns the value of the bit.
func GetBit(value uint8, bit uint8) uint8 {
	return (value >> bit) & 1
}

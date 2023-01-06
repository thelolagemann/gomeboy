package utils

// Val returns the value of the bit at the given index.
func Val(b uint8, i uint8) uint8 {
	return (b >> i) & 1
}

// Reset resets the bit at the given index.
func Reset(b, i uint8) uint8 {
	return b &^ (1 << i)
}

// Set sets the bit at the given index.
func Set(b, i uint8) uint8 {
	return b | (1 << i)
}

// Test tests the bit at the given index.
func Test(b, i uint8) bool {
	return (b>>i)&1 != 0
}

// HalfCarryAdd returns whether a half carry occurred when adding the given values.
func HalfCarryAdd(a, b uint8) bool {
	return (a&0xF)+(b&0xF) > 0xF
}

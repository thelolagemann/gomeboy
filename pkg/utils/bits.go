package utils

import "github.com/thelolagemann/gomeboy/internal/types"

// Val returns the value of the bit at the given index.
func Val(b uint8, i uint8) uint8 {
	return (b >> i) & 1
}

// Reset resets the bit at the given index.
func Reset(b uint8, i types.Bit) uint8 {
	return b &^ i
}

// Set sets the bit at the given index.
func Set(b, i types.Bit) uint8 {
	return b | i
}

// Test tests the bit at the given index.
func Test(b, i uint8) bool {
	return (b>>i)&1 != 0
}

package cpu

import "github.com/thelolagemann/gomeboy/internal/types"

// shiftLeftArithmetic shifts n left by one bit, and sets the carry flag to the
// most significant bit of n.
//
//	SLA n
//	n = B, C, D, E, H, L, (HL), A
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 7 data.
func (c *CPU) shiftLeftArithmetic(n uint8) uint8 {
	computed := n << 1
	c.setFlags(computed == 0, false, false, n&types.Bit7 == types.Bit7)
	return computed
}

// shiftRightArithmetic shifts n right by one bit and sets the carry flag to the
// least significant bit of n. The most significant bit does not change.
//
//	SRA n
//	n = B, C, D, E, H, L, (HL), A
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) shiftRightArithmetic(n uint8) uint8 {
	computed := n>>1 | n&types.Bit7
	c.setFlags(computed == 0, false, false, n&types.Bit0 == types.Bit0)
	return computed
}

// shiftRightLogical shifts n right one bit and sets the carry flag to the
// least significant bit of n.
//
//	SRL n
//	n = B, C, D, E, H, L, (HL), A
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) shiftRightLogical(n uint8) uint8 {
	computed := n >> 1
	c.setFlags(computed == 0, false, false, n&types.Bit0 == types.Bit0)

	return computed
}

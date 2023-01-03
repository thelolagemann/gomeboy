package cpu

// swap the upper and lower nibbles of the given Register.
//
//	SWAP n
//	n = A, B, C, D, E, H, L, (HL)
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Reset.
func (c *CPU) swap(reg *Register) {
	*reg = c.swapByte(*reg)
}

// swapByte is a helper function for that swaps the upper and lower nibbles of
// the given byte, and sets the flags accordingly.
func (c *CPU) swapByte(b uint8) uint8 {
	computed := ((b << 4) & 0xF0) | (b >> 4)
	c.shouldZeroFlag(computed)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	c.clearFlag(FlagCarry)
	return computed
}

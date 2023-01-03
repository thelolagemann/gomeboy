package cpu

// setBit sets the bit at the given position in the given value.
//
//	Bit n, r
//	n = 0-7
//	r = A, B, C, D, E, H, L, (HL)
//
// IF affected:
//
//	Z - Not affected.
//	N - Reset.
//	H - Set.
//	C - Not affected.
func (c *CPU) setBit(value uint8, position uint8) uint8 {
	return value | (1 << position)
}

// clearBit clears the bit at the given position in the given value.
//
//	Bit n, r
//	n = 0-7
//	r = A, B, C, D, E, H, L, (HL)
//
// IF affected:
//
//	Z - Not affected.
//	N - Reset.
//	H - Set.
//	C - Not affected.
func (c *CPU) clearBit(value uint8, position uint8) uint8 {
	return value &^ (1 << position)
}

// testBit tests the bit at the given position in the given Register.
//
//	Bit n, r
//	n = 0-7
//	r = A, B, C, D, E, H, L, (HL)
//
// IF affected:
//
//	Z - Set if bit n of Register r is 0.
//	N - Reset.
//	H - Set.
//	C - Not affected.
func (c *CPU) testBit(value uint8, position uint8) {
	c.shouldZeroFlag((value >> position) & 0x01)
	c.clearFlag(FlagSubtract)
	c.setFlag(FlagHalfCarry)
}

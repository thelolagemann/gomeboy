package cpu

// shiftLeftIntoCarry shifts the given value left by one bit, and sets the
// carry flag to the old bit 7 data. The most significant bit is set to 0.
//
//	SLA n
//	n = A, B, C, D, E, H, L, (HL)
//
// IF affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 7 data.
func (c *CPU) shiftLeftIntoCarry(value uint8) uint8 {
	result := value << 1
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	c.shouldZeroFlag(result)
	if value&0x80 == 0x80 {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	return result
}

// shiftRightIntoCarry shifts the given value right by one bit, and sets the
// carry flag to the old bit 0 data. The most significant bit does not change.
//
//	SRL n
//	n = A, B, C, D, E, H, L, (HL)
//
// IF affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) shiftRightIntoCarry(value uint8) uint8 {
	result := (value >> 1) | (value & 0x80)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	c.shouldZeroFlag(result)
	if value&0x01 == 0x01 {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	return result
}

// shiftRightLogical shifts the given value right by one bit, and sets the
// carry flag to the old bit 0 data. The most significant bit is set to 0.
//
//	SRL n
//	n = A, B, C, D, E, H, L, (HL)
//
// IF affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) shiftRightLogical(value uint8) uint8 {
	result := value >> 1
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	c.shouldZeroFlag(result)
	if value&0x01 == 0x01 {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	return result
}

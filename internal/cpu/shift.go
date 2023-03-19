package cpu

// shiftLeftIntoCarry shifts the given value left by one bit, and sets the
// carry flag to the old bit 7 data. The most significant bit is set to 0.
//
//	SLA n
//	n = A, B, C, D, E, H, L, (HL)
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 7 data.
func (c *CPU) shiftLeftIntoCarry(value uint8) uint8 {
	newCarry := value >> 7
	computed := (value << 1) & 0xFF
	c.setFlags(computed == 0, false, false, newCarry == 1)
	return computed
}

// shiftRightIntoCarry shifts the given value right by one bit, and sets the
// carry flag to the old bit 0 data. The most significant bit does not change.
//
//	SRA n
//	n = A, B, C, D, E, H, L, (HL)
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) shiftRightIntoCarry(value uint8) uint8 {
	result := (value >> 1) | (value & 0x80)
	c.setFlags(result == 0, false, false, value&0x01 == 0x01)
	return result
}

// shiftRightLogical shifts the given value right by one bit, and sets the
// carry flag to the old bit 0 data. The most significant bit is set to 0.
//
//	SRL n
//	n = A, B, C, D, E, H, L, (HL)
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) shiftRightLogical(value uint8) uint8 {
	newCarry := value & 0x1
	computed := value >> 1
	c.setFlags(computed == 0, false, false, newCarry == 1)

	return computed
}

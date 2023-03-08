package cpu

// rotateLeft rotates the given value left by 1 bit. Bit 7 is copied to both
// the carry flag and the least significant bit.
//
//	RLC n
//	n = A, B, C, D, E, H, L, (HL)
//
// Flags affected:
//
//	Z - Set if Result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 7 data.
func (c *CPU) rotateLeft(value uint8) uint8 {
	carry := value >> 7
	rotated := (value<<1)&0xFF | carry
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	c.shouldZeroFlag(rotated)

	if carry == 1 {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	return rotated
}

// rotateRight rotates the given value right by 1 bit. The most significant bit is
// copied to the carry flag, and the least significant bit is copied to the most
// significant bit.
//
//	RRC n
//	n = A, B, C, D, E, H, L, (HL)
//
// Flags affected:
//
//	Z - Set if Result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) rotateRight(value uint8) uint8 {
	newCarry := value & 0x1
	computed := (value >> 1) | (newCarry << 7)
	c.shouldZeroFlag(computed)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	if newCarry == 1 {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	return computed
}

// rotateRightThroughCarry rotates the given value right by 1 bit through the carry flag.
//
//	RR n
//	n = A, B, C, D, E, H, L, (HL)
//
// Flags affected:
//
//	Z - Set if Result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) rotateRightThroughCarry(value uint8) uint8 {
	newCarry := value & 0x01
	computed := value >> 1
	if c.isFlagSet(FlagCarry) {
		computed |= 1 << 7
	}

	if newCarry == 1 {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	c.shouldZeroFlag(computed)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	return computed
}

// rotateLeftThroughCarry rotates the given value left by 1 bit through the carry flag.
//
//	RL n
//	n = A, B, C, D, E, H, L, (HL)
//
// Flags affected:
//
//	Z - Set if Result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 7 data.
func (c *CPU) rotateLeftThroughCarry(value uint8) uint8 {
	newCarry := value >> 7
	computed := (value << 1) & 0xFF
	if c.isFlagSet(FlagCarry) {
		computed |= 0x01
	}

	if newCarry == 1 {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	c.shouldZeroFlag(computed)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	return computed
}

// rotateLeftAccumulator rotates the accumulator left by 1 bit. The least significant bit is
// copied to the carry flag, and the most significant bit is copied to the least
// significant bit.
//
//	RLCA
//
// Flags affected:
//
//	Z - Reset.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 7 data.
func (c *CPU) rotateLeftAccumulator() {
	carry := c.A&0x80 != 0
	c.A = (c.A << 1) & 0xFF
	if carry {
		c.A |= 0x01
	}
	c.clearFlag(FlagZero)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	if carry {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
}

// rotateLeftAccumulatorThroughCarry rotates the accumulator left by 1 bit through the carry flag.
//
//	RLA
//
// Flags affected:
//
//	Z - Reset.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 7 data.
func (c *CPU) rotateLeftAccumulatorThroughCarry() {
	newCarry := c.A&0x80 != 0
	oldCarry := c.isFlagSet(FlagCarry)
	c.A = (c.A << 1) & 0xFF
	if oldCarry {
		c.A |= 0x01
	}
	c.clearFlag(FlagZero)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	if newCarry {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
}

// rotateRightAccumulator rotates the accumulator right by 1 bit. The most significant bit is
// copied to the carry flag, and the least significant bit is copied to the most
// significant bit.
//
//	RRCA
//
// Flags affected:
//
//	Z - Reset.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) rotateRightAccumulator() {
	carry := c.A&0x1 != 0
	c.A = (c.A >> 1) & 0xFF
	if carry {
		c.A |= 0x80
	}
	c.clearFlag(FlagZero)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	if carry {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
}

// rotateRightAccumulatorThroughCarry rotates the accumulator right by 1 bit through the carry flag.
//
//	RRA
//
// Flags affected:
//
//	Z - Reset.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) rotateRightAccumulatorThroughCarry() {
	newCarry := c.A&0x1 != 0
	c.A = (c.A >> 1) & 0xFF
	if c.isFlagSet(FlagCarry) {
		c.A |= 0x80
	}

	c.clearFlag(FlagZero)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	if newCarry {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
}

func init() {
	DefineInstruction(0x07, "RLCA", func(c *CPU) { c.rotateLeftAccumulator() })
	DefineInstruction(0x0F, "RRCA", func(c *CPU) { c.rotateRightAccumulator() })
	DefineInstruction(0x17, "RLA", func(c *CPU) { c.rotateLeftAccumulatorThroughCarry() })
	DefineInstruction(0x1F, "RRA", func(c *CPU) { c.rotateRightAccumulatorThroughCarry() })
}

package cpu

// rotateLeft rotates the given value left by 1 bit. The least significant bit is
// copied to the carry flag, and the most significant bit is copied to the least
// significant bit.
//
//	RLC n
//	n = A, B, C, D, E, H, L, (HL)
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 7 data.
func (c *CPU) rotateLeft(value uint8) uint8 {
	result := value << 1
	if value&0x80 == 0x80 {
		c.setFlag(FlagCarry)
		result ^= 0x01
	} else {
		c.clearFlag(FlagCarry)
	}
	c.shouldZeroFlag(result)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	return result
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
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) rotateRight(value uint8) uint8 {
	result := value >> 1
	if value&1 == 1 {
		c.setFlag(FlagCarry)
		result ^= 0x80
	} else {
		c.clearFlag(FlagCarry)
	}
	c.shouldZeroFlag(result)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	return result
}

// rotateLeftThroughCarry rotates the given value left by 1 bit through the carry flag.
//
//	RL n
//	n = A, B, C, D, E, H, L, (HL)
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 7 data.
func (c *CPU) rotateLeftThroughCarry(value uint8) uint8 {
	result := value << 1
	if c.isFlagSet(FlagCarry) {
		result ^= 0x01
	}
	if value&0x80 == 0x80 {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	c.shouldZeroFlag(result)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	return result
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
	computed := c.A << 1
	if c.A&0x80 == 0x80 {
		c.setFlag(FlagCarry)
		computed ^= 0x01
	} else {
		c.clearFlag(FlagCarry)
	}
	c.clearFlag(FlagZero)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	c.A = computed
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
	bit7 := false
	if c.A&0x80 == 0x80 {
		bit7 = true
	}
	computed := c.A << 1
	if c.isFlagSet(FlagCarry) {
		computed ^= 0x01
	}
	if bit7 {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	c.clearFlag(FlagZero)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	c.A = computed
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
	computed := c.A >> 1
	if c.A&1 == 1 {
		c.setFlag(FlagCarry)
		computed ^= 0x80
	} else {
		c.clearFlag(FlagCarry)
	}
	c.clearFlag(FlagZero)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	c.A = computed
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
	bit0 := false
	if c.A&1 == 1 {
		bit0 = true
	}
	computed := c.A >> 1
	if c.isFlagSet(FlagCarry) {
		computed ^= 0x80
	}
	if bit0 {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	c.clearFlag(FlagZero)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	c.A = computed
}

// rotateRightThroughCarry rotates the given value right by 1 bit through the carry flag.
//
//	RR n
//	n = A, B, C, D, E, H, L, (HL)
//
// IF affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) rotateRightThroughCarry(value uint8) uint8 {
	result := value >> 1
	if c.isFlagSet(FlagCarry) {
		result ^= 0x80
	}
	if result&1 == 1 {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	c.shouldZeroFlag(result)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	return result
}

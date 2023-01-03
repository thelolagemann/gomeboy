package cpu

// andRegister performs a bitwise AND operation on the given Register and the
// A Register.
//
//	AND n
//	n = A, B, C, D, E, H, L, (HL)
//
// IF affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Set.
//	C - Reset.
func (c *CPU) andRegister(reg *Register) {
	c.A = c.and(c.A, *reg)
}

// and is a helper function for that performs a bitwise AND operation on the
// two given values, and sets the flags accordingly.
func (c *CPU) and(a, b uint8) uint8 {
	c.setFlag(FlagHalfCarry)
	c.clearFlag(FlagCarry)
	c.clearFlag(FlagSubtract)
	computed := a & b
	if computed == 0x00 {
		c.setFlag(FlagZero)
	} else {
		c.clearFlag(FlagZero)
	}
	return computed
}

// orRegister performs a bitwise OR operation on the given Register and the A
// Register.
//
//	OR n
//	n = A, B, C, D, E, H, L, (HL)
//
// IF affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Reset.
func (c *CPU) orRegister(reg *Register) {
	c.A = c.or(c.A, *reg)
}

// or is a helper function for that performs a bitwise OR operation on the two
// given values, and sets the flags accordingly.
func (c *CPU) or(a, b uint8) uint8 {
	c.clearFlag(FlagHalfCarry)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagCarry)
	computed := a | b
	if computed == 0x00 {
		c.setFlag(FlagZero)
	} else {
		c.clearFlag(FlagZero)
	}
	return computed
}

// xorRegister performs a bitwise XOR operation on the given Register and the A
// Register.
//
//	XOR n
//	n = A, B, C, D, E, H, L, (HL)
//
// IF affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Reset.
func (c *CPU) xorRegister(reg *Register) {
	c.A = c.xor(c.A, *reg)
}

// xor is a helper function for that performs a bitwise XOR operation on the two
// given values, and sets the flags accordingly.
func (c *CPU) xor(a, b uint8) uint8 {
	c.clearFlag(FlagHalfCarry)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagCarry)
	computed := a ^ b
	if computed == 0x00 {
		c.setFlag(FlagZero)
	} else {
		c.clearFlag(FlagZero)
	}
	return computed
}

// compareRegister compares the given Register with the A Register.
//
//	CP n
//	n = A, B, C, D, E, H, L, (HL)
//
// IF affected:
//
//	Z - Set if result is zero.
//	N - Set.
//	H - Set if no borrow from bit 4.
//	C - Set if no borrow.
func (c *CPU) compareRegister(reg *Register) {
	c.compare(*reg)
}

// compare is a helper function for that compares the two given values, and sets
// the flags accordingly.
func (c *CPU) compare(b uint8) {
	// c.mmu.Bus.Log().Debugf("compare: %d %d", a, b)
	c.setFlag(FlagSubtract)
	if c.A&0xF < b&0xF {
		c.setFlag(FlagHalfCarry)
	} else {
		c.clearFlag(FlagHalfCarry)
	}
	if c.A < b {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	c.shouldZeroFlag(c.A - b)
}

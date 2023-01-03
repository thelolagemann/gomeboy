package cpu

// addN adds the given value to the A Register.
//
//	ADD A, n
//	n = 8-bit value
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Set if carry from bit 3.
//	C - Set if carry from bit 7.
func (c *CPU) addN(value uint8) {
	c.A = c.add(c.A, value)
}

// addNCarry adds the given value + the carry flag to the A Register.
//
//	ADC A, n
//	n = 8-bit value
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Set if carry from bit 3.
//	C - Set if carry from bit 7.
func (c *CPU) addNCarry(value uint8) {
	if c.isFlagSet(FlagCarry) {
		value++
	}
	c.A = c.add(c.A, value)
}

// subtractN subtracts the given value from the A Register.
//
//	SUB n
//	n = 8-bit value
//
// IF affected:
//
//	Z - Set if result is zero.
//	N - Set.
//	H - Set if no borrow from bit 4.
//	C - Set if no borrow.
func (c *CPU) subtractN(value uint8) {
	c.A = c.sub(c.A, value)
}

// subtractNCarry subtracts the given value + the carry flag from the A Register.
//
//	SBC A, n
//	n = 8-bit value
//
// IF affected:
//
//	Z - Set if result is zero.
//	N - Set.
//	H - Set if no borrow from bit 4.
//	C - Set if no borrow.
func (c *CPU) subtractNCarry(value uint8) {
	if c.isFlagSet(FlagCarry) {
		value++
	}
	c.A = c.sub(c.A, value)
}

// incrementN increments the given register by 1.
//
//	INC n
//	n = 8-bit register
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Set if carry from bit 3.
//	C - Not affected.
func (c *CPU) incrementN(register *Register) {
	*register = c.increment(*register)
}

// incrementNN increments the given RegisterPair by 1.
//
//	INC nn
//	nn = 16-bit register
func (c *CPU) incrementNN(register *RegisterPair) {
	register.SetUint16(register.Uint16() + 1)
}

// decrementN decrements the given register by 1.
//
//	DEC n
//	n = 8-bit register
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Set.
//	H - Set if no borrow from bit 4.
//	C - Not affected.
func (c *CPU) decrementN(register *Register) {
	*register = c.decrement(*register)
}

// decrementNN decrements the given RegisterPair by 1.
//
//	DEC nn
//	nn = 16-bit register
//
// Flags affected:
//
//	Z - Not affected.
//	N - Set.
//	H - Set if no borrow from bit 12.
//	C - Not affected.
func (c *CPU) decrementNN(register *RegisterPair) {
	register.SetUint16(register.Uint16() - 1)
}

// addHLRR adds the given RegisterPair to the HL RegisterPair.
//
//	ADD HL, rr
//	rr = 16-bit register
//
// Flags affected:
//
//	Z - Not affected.
//	N - Reset.
//	H - Set if carry from bit 11.
//	C - Set if carry from bit 15.
func (c *CPU) addHL(register *RegisterPair) {
	c.HL.SetUint16(c.addUint16(c.HL.Uint16(), register.Uint16()))
}

// add is a helper function for adding two bytes together and
// setting the flags accordingly.
func (c *CPU) add(a, b uint8) uint8 {
	computed := a + b
	c.clearFlag(FlagSubtract)
	if computed == 0x00 {
		c.setFlag(FlagZero)
	} else {
		c.clearFlag(FlagZero)
	}
	if (computed^b&a)&0x10 == 0x10 {
		c.setFlag(FlagHalfCarry)
	} else {
		c.clearFlag(FlagHalfCarry)
	}
	if computed < a {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	return computed
}

// addBytePair is a helper function for adding two uint16 values together and
// setting the flags accordingly.
func (c *CPU) addUint16(a, b uint16) uint16 {
	computed := a + b
	if (computed^b&a)&0x1000 == 0x1000 {
		c.setFlag(FlagHalfCarry)
	} else {
		c.clearFlag(FlagHalfCarry)
	}
	if computed < a {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	c.clearFlag(FlagSubtract)
	return computed
}

// sub is a helper function for subtracting two bytes together and
// setting the flags accordingly.
func (c *CPU) sub(a, b uint8) uint8 {
	computed := a - b
	// if the lower nibble of a is less than the lower nibble of b, then
	// there was a borrow from bit 4.
	if a&0x0f < b&0x0f {
		c.setFlag(FlagHalfCarry)
	} else {
		c.clearFlag(FlagHalfCarry)
	}
	c.setFlag(FlagSubtract)
	if computed == 0x00 {
		c.setFlag(FlagZero)
	} else {
		c.clearFlag(FlagZero)
	}
	if computed > a {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	return computed
}

// increment is a helper function for incrementing a byte and
// setting the flags accordingly.
func (c *CPU) increment(value uint8) uint8 {
	incremented := value + 0x01
	c.clearFlag(FlagSubtract)
	if incremented == 0x00 {
		c.setFlag(FlagZero)
	} else {
		c.clearFlag(FlagZero)
	}
	if (incremented^value)&0x10 == 0x10 {
		c.setFlag(FlagHalfCarry)
	} else {
		c.clearFlag(FlagHalfCarry)
	}
	return incremented
}

// decrement is a helper function for decrementing a byte and
// setting the flags accordingly.
func (c *CPU) decrement(value uint8) uint8 {
	decremented := value - 0x01
	c.setFlag(FlagSubtract)
	if decremented == 0x00 {
		c.setFlag(FlagZero)
	} else {
		c.clearFlag(FlagZero)
	}
	if (decremented^value)&0x10 == 0x10 {
		c.setFlag(FlagHalfCarry)
	} else {
		c.clearFlag(FlagHalfCarry)
	}
	return decremented
}

package cpu

import "github.com/thelolagemann/go-gameboy/pkg/utils"

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

// addUint16Signed is a helper function for adding a signed byte to a uint16 value
// and setting the flags accordingly.
func (c *CPU) addUint16Signed(a uint16, b int8) uint16 {
	total := uint16(int32(a) + int32(b))

	tmpVal := a ^ uint16(b) ^ total

	if (tmpVal & 0x10) == 0x10 {
		c.setFlag(FlagHalfCarry)
	} else {
		c.clearFlag(FlagHalfCarry)
	}
	if (tmpVal & 0x100) == 0x100 {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	c.clearFlag(FlagZero)
	c.clearFlag(FlagSubtract)
	return total
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
	c.shouldZeroFlag(incremented)
	if utils.HalfCarryAdd(value, 1) {
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
	c.shouldZeroFlag(decremented)
	if value&0x0f == 0 {
		c.setFlag(FlagHalfCarry)
	} else {
		c.clearFlag(FlagHalfCarry)
	}
	return decremented
}

func init() {
	InstructionSet[0x03] = NewInstruction("INC BC", 1, 2, func(c *CPU) { c.incrementNN(c.BC) })
	InstructionSet[0x04] = NewInstruction("INC B", 1, 1, func(c *CPU) { c.incrementN(&c.B) })
	InstructionSet[0x05] = NewInstruction("DEC B", 1, 1, func(c *CPU) { c.decrementN(&c.B) })
	InstructionSet[0x09] = NewInstruction("ADD HL, BC", 1, 2, func(c *CPU) { c.addHL(c.BC) })
	InstructionSet[0x0B] = NewInstruction("DEC BC", 1, 2, func(c *CPU) { c.decrementNN(c.BC) })
	InstructionSet[0x0C] = NewInstruction("INC C", 1, 1, func(c *CPU) { c.incrementN(&c.C) })
	InstructionSet[0x0D] = NewInstruction("DEC C", 1, 1, func(c *CPU) { c.decrementN(&c.C) })
	InstructionSet[0x13] = NewInstruction("INC DE", 1, 2, func(c *CPU) { c.incrementNN(c.DE) })
	InstructionSet[0x14] = NewInstruction("INC D", 1, 1, func(c *CPU) { c.incrementN(&c.D) })
	InstructionSet[0x15] = NewInstruction("DEC D", 1, 1, func(c *CPU) { c.decrementN(&c.D) })
	InstructionSet[0x19] = NewInstruction("ADD HL, DE", 1, 2, func(c *CPU) { c.addHL(c.DE) })
	InstructionSet[0x1B] = NewInstruction("DEC DE", 1, 2, func(c *CPU) { c.decrementNN(c.DE) })
	InstructionSet[0x1C] = NewInstruction("INC E", 1, 1, func(c *CPU) { c.incrementN(&c.E) })
	InstructionSet[0x1D] = NewInstruction("DEC E", 1, 1, func(c *CPU) { c.decrementN(&c.E) })
	InstructionSet[0x23] = NewInstruction("INC HL", 1, 2, func(c *CPU) { c.incrementNN(c.HL) })
	InstructionSet[0x24] = NewInstruction("INC H", 1, 1, func(c *CPU) { c.incrementN(&c.H) })
	InstructionSet[0x25] = NewInstruction("DEC H", 1, 1, func(c *CPU) { c.decrementN(&c.H) })
	InstructionSet[0x29] = NewInstruction("ADD HL, HL", 1, 2, func(c *CPU) { c.addHL(c.HL) })
	InstructionSet[0x2B] = NewInstruction("DEC HL", 1, 2, func(c *CPU) { c.decrementNN(c.HL) })
	InstructionSet[0x2C] = NewInstruction("INC L", 1, 1, func(c *CPU) { c.incrementN(&c.L) })
	InstructionSet[0x2D] = NewInstruction("DEC L", 1, 1, func(c *CPU) { c.decrementN(&c.L) })
	InstructionSet[0x33] = NewInstruction("INC SP", 1, 2, func(c *CPU) { c.SP++ })
	InstructionSet[0x34] = NewInstruction("INC (HL)", 1, 3, func(c *CPU) { c.mmu.Write(c.HL.Uint16(), c.increment(c.mmu.Read(c.HL.Uint16()))) })
	InstructionSet[0x35] = NewInstruction("DEC (HL)", 1, 3, func(c *CPU) { c.mmu.Write(c.HL.Uint16(), c.decrement(c.mmu.Read(c.HL.Uint16()))) })
	InstructionSet[0x39] = NewInstruction("ADD HL, SP", 1, 2, func(c *CPU) { c.HL.SetUint16(c.addUint16(c.HL.Uint16(), c.SP)) })
	InstructionSet[0x3B] = NewInstruction("DEC SP", 1, 2, func(c *CPU) { c.SP-- })
	InstructionSet[0x3C] = NewInstruction("INC A", 1, 1, func(c *CPU) { c.incrementN(&c.A) })
	InstructionSet[0x3D] = NewInstruction("DEC A", 1, 1, func(c *CPU) { c.decrementN(&c.A) })
	InstructionSet[0xC1] = NewInstruction("POP BC", 1, 3, func(c *CPU) { c.popNN(&c.B, &c.C) })
	InstructionSet[0xC5] = NewInstruction("PUSH BC", 1, 4, func(c *CPU) { c.pushNN(c.B, c.C) })
	InstructionSet[0xD1] = NewInstruction("POP DE", 1, 3, func(c *CPU) { c.popNN(&c.D, &c.E) })
	InstructionSet[0xD5] = NewInstruction("PUSH DE", 1, 4, func(c *CPU) { c.pushNN(c.D, c.E) })
	InstructionSet[0xE1] = NewInstruction("POP HL", 1, 3, func(c *CPU) { c.popNN(&c.H, &c.L) })
	InstructionSet[0xE5] = NewInstruction("PUSH HL", 1, 4, func(c *CPU) { c.pushNN(c.H, c.L) })
	InstructionSet[0xF1] = NewInstruction("POP AF", 1, 3, func(c *CPU) { c.popNN(&c.A, &c.F) })
	InstructionSet[0xF5] = NewInstruction("PUSH AF", 1, 4, func(c *CPU) { c.pushNN(c.A, c.F) })
	InstructionSet[0xE8] = NewInstruction("ADD SP, n", 2, 4, func(c *CPU, operands []byte) {
		c.SP = c.addUint16Signed(c.SP, int8(operands[0]))
	})
}

func (c *CPU) pushNN(h, l Register) {
	c.push(h)
	c.push(l)
}

func (c *CPU) popNN(h, l *Register) {
	*l = c.pop()
	*h = c.pop()
}

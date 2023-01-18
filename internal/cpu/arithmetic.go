package cpu

import "github.com/thelolagemann/go-gameboy/pkg/utils"

// increment the given value and set the flags accordingly.
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

// incrementNN increments the given RegisterPair by 1.
//
//	INC nn
//	nn = 16-bit register
//
// Flags affected:
//
//	Z - Not affected.
//	N - Reset.
//	H - Not affected.
//	C - Not affected.
func (c *CPU) incrementNN(register *RegisterPair) {
	register.SetUint16(register.Uint16() + 1)
	c.ticks(4)
}

// decrement the given value and set the flags accordingly.
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

// decrementNN decrements the given RegisterPair by 1.
//
//	DEC nn
//	nn = 16-bit register
//
// Flags affected:
//
//	Z - Not affected.
//	N - Not affected.
//	H - Not affected.
//	C - Not affected.
func (c *CPU) decrementNN(register *RegisterPair) {
	value := uint16(*register.High)<<8 | uint16(*register.Low)
	value--
	*register.High = uint8(value >> 8)
	*register.Low = uint8(value & 0xFF)
	c.ticks(4)
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
	c.ticks(4)
}

// add is a helper function for adding two bytes together and
// setting the flags accordingly.
//
// Used by:
//
//	ADD A, n
//	ADC A, n
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Set if carry from bit 3.
//	C - Set if carry from bit 7.
func (c *CPU) add(a, b uint8, shouldCarry bool) uint8 {
	newCarry := c.isFlagSet(FlagCarry) && shouldCarry
	sum := int16(a) + int16(b)
	sumHalf := int16(a&0xF) + int16(b&0xF)
	if newCarry {
		sum++
		sumHalf++
	}
	c.shouldZeroFlag(uint8(sum))
	c.clearFlag(FlagSubtract)
	if sumHalf > 0xF {
		c.setFlag(FlagHalfCarry)
	} else {
		c.clearFlag(FlagHalfCarry)
	}
	if sum > 0xFF {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	return uint8(sum)
}

// addBytePair is a helper function for adding two uint16 values together and
// setting the flags accordingly.
func (c *CPU) addUint16(a, b uint16) uint16 {
	sum := int32(a) + int32(b)
	c.clearFlag(FlagSubtract)
	if int32(a&0xFFF) > (sum & 0xFFF) {
		c.setFlag(FlagHalfCarry)
	} else {
		c.clearFlag(FlagHalfCarry)
	}
	if sum > 0xFFFF {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	return uint16(sum)
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
//
// Used by:
//
//	SUB A, n
//	SBC A, n
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Set.
//	H - Set if no borrow from bit 4.
//	C - Set if no borrow.
func (c *CPU) sub(a, b uint8, shouldCarry bool) uint8 {
	newCarry := c.isFlagSet(FlagCarry) && shouldCarry
	sub := int16(a) - int16(b)
	subHalf := int16(a&0x0F) - int16(b&0x0F)
	if newCarry {
		sub--
		subHalf--
	}

	c.shouldZeroFlag(uint8(sub))
	c.setFlag(FlagSubtract)
	if subHalf < 0 {
		c.setFlag(FlagHalfCarry)
	} else {
		c.clearFlag(FlagHalfCarry)
	}
	if sub < 0 {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	return uint8(sub)
}

// pushNN pushes the two registers onto the stack.
//
//	PUSH nn
//	nn = 16-bit register
//
// Flags affected:
//
//	Z - Not affected.
//	N - Not affected.
//	H - Not affected.
//	C - Not affected.
func (c *CPU) pushNN(h, l Register) {
	c.ticks(4)
	c.push(h, l)
}

// popNN pops the two registers off the stack.
//
//	POP nn
//	nn = 16-bit register
//
// Flags affected:
//
//	Z - Not affected.
//	N - Not affected.
//	H - Not affected.
//	C - Not affected.
func (c *CPU) popNN(h, l *Register) {
	*l = c.readByte(c.SP)
	c.SP++
	*h = c.readByte(c.SP)
	c.SP++
}

func (c *CPU) addSPSigned() uint16 {
	result := c.addUint16Signed(c.SP, int8(c.readByte(c.PC)))

	c.ticks(4)
	return result
}

func init() {
	DefineInstruction(0x03, "INC BC", func(c *CPU) { c.incrementNN(c.BC) })
	DefineInstruction(0x04, "INC B", func(c *CPU) { c.B = c.increment(c.B) })
	DefineInstruction(0x05, "DEC B", func(c *CPU) { c.B = c.decrement(c.B) })
	DefineInstruction(0x09, "ADD HL, BC", func(c *CPU) { c.addHL(c.BC) })
	DefineInstruction(0x0B, "DEC BC", func(c *CPU) { c.decrementNN(c.BC) })
	DefineInstruction(0x0C, "INC C", func(c *CPU) { c.C = c.increment(c.C) })
	DefineInstruction(0x0D, "DEC C", func(c *CPU) { c.C = c.decrement(c.C) })
	DefineInstruction(0x13, "INC DE", func(c *CPU) { c.incrementNN(c.DE) })
	DefineInstruction(0x14, "INC D", func(c *CPU) { c.D = c.increment(c.D) })
	DefineInstruction(0x15, "DEC D", func(c *CPU) { c.D = c.decrement(c.D) })
	DefineInstruction(0x19, "ADD HL, DE", func(c *CPU) { c.addHL(c.DE) })
	DefineInstruction(0x1B, "DEC DE", func(c *CPU) { c.decrementNN(c.DE) })
	DefineInstruction(0x1C, "INC E", func(c *CPU) { c.E = c.increment(c.E) })
	DefineInstruction(0x1D, "DEC E", func(c *CPU) { c.E = c.decrement(c.E) })
	DefineInstruction(0x23, "INC HL", func(c *CPU) { c.incrementNN(c.HL) })
	DefineInstruction(0x24, "INC H", func(c *CPU) { c.H = c.increment(c.H) })
	DefineInstruction(0x25, "DEC H", func(c *CPU) { c.H = c.decrement(c.H) })
	DefineInstruction(0x29, "ADD HL, HL", func(c *CPU) { c.addHL(c.HL) })
	DefineInstruction(0x2B, "DEC HL", func(c *CPU) { c.decrementNN(c.HL) })
	DefineInstruction(0x2C, "INC L", func(c *CPU) { c.L = c.increment(c.L) })
	DefineInstruction(0x2D, "DEC L", func(c *CPU) { c.L = c.decrement(c.L) })
	DefineInstruction(0x33, "INC SP", func(c *CPU) { c.SP++; c.ticks(4) })
	DefineInstruction(0x34, "INC (HL)", func(c *CPU) {
		c.writeByte(c.HL.Uint16(), c.increment(c.readByte(c.HL.Uint16())))
	})
	DefineInstruction(0x35, "DEC (HL)", func(c *CPU) {
		c.writeByte(c.HL.Uint16(), c.decrement(c.readByte(c.HL.Uint16())))
	})
	DefineInstruction(0x39, "ADD HL, SP", func(c *CPU) { c.HL.SetUint16(c.addUint16(c.HL.Uint16(), c.SP)); c.ticks(4) })
	DefineInstruction(0x3B, "DEC SP", func(c *CPU) { c.SP--; c.ticks(4) })
	DefineInstruction(0x3C, "INC A", func(c *CPU) { c.A = c.increment(c.A) })
	DefineInstruction(0x3D, "DEC A", func(c *CPU) { c.A = c.decrement(c.A) })
	DefineInstruction(0xC1, "POP BC", func(c *CPU) { c.popNN(&c.B, &c.C) })
	DefineInstruction(0xC5, "PUSH BC", func(c *CPU) { c.pushNN(c.B, c.C) })
	DefineInstruction(0xD1, "POP DE", func(c *CPU) { c.popNN(&c.D, &c.E) })
	DefineInstruction(0xD5, "PUSH DE", func(c *CPU) { c.pushNN(c.D, c.E) })
	DefineInstruction(0xE1, "POP HL", func(c *CPU) { c.popNN(&c.H, &c.L) })
	DefineInstruction(0xE5, "PUSH HL", func(c *CPU) { c.pushNN(c.H, c.L) })
	DefineInstruction(0xF1, "POP AF", func(c *CPU) {
		c.popNN(&c.A, &c.F)
		c.F &= 0xF0
	})
	DefineInstruction(0xF5, "PUSH AF", func(c *CPU) { c.pushNN(c.A, c.F) })
	DefineInstruction(0xE8, "ADD SP, r8", func(c *CPU) {
		c.SP = c.addSPSigned()
		c.ticks(4)
	})
}

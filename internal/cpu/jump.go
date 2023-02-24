package cpu

import (
	"fmt"
)

// pushStack pushes a 16 bit value onto the stack.
func (c *CPU) pushStack(high, low uint8) {
	c.tickCycle()

	c.push(high, low)
}

func (c *CPU) push(high, low uint8) {
	c.SP--
	c.writeByte(c.SP, high)
	c.SP--
	c.writeByte(c.SP, low)
}

// popStack pops a 16 bit value off the stack.
func (c *CPU) popStack(high *uint8, low *uint8) {
	*low = c.readByte(c.SP)
	c.SP++
	*high = c.readByte(c.SP)
	c.SP++
}

// call pushes the address of the next instruction onto the stack and jumps to
// the given address.
//
//	CALL nn
//	nn = 16-bit immediate value
func (c *CPU) call() {
	PC := c.PC + 2
	c.jumpAbsolute()
	c.push(uint8(PC>>8), uint8(PC&0xFF))
}

// callConditional pushes the address of the next instruction onto the stack and
// jumps to the given address if the given condition is true.
//
//	CALL cc, nn
//	cc = NZ, Z, NC, C
//	nn = 16-bit immediate value
func (c *CPU) callConditional(condition bool) {
	if condition {
		c.call()
	} else {
		c.skipOperand()
		c.skipOperand()
	}
}

// jumpRelative jumps to the address relative to the current PC.
//
//	JR e
//	e = 8-bit signed immediate value
func (c *CPU) jumpRelative() {
	value := int8(c.readOperand())
	c.PC = uint16(int16(c.PC) + int16(value))

	c.tickCycle()
}

// jumpRelativeConditional jumps to the address relative to the current PC if
// the given condition is true.
//
//	JR cc, e
//	cc = NZ, Z, NC, C
//	e = 8-bit signed immediate value
func (c *CPU) jumpRelativeConditional(condition bool) {
	if condition {
		c.jumpRelative()
	} else {
		c.skipOperand()
	}
}

// jumpAbsolute jumps to the given address.
//
//	JP nn
//	nn = 16-bit immediate value
func (c *CPU) jumpAbsolute() {
	low := c.readOperand()
	high := c.readOperand()

	c.PC = uint16(high)<<8 | uint16(low)
	c.tickCycle()
}

// jumpAbsoluteConditional jumps to the given address if the given condition is
// true.
//
//	JP cc, nn
//	cc = NZ, Z, NC, C
//	nn = 16-bit immediate value
func (c *CPU) jumpAbsoluteConditional(condition bool) {
	if condition {
		c.jumpAbsolute()
	} else {
		c.skipOperand()
		c.skipOperand()
	}
}

// ret pops the top two bytes off the stack and jumps to that address.
//
//	RET
func (c *CPU) ret() {
	var high, low uint8
	c.popStack(&high, &low)
	c.PC = uint16(high)<<8 | uint16(low)
	c.tickCycle()
}

// retConditional pops the top two bytes off the stack and jumps to that
// address if the given condition is true.
//
//	RET cc
//	cc = NZ, Z, NC, C
func (c *CPU) retConditional(condition bool) {
	if condition {
		c.ret()
	}
}

// retInterrupt pops the top two bytes off the stack and jumps to that address.
// It also enables interrupts.
//
//	RETI
func (c *CPU) retInterrupt() {
	c.IRQ.IME = true
	c.ret()
}

func init() {
	// 0x18 - JR n // TODO variable cycles
	DefineInstruction(0x18, "JR n", func(c *CPU) { c.jumpRelative() })
	DefineInstruction(0x20, "JR NZ, n", func(c *CPU) {
		c.jumpRelativeConditional(!c.isFlagSet(FlagZero))
	})
	DefineInstruction(0x28, "JR Z, n", func(c *CPU) {
		c.jumpRelativeConditional(c.isFlagSet(FlagZero))
	})
	DefineInstruction(0x30, "JR NC, n", func(c *CPU) {
		c.jumpRelativeConditional(!c.isFlagSet(FlagCarry))
	})
	DefineInstruction(0x38, "JR C, n", func(c *CPU) {
		c.jumpRelativeConditional(c.isFlagSet(FlagCarry))
	})
	DefineInstruction(0xC0, "RET NZ", func(c *CPU) { c.tickCycle(); c.retConditional(!c.isFlagSet(FlagZero)) })
	DefineInstruction(0xC2, "JP NZ, nn", func(c *CPU) {
		c.jumpAbsoluteConditional(!c.isFlagSet(FlagZero))
	})
	DefineInstruction(0xC3, "JP nn", func(c *CPU) {
		c.jumpAbsolute()
	})
	DefineInstruction(0xC4, "CALL NZ, nn", func(c *CPU) {
		c.callConditional(!c.isFlagSet(FlagZero))
	})
	DefineInstruction(0xC8, "RET Z", func(c *CPU) { c.tickCycle(); c.retConditional(c.isFlagSet(FlagZero)) })
	DefineInstruction(0xC9, "RET", func(c *CPU) { c.ret() })
	DefineInstruction(0xCA, "JP Z, nn", func(c *CPU) {
		c.jumpAbsoluteConditional(c.isFlagSet(FlagZero))
	})
	DefineInstruction(0xCC, "CALL Z, nn", func(c *CPU) {
		c.callConditional(c.isFlagSet(FlagZero))
	})
	DefineInstruction(0xCD, "CALL nn", func(c *CPU) {
		c.call()
	})
	DefineInstruction(0xD0, "RET NC", func(c *CPU) { c.tickCycle(); c.retConditional(!c.isFlagSet(FlagCarry)) })
	DefineInstruction(0xD2, "JP NC, nn", func(c *CPU) {
		c.jumpAbsoluteConditional(!c.isFlagSet(FlagCarry))
	})
	DefineInstruction(0xD4, "CALL NC, nn", func(c *CPU) {
		c.callConditional(!c.isFlagSet(FlagCarry))
	})
	DefineInstruction(0xD8, "RET C", func(c *CPU) { c.tickCycle(); c.retConditional(c.isFlagSet(FlagCarry)) })
	DefineInstruction(0xD9, "RETI", func(c *CPU) { c.retInterrupt() })
	DefineInstruction(0xDA, "JP C, nn", func(c *CPU) {
		c.jumpAbsoluteConditional(c.isFlagSet(FlagCarry))
	})
	DefineInstruction(0xDC, "CALL C, nn", func(c *CPU) {
		c.callConditional(c.isFlagSet(FlagCarry))
	})
	DefineInstruction(0xe9, "JP HL", func(c *CPU) {
		c.PC = c.HL.Uint16()
	})
}

// generateRSTInstructions generates the 8 RST instructions.
func (c *CPU) generateRSTInstructions() {
	for i := uint8(0); i < 8; i++ {
		address := uint16(i * 8)
		DefineInstruction(0xC7+i*8, fmt.Sprintf("RST %02Xh", address), func(c *CPU) {
			c.pushStack(uint8(c.PC>>8), uint8(c.PC&0xFF))
			c.PC = address
		})
	}
}

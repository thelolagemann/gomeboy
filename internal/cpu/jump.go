package cpu

import (
	"encoding/binary"
	"fmt"
)

// pushStack pushes a 16 bit value onto the stack.
func (c *CPU) pushStack(value uint16) {
	c.mmu.Write(c.SP-1, uint8(uint16(value&0xFF00)>>8))
	c.mmu.Write(c.SP-2, uint8(value&0xFF))
	c.SP -= 2
}

// popStack pops a 16 bit value off the stack.
func (c *CPU) popStack() uint16 {
	lower := uint16(c.mmu.Read(c.SP))
	upper := uint16(c.mmu.Read(c.SP+1)) << 8
	c.SP += 2
	return lower | upper
}

// call pushes the address of the next instruction onto the stack and jumps to
// the given address.
//
//	CALL nn
//	nn = 16-bit immediate value
func (c *CPU) call(address uint16) {
	c.pushStack(c.PC)
	c.PC = address
}

// callConditional pushes the address of the next instruction onto the stack and
// jumps to the given address if the given condition is true.
//
//	CALL cc, nn
//	cc = NZ, Z, NC, C
//	nn = 16-bit immediate value
func (c *CPU) callConditional(condition bool, address uint16) {
	if condition {
		c.call(address)
	}
}

// jumpRelative jumps to the address relative to the current PC.
//
//	JR e
//	e = 8-bit signed immediate value
func (c *CPU) jumpRelative(offset uint8) {
	v := int8(offset)
	addr := int32(c.PC) + int32(v)
	c.jumpAbsolute(uint16(addr))
}

// jumpRelativeConditional jumps to the address relative to the current PC if
// the given condition is true.
//
//	JR cc, e
//	cc = NZ, Z, NC, C
//	e = 8-bit signed immediate value
func (c *CPU) jumpRelativeConditional(condition bool, offset uint8) {
	if condition {
		c.jumpRelative(offset)
	}
}

// jumpAbsolute jumps to the given address.
//
//	JP nn
//	nn = 16-bit immediate value
func (c *CPU) jumpAbsolute(address uint16) {
	c.PC = address
}

// jumpAbsoluteConditional jumps to the given address if the given condition is
// true.
//
//	JP cc, nn
//	cc = NZ, Z, NC, C
//	nn = 16-bit immediate value
func (c *CPU) jumpAbsoluteConditional(condition bool, address uint16) {
	if condition {
		c.jumpAbsolute(address)
	}
}

// ret pops the top two bytes off the stack and jumps to that address.
//
//	RET
func (c *CPU) ret() {
	c.PC = c.popStack()
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
	c.ret()
	c.irq.IME = true
}

func init() {
	// 0x18 - JR n
	InstructionSet[0x18] = NewInstruction("JR n", 2, 3, func(c *CPU, operands []byte) {
		c.jumpRelative(operands[0])
	})
	// 0x20 - JR NZ, n
	InstructionSet[0x20] = NewInstruction("JR NZ, n", 2, 3, func(c *CPU, operands []byte) {
		c.jumpRelativeConditional(!c.isFlagSet(FlagZero), operands[0])
	})
	// 0x28 - JR Z, n
	InstructionSet[0x28] = NewInstruction("JR Z, n", 2, 3, func(c *CPU, operands []byte) {
		c.jumpRelativeConditional(c.isFlagSet(FlagZero), operands[0])
	})
	// 0x30 - JR NC, n
	InstructionSet[0x30] = NewInstruction("JR NC, n", 2, 3, func(c *CPU, operands []byte) {
		c.jumpRelativeConditional(!c.isFlagSet(FlagCarry), operands[0])
	})
	// 0x38 - JR C, n
	InstructionSet[0x38] = NewInstruction("JR C, n", 2, 3, func(c *CPU, operands []byte) {
		c.jumpRelativeConditional(c.isFlagSet(FlagCarry), operands[0])
	})
	// 0xC1 - RET NZ
	InstructionSet[0xC0] = NewInstruction("RET NZ", 1, 5, func(c *CPU, operands []byte) {
		c.retConditional(!c.isFlagSet(FlagZero))
	})
	// 0xC2 - JP NZ, nn
	InstructionSet[0xC2] = NewInstruction("JP NZ, nn", 3, 3, func(c *CPU, operands []byte) {
		c.jumpAbsoluteConditional(!c.isFlagSet(FlagZero), binary.LittleEndian.Uint16(operands))
	})
	// 0xC3 - JP nn
	InstructionSet[0xC3] = NewInstruction("JP nn", 3, 4, func(c *CPU, operands []byte) {
		c.jumpAbsolute(binary.LittleEndian.Uint16(operands))
	})
	// 0xC4 - CALL NZ, nn
	InstructionSet[0xC4] = NewInstruction("CALL NZ, nn", 3, 3, func(c *CPU, operands []byte) {
		c.callConditional(!c.isFlagSet(FlagZero), binary.LittleEndian.Uint16(operands))
	})
	// 0xC8 - RET Z
	InstructionSet[0xC8] = NewInstruction("RET Z", 1, 5, func(c *CPU, operands []byte) {
		c.retConditional(c.isFlagSet(FlagZero))
	})
	// 0xC9 - RET
	InstructionSet[0xC9] = NewInstruction("RET", 1, 4, func(c *CPU, operands []byte) {
		c.ret()
	})
	// 0xCA - JP Z, nn
	InstructionSet[0xCA] = NewInstruction("JP Z, nn", 3, 3, func(c *CPU, operands []byte) {
		c.jumpAbsoluteConditional(c.isFlagSet(FlagZero), binary.LittleEndian.Uint16(operands))
	})
	// 0xCC - CALL Z, nn
	InstructionSet[0xCC] = NewInstruction("CALL Z, nn", 3, 3, func(c *CPU, operands []byte) {
		c.callConditional(c.isFlagSet(FlagZero), binary.LittleEndian.Uint16(operands))
	})
	// 0xCD - CALL nn
	InstructionSet[0xCD] = NewInstruction("CALL nn", 3, 6, func(c *CPU, operands []byte) {
		c.call(binary.LittleEndian.Uint16(operands))
	})
	// 0xD0 - RET NC
	InstructionSet[0xD0] = NewInstruction("RET NC", 1, 5, func(c *CPU, operands []byte) {
		c.retConditional(!c.isFlagSet(FlagCarry))
	})
	// 0xD2 - JP NC, nn
	InstructionSet[0xD2] = NewInstruction("JP NC, nn", 3, 3, func(c *CPU, operands []byte) {
		c.jumpAbsoluteConditional(!c.isFlagSet(FlagCarry), binary.LittleEndian.Uint16(operands))
	})
	// 0xD4 - CALL NC, nn
	InstructionSet[0xD4] = NewInstruction("CALL NC, nn", 3, 3, func(c *CPU, operands []byte) {
		c.callConditional(!c.isFlagSet(FlagCarry), binary.LittleEndian.Uint16(operands))
	})
	// 0xD8 - RET C
	InstructionSet[0xD8] = NewInstruction("RET C", 1, 5, func(c *CPU, operands []byte) {
		c.retConditional(c.isFlagSet(FlagCarry))
	})
	// 0xD9 - RETI
	InstructionSet[0xD9] = NewInstruction("RETI", 1, 4, func(c *CPU, operands []byte) {
		c.retInterrupt()
	})
	// 0xDA - JP C, nn
	InstructionSet[0xDA] = NewInstruction("JP C, nn", 3, 3, func(c *CPU, operands []byte) {
		c.jumpAbsoluteConditional(c.isFlagSet(FlagCarry), binary.LittleEndian.Uint16(operands))
	})
	// 0xDC - CALL C, nn
	InstructionSet[0xDC] = NewInstruction("CALL C, nn", 3, 3, func(c *CPU, operands []byte) {
		c.callConditional(c.isFlagSet(FlagCarry), binary.LittleEndian.Uint16(operands))
	})
	// 0xE9 - JP (HL)
	InstructionSet[0xE9] = NewInstruction("JP (HL)", 1, 1, func(c *CPU, operands []byte) {
		c.jumpAbsolute(c.HL.Uint16())
	})

}

// generateRSTInstructions generates the 8 RST instructions.
func (c *CPU) generateRSTInstructions() {
	for i := uint8(0); i < 8; i++ {
		address := uint16(i * 8)
		InstructionSet[0xC7+i*8] = NewInstruction(fmt.Sprintf("RST %02Xh", address), 1, 4, func(c *CPU, operands []byte) {
			c.call(address)
		})
	}
}

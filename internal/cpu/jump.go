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
	c.irq.Enabling = true
}

func init() {
	// 0x18 - JR n // TODO variable cycles
	DefineInstruction(0x18, "JR n", func(c *CPU, operands []byte) { c.jumpRelative(operands[0]) }, Length(2), Cycles(3))
	DefineInstruction(0x20, "JR NZ, n", func(c *CPU, operands []byte) {
		c.jumpRelativeConditional(!c.isFlagSet(FlagZero), operands[0])
	}, Length(2), Cycles(3))
	DefineInstruction(0x28, "JR Z, n", func(c *CPU, operands []byte) {
		c.jumpRelativeConditional(c.isFlagSet(FlagZero), operands[0])
	}, Length(2), Cycles(3))
	DefineInstruction(0x30, "JR NC, n", func(c *CPU, operands []byte) {
		c.jumpRelativeConditional(!c.isFlagSet(FlagCarry), operands[0])
	}, Length(2), Cycles(3))
	DefineInstruction(0x38, "JR C, n", func(c *CPU, operands []byte) {
		c.jumpRelativeConditional(c.isFlagSet(FlagCarry), operands[0])
	}, Length(2), Cycles(3))
	DefineInstruction(0xC0, "RET NZ", func(c *CPU, operands []byte) { c.retConditional(!c.isFlagSet(FlagZero)) }, Cycles(2))
	DefineInstruction(0xC2, "JP NZ, nn", func(c *CPU, operands []byte) {
		c.jumpAbsoluteConditional(!c.isFlagSet(FlagZero), binary.LittleEndian.Uint16(operands))
	}, Length(3), Cycles(3))
	DefineInstruction(0xC3, "JP nn", func(c *CPU, operands []byte) {
		c.jumpAbsolute(binary.LittleEndian.Uint16(operands))
	}, Length(3), Cycles(4))
	DefineInstruction(0xC4, "CALL NZ, nn", func(c *CPU, operands []byte) {
		c.callConditional(!c.isFlagSet(FlagZero), binary.LittleEndian.Uint16(operands))
	}, Length(3), Cycles(3))
	DefineInstruction(0xC8, "RET Z", func(c *CPU, operands []byte) { c.retConditional(c.isFlagSet(FlagZero)) }, Cycles(2))
	DefineInstruction(0xC9, "RET", func(c *CPU, operands []byte) { c.ret() }, Cycles(4))
	DefineInstruction(0xCA, "JP Z, nn", func(c *CPU, operands []byte) {
		c.jumpAbsoluteConditional(c.isFlagSet(FlagZero), binary.LittleEndian.Uint16(operands))
	}, Length(3), Cycles(3))
	DefineInstruction(0xCC, "CALL Z, nn", func(c *CPU, operands []byte) {
		c.callConditional(c.isFlagSet(FlagZero), binary.LittleEndian.Uint16(operands))
	}, Length(3), Cycles(3))
	DefineInstruction(0xCD, "CALL nn", func(c *CPU, operands []byte) {
		c.call(binary.LittleEndian.Uint16(operands))
	}, Length(3), Cycles(5))
	DefineInstruction(0xD0, "RET NC", func(c *CPU, operands []byte) { c.retConditional(!c.isFlagSet(FlagCarry)) }, Cycles(2))
	DefineInstruction(0xD2, "JP NC, nn", func(c *CPU, operands []byte) {
		c.jumpAbsoluteConditional(!c.isFlagSet(FlagCarry), binary.LittleEndian.Uint16(operands))
	}, Length(3), Cycles(3))
	DefineInstruction(0xD4, "CALL NC, nn", func(c *CPU, operands []byte) {
		c.callConditional(!c.isFlagSet(FlagCarry), binary.LittleEndian.Uint16(operands))
	}, Length(3), Cycles(3))
	DefineInstruction(0xD8, "RET C", func(c *CPU, operands []byte) { c.retConditional(c.isFlagSet(FlagCarry)) }, Cycles(2))
	DefineInstruction(0xD9, "RETI", func(c *CPU, operands []byte) { c.retInterrupt() }, Cycles(4))
	DefineInstruction(0xDA, "JP C, nn", func(c *CPU, operands []byte) {
		c.jumpAbsoluteConditional(c.isFlagSet(FlagCarry), binary.LittleEndian.Uint16(operands))
	}, Length(3), Cycles(3))
	DefineInstruction(0xDC, "CALL C, nn", func(c *CPU, operands []byte) {
		c.callConditional(c.isFlagSet(FlagCarry), binary.LittleEndian.Uint16(operands))
	}, Length(3), Cycles(3))
	DefineInstruction(0xe9, "JP (HL)", func(c *CPU, operands []byte) { c.jumpAbsolute(c.HL.Uint16()) }, Cycles(4))
}

// generateRSTInstructions generates the 8 RST instructions.
func (c *CPU) generateRSTInstructions() {
	for i := uint8(0); i < 8; i++ {
		address := uint16(i * 8)
		DefineInstruction(0xC7+i*8, fmt.Sprintf("RST %02Xh", address), func(c *CPU, operands []byte) {
			c.call(address)
		})
	}
}

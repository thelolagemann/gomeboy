package cpu

import (
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/ppu/lcd"
)

// call pushes the address of the next instruction onto the stack and jumps to
// the address nn.
//
// Used by:
//
//	CALL cc, nn
//	CALL nn
//	cc = Z, N, H, C
//	nn = 16-bit immediate value
func (c *CPU) call(condition bool) {
	if condition {
		PC := c.PC + 2
		c.jumpAbsolute(true)
		c.push(uint8(PC>>8), uint8(PC&0xFF))
	} else {
		c.skipOperand()
		c.skipOperand()
	}
}

// jumpRelative jumps to the address relative to the current PC.
//
// Used by:
//
//	JR cc, s8
//	JR s8
//	cc = Z, N, H, C
//	s8 = 8-bit signed immediate value
func (c *CPU) jumpRelative(condition bool) {
	if condition {
		value := int8(c.readOperand())
		c.PC = uint16(int16(c.PC) + int16(value))

		c.s.Tick(4)
	} else {
		c.s.Tick(4)
		c.PC++
	}
}

// jumpAbsolute jumps to the address nn.
//
// Used by:
//
//	JP cc, nn
//	JP nn
//	nn = 16-bit immediate value
func (c *CPU) jumpAbsolute(condition bool) {
	if condition {
		low := c.readOperand()
		high := c.readOperand()

		c.PC = uint16(high)<<8 | uint16(low)
		c.s.Tick(4)
	} else {
		c.skipOperand()
		c.skipOperand()
	}
}

// ret pops the top two bytes off the stack and jumps to that address.
//
// Used by:
//
//	RET
//	RET cc
//	RETI
func (c *CPU) ret(condition bool) {
	if condition {
		var high, low uint8
		c.pop(&high, &low)
		c.PC = uint16(high)<<8 | uint16(low)
		c.s.Tick(4)
	}
}

// push a 16-bit value onto the stack.
func (c *CPU) push(high, low uint8) {
	if c.SP >= 0xFE00 && c.SP <= 0xFEFF && c.ppu.Mode == lcd.OAM {
		c.ppu.WriteCorruptionOAM()
	}
	c.SP--
	c.writeByte(c.SP, high)
	c.SP--
	c.writeByte(c.SP, low)
}

// pop a 16 bit value off the stack.
func (c *CPU) pop(high *uint8, low *uint8) {
	*low = c.readByte(c.SP)
	c.SP++
	*high = c.readByte(c.SP)
	c.SP++
}

func init() {
	// 0x18 - JR n
	DefineInstruction(0x18, "JR n", func(c *CPU) { c.jumpRelative(true) })
	DefineInstruction(0x20, "JR NZ, n", func(c *CPU) {
		c.jumpRelative(!c.isFlagSet(FlagZero))
	})
	DefineInstruction(0x28, "JR Z, n", func(c *CPU) {
		c.jumpRelative(c.isFlagSet(FlagZero))
	})
	DefineInstruction(0x30, "JR NC, n", func(c *CPU) {
		c.jumpRelative(!c.isFlagSet(FlagCarry))
	})
	DefineInstruction(0x38, "JR C, n", func(c *CPU) {
		c.jumpRelative(c.isFlagSet(FlagCarry))
	})
	DefineInstruction(0xC0, "RET NZ", func(c *CPU) { c.s.Tick(4); c.ret(!c.isFlagSet(FlagZero)) })
	DefineInstruction(0xC2, "JP NZ, nn", func(c *CPU) {
		c.jumpAbsolute(!c.isFlagSet(FlagZero))
	})
	DefineInstruction(0xC3, "JP nn", func(c *CPU) {
		c.jumpAbsolute(true)
	})
	DefineInstruction(0xC4, "CALL NZ, nn", func(c *CPU) {
		c.call(!c.isFlagSet(FlagZero))
	})
	DefineInstruction(0xC8, "RET Z", func(c *CPU) { c.s.Tick(4); c.ret(c.isFlagSet(FlagZero)) })
	DefineInstruction(0xC9, "RET", func(c *CPU) { c.ret(true) })
	DefineInstruction(0xCA, "JP Z, nn", func(c *CPU) {
		c.jumpAbsolute(c.isFlagSet(FlagZero))
	})
	DefineInstruction(0xCC, "CALL Z, nn", func(c *CPU) {
		c.call(c.isFlagSet(FlagZero))
	})
	DefineInstruction(0xCD, "CALL nn", func(c *CPU) {
		c.call(true)
	})
	DefineInstruction(0xD0, "RET NC", func(c *CPU) { c.s.Tick(4); c.ret(!c.isFlagSet(FlagCarry)) })
	DefineInstruction(0xD2, "JP NC, nn", func(c *CPU) {
		c.jumpAbsolute(!c.isFlagSet(FlagCarry))
	})
	DefineInstruction(0xD4, "CALL NC, nn", func(c *CPU) {
		c.call(!c.isFlagSet(FlagCarry))
	})
	DefineInstruction(0xD8, "RET C", func(c *CPU) { c.s.Tick(4); c.ret(c.isFlagSet(FlagCarry)) })
	DefineInstruction(0xD9, "RETI", func(c *CPU) { c.ime = true; c.ret(true) })
	DefineInstruction(0xDA, "JP C, nn", func(c *CPU) {
		c.jumpAbsolute(c.isFlagSet(FlagCarry))
	})
	DefineInstruction(0xDC, "CALL C, nn", func(c *CPU) {
		c.call(c.isFlagSet(FlagCarry))
	})
	DefineInstruction(0xe9, "JP HL", func(c *CPU) {
		c.PC = c.HL.Uint16()
	})
}

// generateRSTInstructions generates the 8 RST instructions.
func generateRSTInstructions() {
	for i := uint8(0); i < 8; i++ {
		address := uint16(i * 8)
		DefineInstruction(0xC7+i*8, fmt.Sprintf("RST %02Xh", address), func(c *CPU) {
			c.s.Tick(4)
			c.push(uint8(c.PC>>8), uint8(c.PC&0xFF))
			c.PC = address
		})
	}
}

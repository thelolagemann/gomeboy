package cpu

import (
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

func (c *CPU) rst(address uint16) {
	c.s.Tick(4)
	c.push(uint8(c.PC>>8), uint8(c.PC&0xFF))
	c.PC = address
}

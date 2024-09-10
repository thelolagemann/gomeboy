package cpu

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
		low := c.b.ClockedRead(c.SP)
		c.SP++
		high := c.b.ClockedRead(c.SP)
		c.SP++
		c.PC = uint16(high)<<8 | uint16(low)
		c.s.Tick(4)
	}
}

// push a 16-bit value onto the stack.
func (c *CPU) push(high, low uint8) {
	c.SP--
	c.b.ClockedWrite(c.SP, high)
	c.SP--
	c.b.ClockedWrite(c.SP, low)
}

package cpu

// call pushes the address of the next instruction onto the stack and jumps to
// the given address.
//
//	CALL nn
//	nn = 16-bit immediate value
func (c *CPU) call(address uint16) {
	c.push(uint8(c.PC >> 8))
	c.push(uint8(c.PC & 0xFF))
	// flip high and low bytes because of big endian
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
	if v == 0x00 {
		return
	}
	if v < 0 {
		c.PC -= uint16(-v)
	} else {
		c.PC += uint16(v)
	}
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
	lower := c.pop()
	upper := c.pop()
	c.PC = uint16(upper)<<8 | uint16(lower)
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
	c.mmu.Bus.Interrupts().IME = true
}

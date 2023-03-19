package cpu

import "github.com/thelolagemann/go-gameboy/internal/types"

// rotateLeft rotates the given value left by 1 bit. Bit 7 is copied to both
// the carry flag and the least significant bit.
//
//	RLC n
//	n = A, B, C, D, E, H, L, (HL)
//
// Flags affected:
//
//	Z - Set if Result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 7 data.
func (c *CPU) rotateLeft(value uint8) uint8 {
	carry := value >> 7
	rotated := (value<<1)&0xFF | carry
	c.setFlags(rotated == 0, false, false, carry == 1)

	return rotated
}

// rotateRight rotates the given value right by 1 bit. The most significant bit is
// copied to the carry flag, and the least significant bit is copied to the most
// significant bit.
//
//	RRC n
//	n = A, B, C, D, E, H, L, (HL)
//
// Flags affected:
//
//	Z - Set if Result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) rotateRight(value uint8) uint8 {
	newCarry := value & 0x1
	computed := (value >> 1) | (newCarry << 7)
	c.setFlags(computed == 0, false, false, newCarry == 1)
	return computed
}

// rotateRightThroughCarry rotates the given value right by 1 bit through the carry flag.
//
//	RR n
//	n = A, B, C, D, E, H, L, (HL)
//
// Flags affected:
//
//	Z - Set if Result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) rotateRightThroughCarry(value uint8) uint8 {
	newCarry := value & 0x01
	computed := value >> 1
	if c.isFlagSet(FlagCarry) {
		computed |= types.Bit7
	}

	c.setFlags(computed == 0, false, false, newCarry == 1)
	return computed
}

// rotateLeftThroughCarry rotates the given value left by 1 bit through the carry flag.
//
//	RL n
//	n = A, B, C, D, E, H, L, (HL)
//
// Flags affected:
//
//	Z - Set if Result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 7 data.
func (c *CPU) rotateLeftThroughCarry(value uint8) uint8 {
	newCarry := value >> 7
	computed := value << 1
	if c.isFlagSet(FlagCarry) {
		computed |= types.Bit0
	}

	c.setFlags(computed == 0, false, false, newCarry == 1)
	return computed
}

// rotateLeftAccumulator rotates the accumulator left by 1 bit. The least significant bit is
// copied to the carry flag, and the most significant bit is copied to the least
// significant bit.
//
//	RLCA
//
// Flags affected:
//
//	Z - Reset.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 7 data.
func (c *CPU) rotateLeftAccumulator() {
	carry := c.A >> 7
	c.A = (c.A<<1)&0xFF | carry
	c.setFlags(false, false, false, carry == 1)
}

// rotateLeftAccumulatorThroughCarry rotates the accumulator left by 1 bit through the carry flag.
//
//	RLA
//
// Flags affected:
//
//	Z - Reset.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 7 data.
func (c *CPU) rotateLeftAccumulatorThroughCarry() {
	newCarry := c.A&0x80 != 0
	oldCarry := c.isFlagSet(FlagCarry)
	c.A <<= 1
	if oldCarry {
		c.A |= 0x01
	}
	c.setFlags(false, false, false, newCarry)
}

// rotateRightAccumulator rotates the accumulator right by 1 bit. The most significant bit is
// copied to the carry flag, and the least significant bit is copied to the most
// significant bit.
//
//	RRCA
//
// Flags affected:
//
//	Z - Reset.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) rotateRightAccumulator() {
	carry := c.A&0x1 != 0
	c.A >>= 1
	if carry {
		c.A |= 0x80
	}
	c.setFlags(false, false, false, carry)
}

// rotateRightAccumulatorThroughCarry rotates the accumulator right by 1 bit through the carry flag.
//
//	RRA
//
// Flags affected:
//
//	Z - Reset.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) rotateRightAccumulatorThroughCarry() {
	newCarry := c.A&0x1 != 0
	c.A >>= 1
	if c.isFlagSet(FlagCarry) {
		c.A |= 0x80
	}

	c.setFlags(false, false, false, newCarry)
}

func init() {
	DefineInstruction(0x07, "RLCA", func(c *CPU) { c.rotateLeftAccumulator() })
	DefineInstruction(0x0F, "RRCA", func(c *CPU) { c.rotateRightAccumulator() })
	DefineInstruction(0x17, "RLA", func(c *CPU) { c.rotateLeftAccumulatorThroughCarry() })
	DefineInstruction(0x1F, "RRA", func(c *CPU) { c.rotateRightAccumulatorThroughCarry() })
}

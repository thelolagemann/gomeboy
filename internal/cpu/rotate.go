package cpu

import "github.com/thelolagemann/gomeboy/internal/types"

// rotateLeftCarry rotates n left by 1 bit. The most significant bit is copied
// to both the carry flag and the least significant bit.
//
//	RLC n
//	n = B, C, D, E, H, L, (HL), A
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 7 data.
func (c *CPU) rotateLeftCarry(n uint8) uint8 {
	carry := n & types.Bit7
	computed := n<<1 | carry>>7
	c.setFlags(computed == 0, false, false, carry == types.Bit7)

	return computed
}

// rotateRightCarry n right by 1 bit. The least significant bit is copied
// to both the carry flag and the most significant bit.
//
//	RRC n
//	n = B, C, D, E, H, L, (HL), A
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) rotateRightCarry(n uint8) uint8 {
	carry := n & types.Bit0
	computed := n>>1 | carry<<7
	c.setFlags(computed == 0, false, false, carry == types.Bit0)
	return computed
}

// rotateRightThroughCarry rotates n right by 1 bit. The carry flag is copied to
// the most significant bit, and the least significant bit is copied to the
// carry flag.
//
//	RR n
//	n = B, C, D, E, H, L, (HL), A
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) rotateRightThroughCarry(n uint8) uint8 {
	computed := n >> 1
	if c.isFlagSet(FlagCarry) {
		computed |= types.Bit7
	}

	c.setFlags(computed == 0, false, false, n&types.Bit0 == types.Bit0)
	return computed
}

// rotateLeftThroughCarry rotates n left by 1 bit.  The carry flag is copied to
// the least significant bit, and the most significant bit is copied to the
// carry flag.
//
//	RL n
//	n = B, C, D, E, H, L, (HL), A
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 7 data.
func (c *CPU) rotateLeftThroughCarry(n uint8) uint8 {
	computed := n << 1
	if c.isFlagSet(FlagCarry) {
		computed |= types.Bit0
	}

	c.setFlags(computed == 0, false, false, n&types.Bit7 == types.Bit7)
	return computed
}

// rotateLeftCarryAccumulator rotates the accumulator left by 1 bit. The most
// significant bit is copied to both the carry flag and the least significant
// bit.
//
//	RLCA
//
// Flags affected:
//
//	Z - Reset.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 7 data.
func (c *CPU) rotateLeftCarryAccumulator() {
	carry := c.A & types.Bit7
	c.A = c.A<<1 | carry>>7
	c.setFlags(false, false, false, carry == types.Bit7)
}

// rotateLeftAccumulatorThroughCarry rotates the accumulator left by 1 bit. The
// carry flag is copied to the least significant bit, and the most significant
// bit is copied to the carry flag.
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
	carry := c.A & types.Bit7
	c.A <<= 1
	if c.isFlagSet(FlagCarry) {
		c.A |= types.Bit0
	}
	c.setFlags(false, false, false, carry == types.Bit7)
}

// rotateRightAccumulator rotates the accumulator right by 1 bit. The least
// significant bit is copied to both the carry flag and the most significant
// bit.
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
	carry := c.A & types.Bit0
	c.A = c.A>>1 | carry<<7
	c.setFlags(false, false, false, carry == types.Bit0)
}

// rotateRightAccumulatorThroughCarry rotates the accumulator right by 1 bit.
// The carry flag is copied to the most significant bit, and the least significant
// bit is copied to the carry flag.
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
	carry := c.A&types.Bit0 == types.Bit0
	c.A >>= 1
	if c.isFlagSet(FlagCarry) {
		c.A |= types.Bit7
	}

	c.setFlags(false, false, false, carry)
}

func init() {
	DefineInstruction(0x07, "RLCA", func(c *CPU) { c.rotateLeftCarryAccumulator() })
	DefineInstruction(0x0F, "RRCA", func(c *CPU) { c.rotateRightAccumulator() })
	DefineInstruction(0x17, "RLA", func(c *CPU) { c.rotateLeftAccumulatorThroughCarry() })
	DefineInstruction(0x1F, "RRA", func(c *CPU) { c.rotateRightAccumulatorThroughCarry() })
}

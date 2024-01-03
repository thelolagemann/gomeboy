package cpu

import (
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/internal/types"
)

// and performs a bitwise AND operation on n and the A Register.
//
//	AND n
//	n = d8, B, C, D, E, H, L, (HL), A
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Set.
//	C - Reset.
func (c *CPU) and(n uint8) {
	c.A &= n
	c.setFlags(c.A == 0, false, true, false)
}

// or performs a bitwise OR operation on n and the A Register.
//
//	OR n
//	n = d8, B, C, D, E, H, L, (HL), A
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Reset.
func (c *CPU) or(n uint8) {
	c.A |= n
	c.setFlags(c.A == 0, false, false, false)
}

// xor performs a bitwise XOR operation on n and the A Register.
//
//	XOR n
//	n = d8, B, C, D, E, H, L, (HL), A
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Reset.
func (c *CPU) xor(n uint8) {
	c.A ^= n
	c.setFlags(c.A == 0, false, false, false)
}

// compare compares n to the A Register.
//
//	CP n
//	n = d8, B, C, D, E, H, L, (HL), A
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Set.
//	H - Set if no borrow from bit 4.
//	C - Set if no borrow.
func (c *CPU) compare(n uint8) {
	c.setFlags(c.A-n == 0, true, n&0x0f > c.A&0x0f, n > c.A)
}

// swap the upper and lower nibbles of a byte
//
// SWAP n
// n = A, B, C, D, E, H, L, (HL)=
//
// Flags affected:
// Z - Set if result is zero.
// N - Reset.
// H - Reset.
// C - Reset.
func (c *CPU) swap(value uint8) uint8 {
	c.setFlags(value == 0, false, false, false)
	return value<<4 | value>>4
}

// testBit tests the bit at the given position in the given Register.
//
//	BIT n, r
//	n = 0-7
//	r = A, B, C, D, E, H, L, (HL)
//
// Flags affected:
//
//	Z - Set if bit n of Register r is 0.
//	N - Reset.
//	H - Set.
//	C - Not affected.
func (c *CPU) testBit(value uint8, b types.Bit) {
	c.setFlags(value&b != b, false, true, c.isFlagSet(flagCarry))
}

// increment n by 1 and set the flags accordingly.
//
//	INC n
//	n = 8-bit value
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Set if carry from lower nibble.
//	C - Not affected.
func (c *CPU) increment(n uint8) uint8 {
	incremented := n + 0x01
	c.setFlags(incremented == 0, false, n&0xF == 0xF, c.isFlagSet(flagCarry))
	return incremented
}

// incrementNN increments the given RegisterPair by 1.
//
//	INC nn
//	nn = AF, BC, DE, HL
//
// Flags affected:
//
//	Z - Not affected.
//	N - Not affected.
//	H - Not affected.
//	C - Not affected.
func (c *CPU) incrementNN(register *RegisterPair) {
	c.handleOAMCorruption(register.Uint16())
	register.SetUint16(register.Uint16() + 1)

	c.s.Tick(4)
}

// decrement n by 1 and set the flags accordingly.
//
//	DEC n
//	n = 8-bit value
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Set.
//	H - Set if carry from bit 3.
//	C - Not affected.
func (c *CPU) decrement(n uint8) uint8 {
	decremented := n - 0x01
	c.setFlags(decremented == 0, true, n&0xF == 0x0, c.isFlagSet(flagCarry))
	return decremented
}

// decrementNN decrements the given RegisterPair by 1.
//
//	DEC nn
//	nn = AF, BC, DE, HL
//
// Flags affected:
//
//	Z - Not affected.
//	N - Not affected.
//	H - Not affected.
//	C - Not affected.
func (c *CPU) decrementNN(register *RegisterPair) {
	if register.Uint16() >= 0xFE00 && register.Uint16() <= 0xFEFF && c.b.Get(types.STAT)&0b11 == ppu.ModeOAM {
		// TODO
		// get the current cycle of mode 2 that the PPU is in
		// the oam is split into 20 rows of 8 bytes each, with
		// each row taking 1 M-cycle to read
		// so we need to figure out which row we're in
		// and then perform the oam corruption
		c.ppu.WriteCorruptionOAM()
	}
	register.SetUint16(register.Uint16() - 1)
	c.s.Tick(4)
}

// addHLRR adds the given RegisterPair to the HL RegisterPair.
//
//	ADD HL, nn
//	nn = AF, BC, DE, HL
//
// Flags affected:
//
//	Z - Not affected.
//	N - Reset.
//	H - Set if carry from bit 11.
//	C - Set if carry from bit 15.
func (c *CPU) addHLRR(register *RegisterPair) {
	c.HL.SetUint16(c.addUint16(c.HL.Uint16(), register.Uint16()))
	c.s.Tick(4)
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
func (c *CPU) add(n uint8, shouldCarry bool) {
	newCarry := c.isFlagSet(flagCarry) && shouldCarry
	sum := uint16(c.A) + uint16(n)
	sumHalf := (c.A & 0xF) + (n & 0xF)
	if newCarry {
		sum++
		sumHalf++
	}
	c.setFlags(uint8(sum) == 0, false, sumHalf > 0xF, sum > 0xFF)
	c.A = uint8(sum)
}

// addUint16 is a helper function for adding two uint16 values together and
// setting the flags accordingly.
//
// Used by:
//
//	ADD HL, nn
//
// Flags affected:
//
//	Z - Not affected.
//	N - Reset.
//	H - Set if carry from bit 11.
//	C - Set if carry from bit 15.
func (c *CPU) addUint16(a, b uint16) uint16 {
	sum := uint32(a) + uint32(b)
	c.setFlags(c.isFlagSet(flagZero), false, (a&0xFFF)+(b&0xFFF) > 0xFFF, sum > 0xFFFF)
	return uint16(sum)
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
func (c *CPU) sub(n uint8, shouldCarry bool) {
	newCarry := c.isFlagSet(flagCarry) && shouldCarry
	sub := int16(c.A) - int16(n)
	subHalf := int16(c.A&0xF) - int16(n&0xF)
	if newCarry {
		sub--
		subHalf--
	}

	c.setFlags(uint8(sub) == 0, true, subHalf < 0, sub < 0)
	c.A = uint8(sub)
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
	c.s.Tick(4)
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
	*l = c.b.ClockedRead(c.SP)
	c.SP++

	if c.SP >= 0xFE00 && c.SP <= 0xFEFF && c.b.Get(types.STAT)&0b11 == ppu.ModeOAM {
		c.ppu.WriteCorruptionOAM()
	}

	*h = c.b.ClockedRead(c.SP)
	c.SP++
}

func (c *CPU) addSPSigned() uint16 {
	value := c.readOperand()
	result := uint16(int32(c.SP) + int32(int8(value)))

	tmpVal := c.SP ^ uint16(int8(value)) ^ result

	c.setFlags(false, false, tmpVal&0x10 == 0x10, tmpVal&0x100 == 0x100)

	c.s.Tick(4)
	return result
}

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
	if c.isFlagSet(flagCarry) {
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
	if c.isFlagSet(flagCarry) {
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
	if c.isFlagSet(flagCarry) {
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
	if c.isFlagSet(flagCarry) {
		c.A |= types.Bit7
	}

	c.setFlags(false, false, false, carry)
}

// shiftLeftArithmetic shifts n left by one bit, and sets the carry flag to the
// most significant bit of n.
//
//	SLA n
//	n = B, C, D, E, H, L, (HL), A
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 7 data.
func (c *CPU) shiftLeftArithmetic(n uint8) uint8 {
	computed := n << 1
	c.setFlags(computed == 0, false, false, n&types.Bit7 == types.Bit7)
	return computed
}

// shiftRightArithmetic shifts n right by one bit and sets the carry flag to the
// least significant bit of n. The most significant bit does not change.
//
//	SRA n
//	n = B, C, D, E, H, L, (HL), A
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) shiftRightArithmetic(n uint8) uint8 {
	computed := n>>1 | n&types.Bit7
	c.setFlags(computed == 0, false, false, n&types.Bit0 == types.Bit0)
	return computed
}

// shiftRightLogical shifts n right one bit and sets the carry flag to the
// least significant bit of n.
//
//	SRL n
//	n = B, C, D, E, H, L, (HL), A
//
// Flags affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Contains old bit 0 data.
func (c *CPU) shiftRightLogical(n uint8) uint8 {
	computed := n >> 1
	c.setFlags(computed == 0, false, false, n&types.Bit0 == types.Bit0)

	return computed
}

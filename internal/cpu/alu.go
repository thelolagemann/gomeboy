package cpu

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

func (c *CPU) addSPSigned() uint16 {
	value := c.readOperand()
	result := uint16(int32(c.SP) + int32(int8(value)))

	tmpVal := c.SP ^ uint16(int8(value)) ^ result

	c.setFlags(false, false, tmpVal&0x10 == 0x10, tmpVal&0x100 == 0x100)

	c.s.Tick(4)
	return result
}

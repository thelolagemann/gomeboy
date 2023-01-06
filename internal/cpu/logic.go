package cpu

// andRegister performs a bitwise AND operation on the given Register and the
// A Register.
//
//	AND n
//	n = A, B, C, D, E, H, L, (HL)
//
// IF affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Set.
//	C - Reset.
func (c *CPU) andRegister(reg *Register) {
	c.A = c.and(c.A, *reg)
}

// and is a helper function for that performs a bitwise AND operation on the
// two given values, and sets the flags accordingly.
func (c *CPU) and(a, b uint8) uint8 {
	c.setFlag(FlagHalfCarry)
	c.clearFlag(FlagCarry)
	c.clearFlag(FlagSubtract)
	computed := a & b
	c.shouldZeroFlag(computed)
	return computed
}

// orRegister performs a bitwise OR operation on the given Register and the A
// Register.
//
//	OR n
//	n = A, B, C, D, E, H, L, (HL)
//
// IF affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Reset.
func (c *CPU) orRegister(reg *Register) {
	c.A = c.or(c.A, *reg)
}

// or is a helper function for that performs a bitwise OR operation on the two
// given values, and sets the flags accordingly.
func (c *CPU) or(a, b uint8) uint8 {
	c.clearFlag(FlagHalfCarry)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagCarry)
	computed := a | b
	c.shouldZeroFlag(computed)
	return computed
}

// xorRegister performs a bitwise XOR operation on the given Register and the A
// Register.
//
//	XOR n
//	n = A, B, C, D, E, H, L, (HL)
//
// IF affected:
//
//	Z - Set if result is zero.
//	N - Reset.
//	H - Reset.
//	C - Reset.
func (c *CPU) xorRegister(reg *Register) {
	c.A = c.xor(c.A, *reg)
}

// xor is a helper function for that performs a bitwise XOR operation on the two
// given values, and sets the flags accordingly.
func (c *CPU) xor(a, b uint8) uint8 {
	c.clearFlag(FlagHalfCarry)
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagCarry)
	computed := a ^ b
	c.shouldZeroFlag(computed)
	return computed
}

// compareRegister compares the given Register with the A Register.
//
//	CP n
//	n = A, B, C, D, E, H, L, (HL)
//
// IF affected:
//
//	Z - Set if result is zero.
//	N - Set.
//	H - Set if no borrow from bit 4.
//	C - Set if no borrow.
func (c *CPU) compareRegister(reg *Register) {
	c.compare(*reg)
}

// compare is a helper function for that compares the two given values, and sets
// the flags accordingly.
func (c *CPU) compare(b uint8) {
	total := c.A - b

	c.setFlag(FlagSubtract)
	if b&0x0f > c.A&0x0f {
		c.setFlag(FlagHalfCarry)
	} else {
		c.clearFlag(FlagHalfCarry)
	}
	if b > c.A {
		c.setFlag(FlagCarry)
	} else {
		c.clearFlag(FlagCarry)
	}
	c.shouldZeroFlag(total)
}

// generateLogicInstructions generates the instructions for the bitwise logic
// operations.
//
// There are 8 instruction groups:
//
//	ADD A, n
//	ADC A, n
//	SUB n
//	SBC A, n
//	AND n
//	OR n
//	XOR n
//	CP n
//
// Where n is one of the following:
//
//	A, B, C, D, E, H, L, (HL), d8
func (c *CPU) generateLogicInstructions() {
	// loop through the 8 instruction groups
	for i := uint8(0); i < 8; i++ {
		// loop through the 8 registers
		for j := uint8(0); j < 8; j++ {
			// handle the special case of (HL)
			if j == 6 {
				switch i {
				case 0:
					InstructionSet[0x86] = NewInstruction("ADD A, (HL)", 1, 2, func(cpu *CPU, bytes []byte) {
						c.addN(c.mmu.Read(c.HL.Uint16()))
					})
				case 1:
					InstructionSet[0x8E] = NewInstruction("ADC A, (HL)", 1, 2, func(cpu *CPU, bytes []byte) {
						c.addNCarry(c.mmu.Read(c.HL.Uint16()))
					})
				case 2:
					InstructionSet[0x96] = NewInstruction("SUB (HL)", 1, 2, func(cpu *CPU, bytes []byte) {
						c.subtractN(c.mmu.Read(c.HL.Uint16()))
					})
				case 3:
					InstructionSet[0x9E] = NewInstruction("SBC A, (HL)", 1, 2, func(cpu *CPU, bytes []byte) {
						c.subtractNCarry(c.mmu.Read(c.HL.Uint16()))
					})
				case 4:
					InstructionSet[0xA6] = NewInstruction("AND (HL)", 1, 2, func(cpu *CPU, bytes []byte) {
						c.A = c.and(c.A, c.mmu.Read(c.HL.Uint16()))
					})
				case 5:
					InstructionSet[0xAE] = NewInstruction("XOR (HL)", 1, 2, func(cpu *CPU, bytes []byte) {
						c.A = c.xor(c.A, c.mmu.Read(c.HL.Uint16()))
					})
				case 6:
					InstructionSet[0xB6] = NewInstruction("OR (HL)", 1, 2, func(cpu *CPU, bytes []byte) {
						c.A = c.or(c.A, c.mmu.Read(c.HL.Uint16()))
					})
				case 7:
					InstructionSet[0xBE] = NewInstruction("CP (HL)", 1, 2, func(cpu *CPU, bytes []byte) {
						c.compare(c.mmu.Read(c.HL.Uint16()))
					})
				}
				continue
			}

			currentReg := j
			// generate the instruction
			switch i {
			case 0:
				InstructionSet[0x80+i*8+j] = NewInstruction("ADD A, "+c.registerName(c.registerIndex(currentReg)), 1, 1, func(c *CPU, operands []byte) {
					c.addN(*c.registerIndex(currentReg))
				})
			case 1:
				InstructionSet[0x80+i*8+j] = NewInstruction("ADC A, "+c.registerName(c.registerIndex(currentReg)), 1, 1, func(c *CPU, operands []byte) {
					c.addNCarry(*c.registerIndex(currentReg))
				})
			case 2:
				InstructionSet[0x80+i*8+j] = NewInstruction("SUB "+c.registerName(c.registerIndex(currentReg)), 1, 1, func(c *CPU, operands []byte) {
					c.subtractN(*c.registerIndex(currentReg))
				})
			case 3:
				InstructionSet[0x80+i*8+j] = NewInstruction("SBC A, "+c.registerName(c.registerIndex(currentReg)), 1, 1, func(c *CPU, operands []byte) {
					c.subtractNCarry(*c.registerIndex(currentReg))
				})
			case 4:
				InstructionSet[0x80+i*8+j] = NewInstruction("AND "+c.registerName(c.registerIndex(currentReg)), 1, 1, func(c *CPU, operands []byte) {
					c.andRegister(c.registerIndex(currentReg))
				})
			case 5:
				InstructionSet[0x80+i*8+j] = NewInstruction("XOR "+c.registerName(c.registerIndex(currentReg)), 1, 1, func(c *CPU, operands []byte) {
					c.xorRegister(c.registerIndex(currentReg))
				})
			case 6:
				InstructionSet[0x80+i*8+j] = NewInstruction("OR "+c.registerName(c.registerIndex(currentReg)), 1, 1, func(c *CPU, operands []byte) {
					c.orRegister(c.registerIndex(currentReg))
				})
			case 7:
				InstructionSet[0x80+i*8+j] = NewInstruction("CP "+c.registerName(c.registerIndex(currentReg)), 1, 1, func(c *CPU, operands []byte) {
					c.compareRegister(c.registerIndex(currentReg))
				})
			}
		}
	}
}

func init() {
	// 0xC6 - ADD A, d8
	InstructionSet[0xC6] = NewInstruction("ADD A, d8", 2, 2, func(c *CPU, operands []byte) {
		c.addN(operands[0])
	})
	// 0xCE - ADC A, d8
	InstructionSet[0xCE] = NewInstruction("ADC A, d8", 2, 2, func(c *CPU, operands []byte) {
		c.addNCarry(operands[0])
	})
	// 0xD6 - SUB d8
	InstructionSet[0xD6] = NewInstruction("SUB d8", 2, 2, func(c *CPU, operands []byte) {
		c.subtractN(operands[0])
	})
	// 0xDE - SBC A, d8
	InstructionSet[0xDE] = NewInstruction("SBC A, d8", 2, 2, func(c *CPU, operands []byte) {
		c.subtractNCarry(operands[0])
	})
	// 0xE6 - AND d8
	InstructionSet[0xE6] = NewInstruction("AND d8", 2, 2, func(c *CPU, operands []byte) {
		c.A = c.and(c.A, operands[0])
	})
	// 0xEE - XOR d8
	InstructionSet[0xEE] = NewInstruction("XOR d8", 2, 2, func(c *CPU, operands []byte) {
		c.A = c.xor(c.A, operands[0])
	})
	// 0xF6 - OR d8
	InstructionSet[0xF6] = NewInstruction("OR d8", 2, 2, func(c *CPU, operands []byte) {
		c.A = c.or(c.A, operands[0])
	})
	// 0xFE - CP d8
	InstructionSet[0xFE] = NewInstruction("CP d8", 2, 2, func(c *CPU, operands []byte) {
		c.compare(operands[0])
	})
}

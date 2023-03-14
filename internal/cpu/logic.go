package cpu

import "fmt"

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
// Flags affected:
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
// Flags affected:
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
// Flags affected:
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
func generateLogicInstructions() {
	// loop through the 8 instruction groups
	for i := uint8(0); i < 8; i++ {
		// loop through the 8 registers
		for j := uint8(0); j < 8; j++ {
			// (HL) is manually handled
			if j == 6 {
				continue
			}

			currentReg := j
			// generate the instruction
			switch i {
			case 0:
				DefineInstruction(0x80+i*8+j, fmt.Sprintf("ADD A, %s", registerNameMap[currentReg]), func(cpu *CPU) {
					cpu.A = cpu.add(cpu.A, cpu.registerIndex(currentReg), false)
				})
			case 1:
				DefineInstruction(0x80+i*8+j, fmt.Sprintf("ADC A, %s", registerNameMap[currentReg]), func(cpu *CPU) {
					cpu.A = cpu.add(cpu.A, cpu.registerIndex(currentReg), true)
				})
			case 2:
				DefineInstruction(0x80+i*8+j, fmt.Sprintf("SUB %s", registerNameMap[currentReg]), func(cpu *CPU) {
					cpu.A = cpu.sub(cpu.A, cpu.registerIndex(currentReg), false)
				})
			case 3:
				DefineInstruction(0x80+i*8+j, fmt.Sprintf("SBC A, %s", registerNameMap[currentReg]), func(cpu *CPU) {
					cpu.A = cpu.sub(cpu.A, cpu.registerIndex(currentReg), true)
				})
			case 4:
				DefineInstruction(0x80+i*8+j, fmt.Sprintf("AND %s", registerNameMap[currentReg]), func(cpu *CPU) {
					cpu.A = cpu.and(cpu.A, cpu.registerIndex(currentReg))
				})
			case 5:
				DefineInstruction(0x80+i*8+j, fmt.Sprintf("XOR %s", registerNameMap[currentReg]), func(cpu *CPU) {
					cpu.A = cpu.xor(cpu.A, cpu.registerIndex(currentReg))
				})
			case 6:
				DefineInstruction(0x80+i*8+j, fmt.Sprintf("OR %s", registerNameMap[currentReg]), func(cpu *CPU) {
					cpu.A = cpu.or(cpu.A, cpu.registerIndex(currentReg))
				})
			case 7:
				DefineInstruction(0x80+i*8+j, fmt.Sprintf("CP %s", registerNameMap[currentReg]), func(cpu *CPU) {
					cpu.compare(cpu.registerIndex(currentReg))
				})
			}
		}
	}
}

func init() {
	// Bitwise d8 instructions
	DefineInstruction(0xC6, "ADD A, d8", func(cpu *CPU) { cpu.A = cpu.add(cpu.A, cpu.readOperand(), false) })
	DefineInstruction(0xCE, "ADC A, d8", func(cpu *CPU) { cpu.A = cpu.add(cpu.A, cpu.readOperand(), true) })
	DefineInstruction(0xD6, "SUB d8", func(cpu *CPU) { cpu.A = cpu.sub(cpu.A, cpu.readOperand(), false) })
	DefineInstruction(0xDE, "SBC A, d8", func(cpu *CPU) { cpu.A = cpu.sub(cpu.A, cpu.readOperand(), true) })
	DefineInstruction(0xE6, "AND d8", func(cpu *CPU) { cpu.A = cpu.and(cpu.A, cpu.readOperand()) })
	DefineInstruction(0xEE, "XOR d8", func(cpu *CPU) { cpu.A = cpu.xor(cpu.A, cpu.readOperand()) })
	DefineInstruction(0xF6, "OR d8", func(cpu *CPU) { cpu.A = cpu.or(cpu.A, cpu.readOperand()) })
	DefineInstruction(0xFE, "CP d8", func(cpu *CPU) { cpu.compare(cpu.readOperand()) })

	// (HL) instructions
	DefineInstruction(0x86, "ADD A, (HL)", func(cpu *CPU) { cpu.A = cpu.add(cpu.A, cpu.readByte(cpu.HL.Uint16()), false) })
	DefineInstruction(0x8E, "ADC A, (HL)", func(cpu *CPU) { cpu.A = cpu.add(cpu.A, cpu.readByte(cpu.HL.Uint16()), true) })
	DefineInstruction(0x96, "SUB (HL)", func(cpu *CPU) { cpu.A = cpu.sub(cpu.A, cpu.readByte(cpu.HL.Uint16()), false) })
	DefineInstruction(0x9E, "SBC A, (HL)", func(cpu *CPU) { cpu.A = cpu.sub(cpu.A, cpu.readByte(cpu.HL.Uint16()), true) })
	DefineInstruction(0xA6, "AND (HL)", func(cpu *CPU) { cpu.A = cpu.and(cpu.A, cpu.readByte(cpu.HL.Uint16())) })
	DefineInstruction(0xAE, "XOR (HL)", func(cpu *CPU) { cpu.A = cpu.xor(cpu.A, cpu.readByte(cpu.HL.Uint16())) })
	DefineInstruction(0xB6, "OR (HL)", func(cpu *CPU) { cpu.A = cpu.or(cpu.A, cpu.readByte(cpu.HL.Uint16())) })
	DefineInstruction(0xBE, "CP (HL)", func(cpu *CPU) { cpu.compare(cpu.readByte(cpu.HL.Uint16())) })
}

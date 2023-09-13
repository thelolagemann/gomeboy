package cpu

import "fmt"

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
					cpu.add(*cpu.registerSlice[currentReg], false)
				})
			case 1:
				DefineInstruction(0x80+i*8+j, fmt.Sprintf("ADC A, %s", registerNameMap[currentReg]), func(cpu *CPU) {
					cpu.add(*cpu.registerSlice[currentReg], true)
				})
			case 2:
				DefineInstruction(0x80+i*8+j, fmt.Sprintf("SUB %s", registerNameMap[currentReg]), func(cpu *CPU) {
					cpu.sub(*cpu.registerSlice[currentReg], false)
				})
			case 3:
				DefineInstruction(0x80+i*8+j, fmt.Sprintf("SBC A, %s", registerNameMap[currentReg]), func(cpu *CPU) {
					cpu.sub(*cpu.registerSlice[currentReg], true)
				})
			case 4:
				DefineInstruction(0x80+i*8+j, fmt.Sprintf("AND %s", registerNameMap[currentReg]), func(cpu *CPU) {
					cpu.and(*cpu.registerSlice[currentReg])
				})
			case 5:
				DefineInstruction(0x80+i*8+j, fmt.Sprintf("XOR %s", registerNameMap[currentReg]), func(cpu *CPU) {
					cpu.xor(*cpu.registerSlice[currentReg])
				})
			case 6:
				DefineInstruction(0x80+i*8+j, fmt.Sprintf("OR %s", registerNameMap[currentReg]), func(cpu *CPU) {
					cpu.or(*cpu.registerSlice[currentReg])
				})
			case 7:
				DefineInstruction(0x80+i*8+j, fmt.Sprintf("CP %s", registerNameMap[currentReg]), func(cpu *CPU) {
					cpu.compare(*cpu.registerSlice[currentReg])
				})
			}
		}
	}
}

func init() {
	// Bitwise d8 instructions
	DefineInstruction(0xC6, "ADD A, d8", func(cpu *CPU) { cpu.add(cpu.readOperand(), false) })
	DefineInstruction(0xCE, "ADC A, d8", func(cpu *CPU) { cpu.add(cpu.readOperand(), true) })
	DefineInstruction(0xD6, "SUB d8", func(cpu *CPU) { cpu.sub(cpu.readOperand(), false) })
	DefineInstruction(0xDE, "SBC A, d8", func(cpu *CPU) { cpu.sub(cpu.readOperand(), true) })
	DefineInstruction(0xE6, "AND d8", func(cpu *CPU) { cpu.and(cpu.readOperand()) })
	DefineInstruction(0xEE, "XOR d8", func(cpu *CPU) { cpu.xor(cpu.readOperand()) })
	DefineInstruction(0xF6, "OR d8", func(cpu *CPU) { cpu.or(cpu.readOperand()) })
	DefineInstruction(0xFE, "CP d8", func(cpu *CPU) { cpu.compare(cpu.readOperand()) })

	// (HL) instructions
	DefineInstruction(0x86, "ADD A, (HL)", func(cpu *CPU) { cpu.add(cpu.readByte(cpu.HL.Uint16()), false) })
	DefineInstruction(0x8E, "ADC A, (HL)", func(cpu *CPU) { cpu.add(cpu.readByte(cpu.HL.Uint16()), true) })
	DefineInstruction(0x96, "SUB (HL)", func(cpu *CPU) { cpu.sub(cpu.readByte(cpu.HL.Uint16()), false) })
	DefineInstruction(0x9E, "SBC A, (HL)", func(cpu *CPU) { cpu.sub(cpu.readByte(cpu.HL.Uint16()), true) })
	DefineInstruction(0xA6, "AND (HL)", func(cpu *CPU) { cpu.and(cpu.readByte(cpu.HL.Uint16())) })
	DefineInstruction(0xAE, "XOR (HL)", func(cpu *CPU) { cpu.xor(cpu.readByte(cpu.HL.Uint16())) })
	DefineInstruction(0xB6, "OR (HL)", func(cpu *CPU) { cpu.or(cpu.readByte(cpu.HL.Uint16())) })
	DefineInstruction(0xBE, "CP (HL)", func(cpu *CPU) { cpu.compare(cpu.readByte(cpu.HL.Uint16())) })
}

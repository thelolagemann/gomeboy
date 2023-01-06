package cpu

import (
	"encoding/binary"
	"fmt"
)

// loadRegisterToRegister loads the value of the given Register into the given
// Register.
//
//	LD n, n
//	n = A, B, C, D, E, H, L
func (c *CPU) loadRegisterToRegister(register *Register, value *Register) {
	*register = *value
}

// loadRegister8 loads the given value into the given Register.
//
//	LD n, d8
//	n = A, B, C, D, E, H, L
//	d8 = 8-bit immediate value
func (c *CPU) loadRegister8(reg *Register, value uint8) {
	*reg = value
}

// loadMemoryToRegister loads the value at the given memory address into the
// given Register.
//
//	LD n, (HL)
//	n = A, B, C, D, E, H, L
func (c *CPU) loadMemoryToRegister(reg *Register, address uint16) {
	*reg = c.mmu.Read(address)
}

// loadRegisterToMemory loads the value of the given Register into the given
// memory address.
//
//	LD (HL), n
//	n = A, B, C, D, E, H, L
func (c *CPU) loadRegisterToMemory(reg *Register, address uint16) {
	c.mmu.Write(address, *reg)
}

// loadRegisterToHardware loads the value of the given Register into the given
// hardware address. (e.g. LD (0xFF00 + n), A)
//
//	LD (0xFF00 + n), A
//	n = B, C, D, E, H, L, 8 bit immediate value (0xFF00 + n)
func (c *CPU) loadRegisterToHardware(reg *Register, address uint8) {
	c.mmu.Write(0xFF00+uint16(address), *reg)
}

// loadNNToRegisters loads value nn into the given Registers
//
//		LD n, nn
//	 n = BC, DE, HL, SP
//	 nn = 16-bit immediate value
func (c *CPU) loadNNToRegisters(h, l *Register, operands []byte) {
	*h = operands[1]
	*l = operands[0]
}

// loadRegister16 loads the given value into the given Register pair.
//
//	LD nn, d16
//	nn = BC, DE, HL, SP
//	d16 = 16-bit immediate value
func (c *CPU) loadRegister16(reg *RegisterPair, value uint16) {
	reg.SetUint16(value)
}

// loadHLToSP loads the value of HL into SP.
//
//	LD SP, HL
func (c *CPU) loadHLToSP() {
	c.SP = c.HL.Uint16()
}

func init() {
	// 0x01 LD BC, d16 - Load 16-bit immediate value into BC, the first byte is the low byte, the second byte is the high byte
	InstructionSet[0x01] = NewInstruction("LD BC, d16", 3, 3, func(c *CPU, operands []uint8) {
		c.loadRegister16(c.BC, binary.LittleEndian.Uint16(operands))
	})
	// 0x02 LD (BC), A - Load A into memory address pointed to by BC
	InstructionSet[0x02] = NewInstruction("LD (BC), A", 1, 2, func(c *CPU, operands []uint8) {
		c.loadRegisterToMemory(&c.A, c.BC.Uint16())
	})
	// 0x06 LD B, d8 - Load 8-bit immediate value into B
	InstructionSet[0x06] = Instruction{"LD B, d8", 2, 2, func(c *CPU, operands []uint8) {
		c.loadRegister8(&c.B, operands[0])
	}}
	// 0x08 LD (a16), SP - Load SP into memory address pointed to by 16-bit immediate value (first byte is low byte, second byte is high byte)
	InstructionSet[0x08] = NewInstruction("LD (a16), SP", 3, 5, func(c *CPU, operands []uint8) {
		c.mmu.Write16(binary.LittleEndian.Uint16(operands), c.SP)
	})
	// 0x0A LD A, (BC) - Load the 8-bit value at the memory address pointed to by BC into A
	InstructionSet[0x0A] = NewInstruction("LD A, (BC)", 1, 2, func(c *CPU, operands []uint8) {
		c.loadMemoryToRegister(&c.A, c.BC.Uint16())
	})
	// 0x0E LD C, d8 - Load 8-bit immediate value into C
	InstructionSet[0x0E] = NewInstruction("LD C, d8", 2, 2, func(c *CPU, operands []uint8) {
		c.loadRegister8(&c.C, operands[0])
	})
	// 0x11 LD DE, d16 - Load 16-bit immediate value into DE, the first byte is the low byte, the second byte is the high byte
	InstructionSet[0x11] = NewInstruction("LD DE, d16", 3, 3, func(c *CPU, operands []uint8) {
		c.loadRegister16(c.DE, binary.LittleEndian.Uint16(operands))
	})
	// 0x12 LD (DE), A - Load A into memory address pointed to by DE
	InstructionSet[0x12] = NewInstruction("LD (DE), A", 1, 2, func(c *CPU, operands []uint8) {
		c.loadRegisterToMemory(&c.A, c.DE.Uint16())
	})
	// 0x16 LD D, d8 - Load 8-bit immediate value into D
	InstructionSet[0x16] = Instruction{"LD D, d8", 2, 2, func(c *CPU, operands []uint8) {
		c.loadRegister8(&c.D, operands[0])
	}}
	// 0x1A LD A, (DE) - Load the 8-bit value at the memory address pointed to by DE into A
	InstructionSet[0x1A] = NewInstruction("LD A, (DE)", 1, 2, func(c *CPU, operands []uint8) {
		c.loadMemoryToRegister(&c.A, c.DE.Uint16())
	})
	// 0x1E LD E, d8 - Load 8-bit immediate value into E
	InstructionSet[0x1E] = Instruction{"LD E, d8", 2, 2, func(c *CPU, operands []uint8) {
		c.loadRegister8(&c.E, operands[0])
	}}
	// 0x21 LD HL, d16 - Load 16-bit immediate value into HL, the first byte is the low byte, the second byte is the high byte
	InstructionSet[0x21] = NewInstruction("LD HL, d16", 3, 3, func(c *CPU, operands []uint8) {
		c.loadRegister16(c.HL, binary.LittleEndian.Uint16(operands))
	})
	// 0x22 LD (HL+), A - Load A into memory address pointed to by HL, then increment HL
	InstructionSet[0x22] = Instruction{"LD (HL+), A", 1, 2, func(c *CPU, operands []uint8) {
		c.loadRegisterToMemory(&c.A, c.HL.Uint16())
		c.incrementNN(c.HL)
	}}
	// 0x26 LD H, d8 - Load 8-bit immediate value into H
	InstructionSet[0x26] = Instruction{"LD H, d8", 2, 2, func(c *CPU, operands []uint8) {
		c.loadRegister8(&c.H, operands[0])
	}}
	// 0x2A LD A, (HL+) - Load the 8-bit value at the memory address pointed to by HL into A, then increment HL
	InstructionSet[0x2A] = Instruction{"LD A, (HL+)", 1, 2, func(c *CPU, operands []uint8) {
		c.loadMemoryToRegister(&c.A, c.HL.Uint16())
		c.incrementNN(c.HL)
	}}
	// 0x2E LD L, d8 - Load 8-bit immediate value into L
	InstructionSet[0x2E] = Instruction{"LD L, d8", 2, 2, func(c *CPU, operands []uint8) {
		c.loadRegister8(&c.L, operands[0])
	}}
	// 0x31 LD SP, d16 - Load 16-bit immediate value into SP, the first byte is the low byte, the second byte is the high byte
	InstructionSet[0x31] = Instruction{"LD SP, d16", 3, 3, func(c *CPU, operands []uint8) {
		c.SP = binary.LittleEndian.Uint16(operands)
	}}
	// 0x32 LD (HL-), A - Load A into memory address pointed to by HL, then decrement HL
	InstructionSet[0x32] = Instruction{"LD (HL-), A", 1, 2, func(c *CPU, operands []uint8) {
		c.loadRegisterToMemory(&c.A, c.HL.Uint16())
		c.decrementNN(c.HL)
	}}
	// 0x36 LD (HL), d8 - Load 8-bit immediate value into memory address pointed to by HL
	InstructionSet[0x36] = Instruction{"LD (HL), d8", 2, 3, func(c *CPU, operands []uint8) {
		c.mmu.Write(c.HL.Uint16(), operands[0])
	}}
	// 0x3A LD A, (HL-) - Load the 8-bit value at the memory address pointed to by HL into A, then decrement HL
	InstructionSet[0x3A] = Instruction{"LD A, (HL-)", 1, 2, func(c *CPU, operands []uint8) {
		c.loadMemoryToRegister(&c.A, c.HL.Uint16())
		c.decrementNN(c.HL)
	}}
	// 0x3E LD A, d8 - Load 8-bit immediate value into A
	InstructionSet[0x3E] = Instruction{"LD A, d8", 2, 2, func(c *CPU, operands []uint8) {
		c.loadRegister8(&c.A, operands[0])
	}}
	// 0xE0 LD (a8), A
	InstructionSet[0xE0] = Instruction{"LD (a8), A", 2, 3, func(c *CPU, operands []uint8) {
		c.loadRegisterToHardware(&c.A, operands[0])
	}}
	// 0xE2 LD (C), A
	InstructionSet[0xE2] = Instruction{"LD (C), A", 1, 2, func(c *CPU, operands []uint8) {
		c.loadRegisterToHardware(&c.A, c.C)
	}}
	// 0xEA LD (a16), A
	InstructionSet[0xEA] = Instruction{"LD (a16), A", 3, 4, func(c *CPU, operands []uint8) {
		c.mmu.Write(binary.LittleEndian.Uint16(operands), c.A)
	}}
	// 0xF0 LDH A, (a8)
	InstructionSet[0xF0] = Instruction{"LDH A, (a8)", 2, 3, func(c *CPU, operands []uint8) {
		c.loadMemoryToRegister(&c.A, 0xFF00+uint16(operands[0]))
	}}
	// 0xF2 LD A, (C)
	InstructionSet[0xF2] = Instruction{"LD A, (C)", 1, 2, func(c *CPU, operands []uint8) {
		c.loadMemoryToRegister(&c.A, 0xFF00+uint16(c.C))
	}}
	// 0xF8 LD HL, SP+r8 - Add the 8 bit signed operand to the stack pointer and store the result in HL
	InstructionSet[0xF8] = Instruction{"LD HL, SP+r8", 2, 3, func(c *CPU, operands []uint8) {
		total := uint16(int32(c.SP) + int32(int8(operands[0])))
		c.HL.SetUint16(total)
		tmpVal := c.SP ^ uint16(int8(operands[0])) ^ total
		if tmpVal&0x10 == 0x10 {
			c.setFlag(FlagHalfCarry)
		} else {
			c.clearFlag(FlagHalfCarry)
		}
		if tmpVal&0x100 == 0x100 {
			c.setFlag(FlagCarry)
		} else {
			c.clearFlag(FlagCarry)
		}
		c.clearFlag(FlagSubtract)
		c.clearFlag(FlagZero)
	}}
	// 0xF9 LD SP, HL
	InstructionSet[0xF9] = Instruction{"LD SP, HL", 1, 2, func(c *CPU, operands []uint8) {
		c.SP = c.HL.Uint16()
	}}
	// 0xFA LD A, (a16)
	InstructionSet[0xFA] = Instruction{"LD A, (a16)", 3, 4, func(c *CPU, operands []uint8) {
		c.loadMemoryToRegister(&c.A, binary.LittleEndian.Uint16(operands))
	}}
}

// generateLoadRegisterToRegisterInstructions generates the instructions
// for loading a register to another register. (e.g. LD B, A)
//
// The instructions are generated in the following format:
//
//	0x40 LD B, B
//	0x41 LD B, C
//	....
//	0x7F LD A, A
func (c *CPU) generateLoadRegisterToRegisterInstructions() {
	// Loop over each register
	for i := uint8(0); i < 8; i++ {
		// handle the special case of LD (HL), r
		if i == 6 {
			for j := uint8(0); j < 8; j++ {
				// skip 0x76 (HALT)
				if j == 6 {
					continue
				}
				fromRegister := j
				InstructionSet[0x70+j] = Instruction{fmt.Sprintf("LD (HL), %s", c.registerName(c.registerIndex(fromRegister))), 1, 2, func(c *CPU, operands []uint8) {
					c.loadRegisterToMemory(c.registerIndex(fromRegister), c.HL.Uint16())
				}}
			}
			continue
		}

		// Loop over each register again
		for j := uint8(0); j < 8; j++ {

			// get the register to load to (needs to be nested in the inner loop otherwise it will always be the last register)
			toRegister := i
			// if j is 6, then we are loading from memory
			if j == 6 {
				InstructionSet[0x40+(i*8)+j] = Instruction{
					fmt.Sprintf("LD %s, (HL)", c.registerName(c.registerIndex(toRegister))),
					1,
					2,
					func(c *CPU, operands []uint8) {
						c.loadMemoryToRegister(c.registerIndex(toRegister), c.HL.Uint16())
					},
				}
			} else {
				// get the register to load from
				fromRegister := j
				// Generate the instruction
				InstructionSet[0x40+(i*8)+j] = Instruction{
					fmt.Sprintf("LD %s, %s", c.registerName(c.registerIndex(toRegister)), c.registerName(c.registerIndex(fromRegister))),
					1, 1, func(c *CPU, operands []uint8) {
						c.loadRegisterToRegister(c.registerIndex(toRegister), c.registerIndex(fromRegister))
					},
				}
			}

		}
	}
}

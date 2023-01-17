package cpu

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/pkg/utils"
)

type InstructionCB struct {
	Name   string
	Cycles uint8
	fn     func(cpu *CPU)
}

// Instruction returns the Instruction for the given opcode.
func (i InstructionCB) Instruction() Instruction {
	return Instruction{
		name:   i.Name,
		cycles: i.Cycles,
		fn:     func(cpu *CPU) { i.fn(cpu) },
		length: 1,
	}
}

var InstructionSetCB = map[uint8]InstructionCB{
	// 0x00 - RLC B
	0x00: {"RLC B", 2, func(cpu *CPU) {
		cpu.B = cpu.rotateLeft(cpu.B)
	}},
	// 0x01 - RLC C
	0x01: {"RLC C", 2, func(cpu *CPU) {
		cpu.C = cpu.rotateLeft(cpu.C)
	}},
	// 0x02 - RLC D
	0x02: {"RLC D", 2, func(cpu *CPU) {
		cpu.D = cpu.rotateLeft(cpu.D)
	}},
	// 0x03 - RLC E
	0x03: {"RLC E", 2, func(cpu *CPU) {
		cpu.E = cpu.rotateLeft(cpu.E)
	}},
	// 0x04 - RLC H
	0x04: {"RLC H", 2, func(cpu *CPU) {
		cpu.H = cpu.rotateLeft(cpu.H)
	}},
	// 0x05 - RLC L
	0x05: {"RLC L", 2, func(cpu *CPU) {
		cpu.L = cpu.rotateLeft(cpu.L)
	}},
	// 0x06 - RLC (HL)
	0x06: {"RLC (HL)", 4, func(cpu *CPU) {
		cpu.mmu.Write(cpu.HL.Uint16(), cpu.rotateLeft(cpu.mmu.Read(cpu.HL.Uint16())))
	}},
	// 0x07 - RLC A
	0x07: {"RLC A", 2, func(cpu *CPU) {
		cpu.A = cpu.rotateLeft(cpu.A)
	}},
	// 0x08 - RRC B
	0x08: {"RRC B", 2, func(cpu *CPU) {
		cpu.B = cpu.rotateRight(cpu.B)
	}},
	// 0x09 - RRC C
	0x09: {"RRC C", 2, func(cpu *CPU) {
		cpu.C = cpu.rotateRight(cpu.C)
	}},
	// 0x0A - RRC D
	0x0A: {"RRC D", 2, func(cpu *CPU) {
		cpu.D = cpu.rotateRight(cpu.D)
	}},
	// 0x0B - RRC E
	0x0B: {"RRC E", 2, func(cpu *CPU) {
		cpu.E = cpu.rotateRight(cpu.E)
	}},
	// 0x0C - RRC H
	0x0C: {"RRC H", 2, func(cpu *CPU) {
		cpu.H = cpu.rotateRight(cpu.H)
	}},
	// 0x0D - RRC L
	0x0D: {"RRC L", 2, func(cpu *CPU) {
		cpu.L = cpu.rotateRight(cpu.L)
	}},
	// 0x0E - RRC (HL)
	0x0E: {"RRC (HL)", 4, func(cpu *CPU) {
		cpu.mmu.Write(cpu.HL.Uint16(), cpu.rotateRight(cpu.mmu.Read(cpu.HL.Uint16())))
	}},
	// 0x0F - RRC A
	0x0F: {"RRC A", 2, func(cpu *CPU) {
		cpu.A = cpu.rotateRight(cpu.A)
	}},
	// 0x10 - RL B
	0x10: {"RL B", 2, func(cpu *CPU) {
		cpu.B = cpu.rotateLeftThroughCarry(cpu.B)
	}},
	// 0x11 - RL C
	0x11: {"RL C", 2, func(cpu *CPU) {
		cpu.C = cpu.rotateLeftThroughCarry(cpu.C)
	}},
	// 0x12 - RL D
	0x12: {"RL D", 2, func(cpu *CPU) {
		cpu.D = cpu.rotateLeftThroughCarry(cpu.D)
	}},
	// 0x13 - RL E
	0x13: {"RL E", 2, func(cpu *CPU) {
		cpu.E = cpu.rotateLeftThroughCarry(cpu.E)
	}},
	// 0x14 - RL H
	0x14: {"RL H", 2, func(cpu *CPU) {
		cpu.H = cpu.rotateLeftThroughCarry(cpu.H)
	}},
	// 0x15 - RL L
	0x15: {"RL L", 2, func(cpu *CPU) {
		cpu.L = cpu.rotateLeftThroughCarry(cpu.L)
	}},
	// 0x16 - RL (HL)
	0x16: {"RL (HL)", 4, func(cpu *CPU) {
		cpu.mmu.Write(cpu.HL.Uint16(), cpu.rotateLeftThroughCarry(cpu.mmu.Read(cpu.HL.Uint16())))
	}},
	// 0x17 - RL A
	0x17: {"RL A", 2, func(cpu *CPU) {
		cpu.A = cpu.rotateLeftThroughCarry(cpu.A)
	}},
	// 0x18 - RR B
	0x18: {"RR B", 2, func(cpu *CPU) {
		cpu.B = cpu.rotateRightThroughCarry(cpu.B)
	}},
	// 0x19 - RR C
	0x19: {"RR C", 2, func(cpu *CPU) {
		cpu.C = cpu.rotateRightThroughCarry(cpu.C)
	}},
	// 0x1A - RR D
	0x1A: {"RR D", 2, func(cpu *CPU) {
		cpu.D = cpu.rotateRightThroughCarry(cpu.D)
	}},
	// 0x1B - RR E
	0x1B: {"RR E", 2, func(cpu *CPU) {
		cpu.E = cpu.rotateRightThroughCarry(cpu.E)
	}},
	// 0x1C - RR H
	0x1C: {"RR H", 2, func(cpu *CPU) {
		cpu.H = cpu.rotateRightThroughCarry(cpu.H)
	}},
	// 0x1D - RR L
	0x1D: {"RR L", 2, func(cpu *CPU) {
		cpu.L = cpu.rotateRightThroughCarry(cpu.L)
	}},
	// 0x1E - RR (HL)
	0x1E: {"RR (HL)", 4, func(cpu *CPU) {
		cpu.mmu.Write(cpu.HL.Uint16(), cpu.rotateRightThroughCarry(cpu.mmu.Read(cpu.HL.Uint16())))
	}},
	// 0x1F - RR A
	0x1F: {"RR A", 2, func(cpu *CPU) {
		cpu.A = cpu.rotateRightThroughCarry(cpu.A)
	}},
	// 0x20 - SLA B
	0x20: {"SLA B", 2, func(cpu *CPU) {
		cpu.B = cpu.shiftLeftIntoCarry(cpu.B)
	}},
	// 0x21 - SLA C
	0x21: {"SLA C", 2, func(cpu *CPU) {
		cpu.C = cpu.shiftLeftIntoCarry(cpu.C)
	}},
	// 0x22 - SLA D
	0x22: {"SLA D", 2, func(cpu *CPU) {
		cpu.D = cpu.shiftLeftIntoCarry(cpu.D)
	}},
	// 0x23 - SLA E
	0x23: {"SLA E", 2, func(cpu *CPU) {
		cpu.E = cpu.shiftLeftIntoCarry(cpu.E)
	}},
	// 0x24 - SLA H
	0x24: {"SLA H", 2, func(cpu *CPU) {
		cpu.H = cpu.shiftLeftIntoCarry(cpu.H)
	}},
	// 0x25 - SLA L
	0x25: {"SLA L", 2, func(cpu *CPU) {
		cpu.L = cpu.shiftLeftIntoCarry(cpu.L)
	}},
	// 0x26 - SLA (HL)
	0x26: {"SLA (HL)", 4, func(cpu *CPU) {
		cpu.mmu.Write(cpu.HL.Uint16(), cpu.shiftLeftIntoCarry(cpu.mmu.Read(cpu.HL.Uint16())))
	}},
	// 0x27 - SLA A
	0x27: {"SLA A", 2, func(cpu *CPU) {
		cpu.A = cpu.shiftLeftIntoCarry(cpu.A)
	}},
	// 0x28 - SRA B
	0x28: {"SRA B", 2, func(cpu *CPU) {
		cpu.B = cpu.shiftRightIntoCarry(cpu.B)
	}},
	// 0x29 - SRA C
	0x29: {"SRA C", 2, func(cpu *CPU) {
		cpu.C = cpu.shiftRightIntoCarry(cpu.C)
	}},
	// 0x2A - SRA D
	0x2A: {"SRA D", 2, func(cpu *CPU) {
		cpu.D = cpu.shiftRightIntoCarry(cpu.D)
	}},
	// 0x2B - SRA E
	0x2B: {"SRA E", 2, func(cpu *CPU) {
		cpu.E = cpu.shiftRightIntoCarry(cpu.E)
	}},
	// 0x2C - SRA H
	0x2C: {"SRA H", 2, func(cpu *CPU) {
		cpu.H = cpu.shiftRightIntoCarry(cpu.H)
	}},
	// 0x2D - SRA L
	0x2D: {"SRA L", 2, func(cpu *CPU) {
		cpu.L = cpu.shiftRightIntoCarry(cpu.L)
	}},
	// 0x2E - SRA (HL)
	0x2E: {"SRA (HL)", 4, func(cpu *CPU) {
		cpu.mmu.Write(cpu.HL.Uint16(), cpu.shiftRightIntoCarry(cpu.mmu.Read(cpu.HL.Uint16())))
	}},
	// 0x2F - SRA A
	0x2F: {"SRA A", 2, func(cpu *CPU) {
		cpu.A = cpu.shiftRightIntoCarry(cpu.A)
	}},
	// 0x30 - SWAP B
	0x30: {"SWAP B", 2, func(cpu *CPU) {
		cpu.B = cpu.swap(cpu.B)
	}},
	// 0x31 - SWAP C
	0x31: {"SWAP C", 2, func(cpu *CPU) {
		cpu.C = cpu.swap(cpu.C)
	}},
	// 0x32 - SWAP D
	0x32: {"SWAP D", 2, func(cpu *CPU) {
		cpu.D = cpu.swap(cpu.D)
	}},
	// 0x33 - SWAP E
	0x33: {"SWAP E", 2, func(cpu *CPU) {
		cpu.E = cpu.swap(cpu.E)
	}},
	// 0x34 - SWAP H
	0x34: {"SWAP H", 2, func(cpu *CPU) {
		cpu.H = cpu.swap(cpu.H)
	}},
	// 0x35 - SWAP L
	0x35: {"SWAP L", 2, func(cpu *CPU) {
		cpu.L = cpu.swap(cpu.L)
	}},
	// 0x36 - SWAP (HL)
	0x36: {"SWAP (HL)", 4, func(cpu *CPU) {
		cpu.mmu.Write(cpu.HL.Uint16(), cpu.swap(cpu.mmu.Read(cpu.HL.Uint16())))
	}},
	// 0x37 - SWAP A
	0x37: {"SWAP A", 2, func(cpu *CPU) {
		cpu.A = cpu.swap(cpu.A)
	}},
	// 0x38 - SRL B
	0x38: {"SRL B", 2, func(cpu *CPU) {
		cpu.B = cpu.shiftRightLogical(cpu.B)
	}},
	// 0x39 - SRL C
	0x39: {"SRL C", 2, func(cpu *CPU) {
		cpu.C = cpu.shiftRightLogical(cpu.C)
	}},
	// 0x3A - SRL D
	0x3A: {"SRL D", 2, func(cpu *CPU) {
		cpu.D = cpu.shiftRightLogical(cpu.D)
	}},
	// 0x3B - SRL E
	0x3B: {"SRL E", 2, func(cpu *CPU) {
		cpu.E = cpu.shiftRightLogical(cpu.E)
	}},
	// 0x3C - SRL H
	0x3C: {"SRL H", 2, func(cpu *CPU) {
		cpu.H = cpu.shiftRightLogical(cpu.H)
	}},
	// 0x3D - SRL L
	0x3D: {"SRL L", 2, func(cpu *CPU) {
		cpu.L = cpu.shiftRightLogical(cpu.L)
	}},
	// 0x3E - SRL (HL)
	0x3E: {"SRL (HL)", 4, func(cpu *CPU) {
		cpu.mmu.Write(cpu.HL.Uint16(), cpu.shiftRightLogical(cpu.mmu.Read(cpu.HL.Uint16())))
	}},
	// 0x3F - SRL A
	0x3F: {"SRL A", 2, func(cpu *CPU) {
		cpu.A = cpu.shiftRightLogical(cpu.A)
	}},
}

// generateBitInstructions generates the bit instructions
// for the InstructionSetCB map, BIT, RES, and SET.
//
// The instructions are generated in the form of;
//
//	0x40 - BIT 0, B
//	0x41 - BIT 0, C
//	...
//	0xFF - SET 7, A
func (c *CPU) generateBitInstructions() {
	// Loop through each bit
	for bit := uint8(0); bit <= 7; bit++ {
		// Loop through each register
		for reg := uint8(0); reg <= 7; reg++ {
			currentBit := bit // create a copy of the current bit as it will be changed in the outer loop when fn is called
			if reg == 6 {
				// (HL) is not a register, it's a memory address pointed to by HL,
				// so we need to handle it separately

				// BIT
				InstructionSetCB[0x40+bit*8+reg] = InstructionCB{
					Name:   fmt.Sprintf("BIT %d, (HL)", currentBit),
					Cycles: 3,
					fn: func(cpu *CPU) {
						cpu.testBit(cpu.mmu.Read(cpu.HL.Uint16()), currentBit)
					},
				}

				// RES
				InstructionSetCB[0x80+bit*8+reg] = InstructionCB{
					Name:   fmt.Sprintf("RES %d, (HL)", currentBit),
					Cycles: 4,
					fn: func(cpu *CPU) {
						cpu.mmu.Write(cpu.HL.Uint16(), utils.Reset(cpu.mmu.Read(cpu.HL.Uint16()), currentBit))
					},
				}

				// SET
				InstructionSetCB[0xC0+bit*8+reg] = InstructionCB{
					Name:   fmt.Sprintf("SET %d, (HL)", bit),
					Cycles: 4,
					fn: func(cpu *CPU) {
						cpu.mmu.Write(cpu.HL.Uint16(), utils.Set(cpu.mmu.Read(cpu.HL.Uint16()), currentBit))
					},
				}
				continue
			}

			// Get register from index
			register := c.registerIndex(reg)

			// Create BIT instruction
			InstructionSetCB[0x40+(bit*8)+reg] = InstructionCB{
				Name:   fmt.Sprintf("BIT %d, %s", bit, c.registerName(register)),
				Cycles: 2,
				fn: func(cpu *CPU) {
					cpu.testBit(*register, currentBit)
				},
			}

			// Create RES instruction
			InstructionSetCB[0x80+bit*8+reg] = InstructionCB{
				Name:   fmt.Sprintf("RES %d, %s", bit, c.registerName(register)),
				Cycles: 2,
				fn: func(cpu *CPU) {
					*register = utils.Reset(*register, currentBit)
				},
			}
			// Create SET instruction
			InstructionSetCB[0xC0+bit*8+reg] = InstructionCB{
				Name:   fmt.Sprintf("SET %d, %s", bit, c.registerName(register)),
				Cycles: 2,
				fn: func(cpu *CPU) {
					*register = utils.Set(*register, currentBit)
				},
			}
		}
	}
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
	computed := value<<4&0xF0 | value>>4
	c.clearFlag(FlagSubtract)
	c.clearFlag(FlagHalfCarry)
	c.clearFlag(FlagCarry)
	c.shouldZeroFlag(computed)
	return computed
}

// testBit tests the bit at the given position in the given Register.
//
//	Bit n, r
//	n = 0-7
//	r = A, B, C, D, E, H, L, (HL)
//
// Flags affected:
//
//	Z - Set if bit n of Register r is 0.
//	N - Reset.
//	H - Set.
//	C - Not affected.
func (c *CPU) testBit(value uint8, position uint8) {
	c.shouldZeroFlag((value >> position) & 0x01)
	c.clearFlag(FlagSubtract)
	c.setFlag(FlagHalfCarry)
}

// debugInstructions prints the InstructionSetCB map to the console.
func (c *CPU) debugInstructions() {
	keys := make([]uint8, 0, len(InstructionSet))
	for k := range InstructionSet {
		keys = append(keys, k)
	}
	for key := range keys {
		fmt.Printf("%#x - %s - %d cycles\n", key, InstructionSet[uint8(key)].Name(), InstructionSet[uint8(key)].Cycles())
	}
	fmt.Printf("%v\n", len(InstructionSet))
	fmt.Printf("%v\n", len(InstructionSetCB))
}

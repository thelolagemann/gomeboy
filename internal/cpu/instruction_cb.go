package cpu

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/pkg/utils"
)

// TODO update to new instruction format
var InstructionSetCB = [256]Instruction{}

func (c *CPU) generateRotateInstructions() {
	// loop through each register (B, C, D, E, H, L, (HL), A)
	for j := uint8(0); j < 8; j++ {
		// (HL) needs to be handled differently as it is a memory address
		if j == 6 {
			// 0x06 - RLC (HL)
			DefineInstructionCB(0x06, "RLC (HL)", func(cpu *CPU) {
				cpu.writeByte(
					cpu.HL.Uint16(),
					cpu.rotateLeft(cpu.readByte(cpu.HL.Uint16())),
				)
			})
			// 0x0E - RRC (HL)
			DefineInstructionCB(0x0E, "RRC (HL)", func(cpu *CPU) {
				cpu.writeByte(
					cpu.HL.Uint16(),
					cpu.rotateRight(cpu.readByte(cpu.HL.Uint16())),
				)
			})
			// 0x16 - RL (HL)
			DefineInstructionCB(0x16, "RL (HL)", func(cpu *CPU) {
				cpu.writeByte(
					cpu.HL.Uint16(),
					cpu.rotateLeftThroughCarry(cpu.readByte(cpu.HL.Uint16())),
				)
			})
			// 0x1E - RR (HL)
			DefineInstructionCB(0x1E, "RR (HL)", func(cpu *CPU) {
				cpu.writeByte(
					cpu.HL.Uint16(),
					cpu.rotateRightThroughCarry(cpu.readByte(cpu.HL.Uint16())),
				)
			})
			continue
		}

		// get register from index
		reg := c.registerIndex(j)

		// create the 4 rotate instructions for each register

		// 0x00 - 0x07 - RLC r
		DefineInstructionCB(0x00+j, fmt.Sprintf("RLC %s", c.registerName(reg)), func(cpu *CPU) {
			*reg = cpu.rotateLeft(*reg)
		})

		// 0x08 - 0x0F - RRC r
		DefineInstructionCB(0x08+j, fmt.Sprintf("RRC %s", c.registerName(reg)), func(cpu *CPU) {
			*reg = cpu.rotateRight(*reg)
		})

		// 0x10 - 0x17 - RL r
		DefineInstructionCB(0x10+j, fmt.Sprintf("RL %s", c.registerName(reg)), func(cpu *CPU) {
			*reg = cpu.rotateLeftThroughCarry(*reg)
		})

		// 0x18 - 0x1F - RR r
		DefineInstructionCB(0x18+j, fmt.Sprintf("RR %s", c.registerName(reg)), func(cpu *CPU) {
			*reg = cpu.rotateRightThroughCarry(*reg)
		})
	}
}

func (c *CPU) generateShiftInstructions() {
	// loop through each register (B, C, D, E, H, L, (HL), A)
	for j := uint8(0); j < 8; j++ {
		// (HL) needs to be handled differently as it is a memory address
		if j == 6 {
			// 0x26 - SLA (HL)
			DefineInstructionCB(0x26, "SLA (HL)", func(cpu *CPU) {
				cpu.writeByte(
					cpu.HL.Uint16(),
					cpu.shiftLeftIntoCarry(cpu.readByte(cpu.HL.Uint16())),
				)
			})
			// 0x2E - SRA (HL)
			DefineInstructionCB(0x2E, "SRA (HL)", func(cpu *CPU) {
				cpu.writeByte(
					cpu.HL.Uint16(),
					cpu.shiftRightIntoCarry(cpu.readByte(cpu.HL.Uint16())),
				)
			})
			// 0x36 - SWAP (HL)
			DefineInstructionCB(0x36, "SWAP (HL)", func(cpu *CPU) {
				cpu.writeByte(
					cpu.HL.Uint16(),
					cpu.swap(cpu.readByte(cpu.HL.Uint16())),
				)
			})
			// 0x3E - SRL (HL)
			DefineInstructionCB(0x3E, "SRL (HL)", func(cpu *CPU) {
				cpu.writeByte(
					cpu.HL.Uint16(),
					cpu.shiftRightLogical(cpu.readByte(cpu.HL.Uint16())),
				)
			})
			continue
		}

		// get register from index
		reg := c.registerIndex(j)

		// create the 4 shift instructions for each register (SLA, SRA, SWAP, SRL)

		// 0x20 - 0x27 - SLA r
		DefineInstructionCB(0x20+j, fmt.Sprintf("SLA %s", c.registerName(reg)), func(cpu *CPU) {
			*reg = cpu.shiftLeftIntoCarry(*reg)
		})

		// 0x28 - 0x2F - SRA r
		DefineInstructionCB(0x28+j, fmt.Sprintf("SRA %s", c.registerName(reg)), func(cpu *CPU) {
			*reg = cpu.shiftRightIntoCarry(*reg)
		})

		// 0x30 - 0x37 - SWAP r
		DefineInstructionCB(0x30+j, fmt.Sprintf("SWAP %s", c.registerName(reg)), func(cpu *CPU) {
			*reg = cpu.swap(*reg)
		})

		// 0x38 - 0x3F - SRL r
		DefineInstructionCB(0x38+j, fmt.Sprintf("SRL %s", c.registerName(reg)), func(cpu *CPU) {
			*reg = cpu.shiftRightLogical(*reg)
		})
	}
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
				DefineInstructionCB(0x40+bit*8+reg, fmt.Sprintf("BIT %d, (HL)", bit), func(cpu *CPU) {
					cpu.testBit(cpu.readByte(cpu.HL.Uint16()), currentBit)
				})

				// RES
				DefineInstructionCB(0x80+bit*8+reg, fmt.Sprintf("RES %d, (HL)", bit), func(cpu *CPU) {
					cpu.writeByte(
						cpu.HL.Uint16(),
						utils.Reset(cpu.readByte(cpu.HL.Uint16()), currentBit),
					)
				})

				// SET
				DefineInstructionCB(0xC0+bit*8+reg, fmt.Sprintf("SET %d, (HL)", bit), func(cpu *CPU) {
					cpu.writeByte(
						cpu.HL.Uint16(),
						utils.Set(cpu.readByte(cpu.HL.Uint16()), currentBit),
					)
				})
				continue
			}

			// Get register from index
			register := c.registerIndex(reg)

			// Create BIT instruction
			DefineInstructionCB(0x40+bit*8+reg, fmt.Sprintf("BIT %d, %s", bit, c.registerName(register)), func(cpu *CPU) {
				cpu.testBit(*register, currentBit)
			})

			// Create RES instruction
			DefineInstructionCB(0x80+bit*8+reg, fmt.Sprintf("RES %d, %s", bit, c.registerName(register)), func(cpu *CPU) {
				*register = utils.Reset(*register, currentBit)
			})
			// Create SET instruction
			DefineInstructionCB(0xC0+bit*8+reg, fmt.Sprintf("SET %d, %s", bit, c.registerName(register)), func(cpu *CPU) {
				*register = utils.Set(*register, currentBit)
			})
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

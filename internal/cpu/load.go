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
	DefineInstruction(0x01, "LD BC, d16", func(c *CPU, operands []uint8) {
		c.loadRegister16(c.BC, binary.LittleEndian.Uint16(operands))
	}, Length(3), Cycles(3))
	DefineInstruction(0x02, "LD (BC), A", func(c *CPU) { c.loadRegisterToMemory(&c.A, c.BC.Uint16()) }, Cycles(2))
	DefineInstruction(0x06, "LD B, d8", func(c *CPU, operands []uint8) { c.loadRegister8(&c.B, operands[0]) }, Length(2), Cycles(2))
	DefineInstruction(0x08, "LD (a16), SP", func(c *CPU, operands []uint8) {
		c.mmu.Write16(binary.LittleEndian.Uint16(operands), c.SP)
	}, Length(3), Cycles(5))
	DefineInstruction(0x0A, "LD A, (BC)", func(c *CPU) { c.loadMemoryToRegister(&c.A, c.BC.Uint16()) }, Cycles(2))
	DefineInstruction(0x0E, "LD C, d8", func(c *CPU, operands []uint8) { c.loadRegister8(&c.C, operands[0]) }, Length(2), Cycles(2))
	DefineInstruction(0x11, "LD DE, d16", func(c *CPU, operands []uint8) {
		c.loadRegister16(c.DE, binary.LittleEndian.Uint16(operands))
	}, Length(3), Cycles(3))
	DefineInstruction(0x12, "LD (DE), A", func(c *CPU) { c.loadRegisterToMemory(&c.A, c.DE.Uint16()) }, Cycles(2))
	DefineInstruction(0x16, "LD D, d8", func(c *CPU, operands []uint8) { c.loadRegister8(&c.D, operands[0]) }, Length(2), Cycles(2))
	DefineInstruction(0x1A, "LD A, (DE)", func(c *CPU) { c.loadMemoryToRegister(&c.A, c.DE.Uint16()) }, Cycles(2))
	DefineInstruction(0x1E, "LD E, d8", func(c *CPU, operands []uint8) { c.loadRegister8(&c.E, operands[0]) }, Length(2), Cycles(2))
	DefineInstruction(0x21, "LD HL, d16", func(c *CPU, operands []uint8) {
		c.loadRegister16(c.HL, binary.LittleEndian.Uint16(operands))
	}, Length(3), Cycles(3))
	DefineInstruction(0x22, "LD (HL+), A", func(c *CPU) {
		c.loadRegisterToMemory(&c.A, c.HL.Uint16())
		c.HL.SetUint16(c.HL.Uint16() + 1)
	}, Cycles(2))
	DefineInstruction(0x26, "LD H, d8", func(c *CPU, operands []uint8) { c.loadRegister8(&c.H, operands[0]) }, Length(2), Cycles(2))
	DefineInstruction(0x2A, "LD A, (HL+)", func(c *CPU) {
		c.loadMemoryToRegister(&c.A, c.HL.Uint16())
		c.HL.SetUint16(c.HL.Uint16() + 1)
	}, Cycles(2))
	DefineInstruction(0x2E, "LD L, d8", func(c *CPU, operands []uint8) { c.loadRegister8(&c.L, operands[0]) }, Length(2), Cycles(2))
	DefineInstruction(0x31, "LD SP, d16", func(c *CPU, operands []uint8) {
		c.SP = binary.LittleEndian.Uint16(operands)
	}, Length(3), Cycles(3))
	DefineInstruction(0x32, "LD (HL-), A", func(c *CPU) {
		c.loadRegisterToMemory(&c.A, c.HL.Uint16())
		c.decrementNN(c.HL)
	}, Cycles(2))
	DefineInstruction(0x36, "LD (HL), d8", func(c *CPU, operands []uint8) {
		c.mmu.Write(c.HL.Uint16(), operands[0])
	}, Length(2), Cycles(3))
	DefineInstruction(0x3A, "LD A, (HL-)", func(c *CPU) {
		c.loadMemoryToRegister(&c.A, c.HL.Uint16())
		c.decrementNN(c.HL)
	}, Cycles(2))
	DefineInstruction(0x3E, "LD A, d8", func(c *CPU, operands []uint8) { c.loadRegister8(&c.A, operands[0]) }, Length(2), Cycles(2))
	DefineInstruction(0xE0, "LDH (a8), A", func(c *CPU, operands []uint8) {
		c.loadRegisterToHardware(&c.A, operands[0])
	}, Length(2), Cycles(3))
	DefineInstruction(0xE2, "LD (C), A", func(c *CPU) { c.loadRegisterToHardware(&c.A, c.C) }, Cycles(2))
	DefineInstruction(0xEA, "LD (a16), A", func(c *CPU, operands []uint8) {
		c.loadRegisterToMemory(&c.A, binary.LittleEndian.Uint16(operands))
	}, Length(3), Cycles(4))
	DefineInstruction(0xF0, "LDH A, (a8)", func(c *CPU, operands []uint8) {
		c.loadMemoryToRegister(&c.A, uint16(0xFF00)+uint16(operands[0]))
	}, Length(2), Cycles(3))
	DefineInstruction(0xF2, "LD A, (C)", func(c *CPU) {
		c.loadMemoryToRegister(&c.A, uint16(0xFF00)+uint16(c.C))
	}, Cycles(2))
	DefineInstruction(0xF8, "LD HL, SP+r8", func(c *CPU, operands []uint8) {
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
	}, Length(2), Cycles(3))
	DefineInstruction(0xF9, "LD SP, HL", func(c *CPU) { c.SP = c.HL.Uint16() }, Cycles(2))
	DefineInstruction(0xFA, "LD A, (a16)", func(c *CPU, operands []uint8) {
		c.loadMemoryToRegister(&c.A, binary.LittleEndian.Uint16(operands))
	}, Length(3), Cycles(4))
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
				DefineInstruction(0x70+j, fmt.Sprintf("LD (HL), %s", c.registerName(c.registerIndex(fromRegister))), func(c *CPU) {
					c.loadRegisterToMemory(c.registerIndex(fromRegister), c.HL.Uint16())
				}, Cycles(2))
			}
			continue
		}

		// Loop over each register again
		for j := uint8(0); j < 8; j++ {

			// get the register to load to (needs to be nested in the inner loop otherwise it will always be the last register)
			toRegister := i
			// if j is 6, then we are loading from memory
			if j == 6 {
				DefineInstruction(0x40+i*8+j, fmt.Sprintf("LD %s, (HL)", c.registerName(c.registerIndex(toRegister))), func(c *CPU) {
					c.loadMemoryToRegister(c.registerIndex(toRegister), c.HL.Uint16())
				}, Cycles(2))
			} else {
				// get the register to load from
				fromRegister := j
				// Generate the instruction
				DefineInstruction(
					0x40+(i*8)+j,
					fmt.Sprintf("LD %s, %s", c.registerName(c.registerIndex(toRegister)), c.registerName(c.registerIndex(fromRegister))),
					func(c *CPU, operands []uint8) {
						c.loadRegisterToRegister(c.registerIndex(toRegister), c.registerIndex(fromRegister))
					})
			}
		}
	}
}

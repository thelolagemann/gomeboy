package cpu

import (
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
func (c *CPU) loadRegister8(reg *Register) {
	*reg = c.readOperand()
}

// loadMemoryToRegister loads the value at the given memory address into the
// given Register.
//
//	LD n, (HL)
//	n = A, B, C, D, E, H, L
func (c *CPU) loadMemoryToRegister(reg *Register, address uint16) {
	*reg = c.readByte(address)
}

// loadRegisterToMemory loads the value of the given Register into the given
// memory address.
//
//	LD (HL), n
//	n = A, B, C, D, E, H, L
func (c *CPU) loadRegisterToMemory(reg Register, address uint16) {
	c.writeByte(address, reg)
}

// loadRegisterToHardware loads the value of the given Register into the given
// hardware address. (e.g. LD (0xFF00 + n), A)
//
//	LD (0xFF00 + n), A
//	n = B, C, D, E, H, L, 8 bit immediate value (0xFF00 + n)
func (c *CPU) loadRegisterToHardware(reg *Register, address uint8) {
	c.writeByte(0xFF00+uint16(address), *reg)
}

// loadRegister16 loads the given value into the given Register pair.
//
//	LD nn, d16
//	nn = BC, DE, HL, SP
//	d16 = 16-bit immediate value
func (c *CPU) loadRegister16(reg *RegisterPair) {
	*reg.Low = c.readOperand()
	*reg.High = c.readOperand()
}

func init() {
	DefineInstruction(0x01, "LD BC, d16", func(c *CPU) {
		c.loadRegister16(c.BC)
	})
	DefineInstruction(0x02, "LD (BC), A", func(c *CPU) { c.loadRegisterToMemory(c.A, c.BC.Uint16()) })
	DefineInstruction(0x06, "LD B, d8", func(c *CPU) { c.loadRegister8(&c.B) })
	DefineInstruction(0x08, "LD (a16), SP", func(c *CPU) {
		low := c.readOperand()
		high := c.readOperand()

		address := uint16(high)<<8 | uint16(low)
		c.writeByte(address, uint8(c.SP&0xFF))
		c.writeByte(address+1, uint8(c.SP>>8))
	})
	DefineInstruction(0x0A, "LD A, (BC)", func(c *CPU) { c.loadMemoryToRegister(&c.A, c.BC.Uint16()) })
	DefineInstruction(0x0E, "LD C, d8", func(c *CPU) { c.loadRegister8(&c.C) })
	DefineInstruction(0x11, "LD DE, d16", func(c *CPU) {
		c.loadRegister16(c.DE)
	})
	DefineInstruction(0x12, "LD (DE), A", func(c *CPU) { c.loadRegisterToMemory(c.A, c.DE.Uint16()) })
	DefineInstruction(0x16, "LD D, d8", func(c *CPU) { c.loadRegister8(&c.D) })
	DefineInstruction(0x1A, "LD A, (DE)", func(c *CPU) { c.loadMemoryToRegister(&c.A, c.DE.Uint16()) })
	DefineInstruction(0x1E, "LD E, d8", func(c *CPU) { c.loadRegister8(&c.E) })
	DefineInstruction(0x21, "LD HL, d16", func(c *CPU) {
		c.loadRegister16(c.HL)
	})
	DefineInstruction(0x22, "LD (HL+), A", func(c *CPU) {
		c.loadRegisterToMemory(c.A, c.HL.Uint16())
		c.HL.SetUint16(c.HL.Uint16() + 1)
	})
	DefineInstruction(0x26, "LD H, d8", func(c *CPU) { c.loadRegister8(&c.H) })
	DefineInstruction(0x2A, "LD A, (HL+)", func(c *CPU) {
		c.loadMemoryToRegister(&c.A, c.HL.Uint16())
		c.HL.SetUint16(c.HL.Uint16() + 1)
	})
	DefineInstruction(0x2E, "LD L, d8", func(c *CPU) { c.loadRegister8(&c.L) })
	DefineInstruction(0x31, "LD SP, d16", func(c *CPU) {
		low := c.readOperand()
		high := c.readOperand()

		c.SP = uint16(high)<<8 | uint16(low)
	})
	DefineInstruction(0x32, "LD (HL-), A", func(c *CPU) {
		c.loadRegisterToMemory(c.A, c.HL.Uint16())
		c.HL.SetUint16(c.HL.Uint16() - 1)
	})
	DefineInstruction(0x36, "LD (HL), d8", func(c *CPU) {
		c.writeByte(c.HL.Uint16(), c.readOperand())
	})
	DefineInstruction(0x3A, "LD A, (HL-)", func(c *CPU) {
		c.loadMemoryToRegister(&c.A, c.HL.Uint16())
		c.HL.SetUint16(c.HL.Uint16() - 1)
	})
	DefineInstruction(0x3E, "LD A, d8", func(c *CPU) { c.loadRegister8(&c.A) })
	DefineInstruction(0xE0, "LDH (a8), A", func(c *CPU) {
		c.loadRegisterToHardware(&c.A, c.readOperand())
	})
	DefineInstruction(0xE2, "LD (C), A", func(c *CPU) { c.loadRegisterToHardware(&c.A, c.C) })
	DefineInstruction(0xEA, "LD (a16), A", func(c *CPU) {
		low := c.readOperand()
		high := c.readOperand()
		c.loadRegisterToMemory(c.A, uint16(high)<<8|uint16(low))
	})
	DefineInstruction(0xF0, "LDH A, (a8)", func(c *CPU) {
		address := uint16(0xff00) + uint16(c.readOperand())
		c.loadMemoryToRegister(&c.A, address)
	})
	DefineInstruction(0xF2, "LD A, (C)", func(c *CPU) {
		c.loadMemoryToRegister(&c.A, uint16(0xFF00)+uint16(c.C))
	})
	DefineInstruction(0xF8, "LD HL, SP+r8", func(c *CPU) {
		c.HL.SetUint16(c.addSPSigned())
	})
	DefineInstruction(0xF9, "LD SP, HL", func(c *CPU) { c.SP = c.HL.Uint16(); c.s.Tick(4) })
	DefineInstruction(0xFA, "LD A, (a16)", func(c *CPU) {
		low := c.readOperand()
		high := c.readOperand()
		c.loadMemoryToRegister(&c.A, uint16(high)<<8|uint16(low))
	})
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
func generateLoadRegisterToRegisterInstructions() {
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
				DefineInstruction(0x70+j, fmt.Sprintf("LD (HL), %s", registerNameMap[fromRegister]), func(c *CPU) {
					c.loadRegisterToMemory(c.registerIndex(fromRegister), c.HL.Uint16())
				})
			}
			continue
		}

		// Loop over each register again
		for j := uint8(0); j < 8; j++ {

			// get the register to load to (needs to be nested in the inner loop otherwise it will always be the last register)
			toRegister := i
			// if j is 6, then we are loading from memory
			if j == 6 {
				DefineInstruction(0x40+i*8+j, fmt.Sprintf("LD %s, (HL)", registerNameMap[toRegister]), func(c *CPU) {
					c.loadMemoryToRegister(c.registerPointer(toRegister), c.HL.Uint16())
				})
			} else {
				// get the register to load from
				fromRegister := j
				// Generate the instruction
				if toRegister == fromRegister {
					DefineInstruction(
						0x40+(i*8)+j,
						fmt.Sprintf("LD %s, %s", registerNameMap[toRegister], registerNameMap[fromRegister]),
						func(c *CPU) {
							if c.Debug {
								c.DebugBreakpoint = true
							}
						})
					continue
				}
				DefineInstruction(
					0x40+(i*8)+j,
					fmt.Sprintf("LD %s, %s", registerNameMap[toRegister], registerNameMap[fromRegister]),
					func(c *CPU) {
						c.loadRegisterToRegister(c.registerPointer(toRegister), c.registerPointer(fromRegister))
					})
			}
		}
	}
}

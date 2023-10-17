package cpu

import "github.com/thelolagemann/gomeboy/internal/types"

// loadRegisterToRegister loads the value of the given Register into the given
// Register.
//
//	LD n, n
//	n = A, B, C, D, E, H, L
func (c *CPU) loadRegisterToRegister(register *types.Register, value *types.Register) {
	*register = *value
}

// loadRegister8 loads the given value into the given Register.
//
//	LD n, d8
//	n = A, B, C, D, E, H, L
//	d8 = 8-bit immediate value
func (c *CPU) loadRegister8(reg *types.Register) {
	*reg = c.readOperand()
}

// loadMemoryToRegister loads the value at the given memory address into the
// given Register.
//
//	LD n, (HL)
//	n = A, B, C, D, E, H, L
func (c *CPU) loadMemoryToRegister(reg *types.Register, address uint16) {
	*reg = c.b.ClockedRead(address)
}

// loadRegisterToMemory loads the value of the given Register into the given
// memory address.
//
//	LD (HL), n
//	n = A, B, C, D, E, H, L
func (c *CPU) loadRegisterToMemory(reg types.Register, address uint16) {
	c.b.ClockedWrite(address, reg)
}

// loadRegisterToHardware loads the value of the given Register into the given
// hardware address. (e.g. LD (0xFF00 + n), A)
//
//	LD (0xFF00 + n), A
//	n = B, C, D, E, H, L, 8 bit immediate value (0xFF00 + n)
func (c *CPU) loadRegisterToHardware(reg types.Register, address uint8) {
	c.b.ClockedWrite(0xFF00+uint16(address), reg)
}

// loadRegister16 loads the given value into the given Register pair.
//
//	LD nn, d16
//	nn = BC, DE, HL, SP
//	d16 = 16-bit immediate value
func (c *CPU) loadRegister16(reg *types.RegisterPair) {
	*reg.Low = c.readOperand()
	*reg.High = c.readOperand()
}

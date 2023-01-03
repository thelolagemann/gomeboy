package cpu

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
	*reg = Register(value)
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

// pushRegister pushes the given Register pair onto the stack.
//
//	PUSH nn
//	nn = BC, DE, HL, AF
func (c *CPU) pushRegister(reg *RegisterPair) {
	c.push(uint8(*reg.High))
	c.push(uint8(*reg.Low))
}

// popStack loads the value at the top of the stack into the given Register
// pair.
//
//	POP nn
//	nn = BC, DE, HL, AF
func (c *CPU) popStack(reg *RegisterPair) {
	reg.SetUint16(uint16(c.pop()))
}

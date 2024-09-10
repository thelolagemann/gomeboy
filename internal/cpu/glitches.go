package cpu

// doHALTBug is called when the CPU is in HALT mode and
// the IME is disabled. It will execute the next instruction
// and then return to the HALT instruction.
func (c *CPU) doHALTBug() {
	// read the next instruction
	instr := c.readOperand()

	// decrement the PC to execute the instruction again
	c.PC--

	// execute the instruction
	c.decode(instr)
}

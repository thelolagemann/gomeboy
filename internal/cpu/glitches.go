package cpu

import (
	"github.com/thelolagemann/gomeboy/internal/ppu/lcd"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
)

// doHALTBug is called when the CPU is in HALT mode and
// the IME is disabled. It will execute the next instruction
// and then return to the HALT instruction.
func (c *CPU) doHALTBug() {
	// read the next instruction
	instr := c.readOperand()

	// decrement the PC to execute the instruction again
	c.PC--

	// execute the instruction
	c.instructions[instr](c)
}

// handleOAMCorruption is called when the CPU encounters
// a situation where the OAM could become corrupted. If
// conditions are met, the corruption will be emulated
// depending on the model.
func (c *CPU) handleOAMCorruption(pos uint16) {
	if c.b.Model() == types.CGBABC || c.b.Model() == types.CGB0 {
		return // no corruption on CGB
	}
	if pos >= 0xFE00 && pos < 0xFEFF {
		if (c.b.LazyRead(types.STAT)&0b11 == lcd.OAM ||
			c.s.Until(scheduler.PPUContinueOAMSearch) == 4) &&
			c.s.Until(scheduler.PPUEndOAMSearch) != 8 {
			// TODO
			// get the current cycle of mode 2 that the PPU is in
			// the oam is split into 20 rows of 8 bytes each, with
			// each row taking 1 M-cycle to read
			// so we need to figure out which row we're in
			// and then perform the oam corruption
			c.ppu.WriteCorruptionOAM()
		}
	}
}

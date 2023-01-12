package cpu

import "github.com/thelolagemann/go-gameboy/pkg/utils"

type Flag = uint8

const (
	FlagZero      Flag = 7
	FlagSubtract  Flag = 6
	FlagHalfCarry Flag = 5
	FlagCarry     Flag = 4
)

// InstructionFlags represents the flags that are affected by an instruction.
// This is used to determine which flags should be updated after an instruction
// is executed. Whilst some instructions have no effect on the flags, others
// will set, reset or alter the flags based on the result of the instruction.
type InstructionFlags struct {
	// Set is a slice of flags that should be set
	// regardless of the result of the instruction.
	Set []Flag
	// Reset is a slice of flags that should be reset
	// regardless of the result of the instruction.
	Reset []Flag
	// Operation is a slice of flags that should be
	// updated based on the result of the instruction.
	Operation []Flag
}

// clearFlag clears a flag from the F register.
func (c *CPU) clearFlag(flag Flag) {
	c.F = utils.Reset(c.F, flag)
	c.F &= 0xF0
}

// clearFlags clears the given flags.
func (c *CPU) clearFlags(flags ...Flag) {
	for _, flag := range flags {
		c.clearFlag(flag)
	}
}

// setFlag sets a flag to the given value.
func (c *CPU) setFlag(flag Flag) {
	c.F = utils.Set(c.F, flag)
	c.F &= 0xF0 // the lower 4 bits of the F register are always 0
}

// isFlagSet returns true if the given flag is set.
func (c *CPU) isFlagSet(flag Flag) bool {
	switch flag {
	case FlagZero:
		return c.F&0x80 == 0x80
	case FlagSubtract:
		return c.F&0x40 == 0x40
	case FlagHalfCarry:
		return c.F&0x20 == 0x20
	case FlagCarry:
		return c.F&0x10 == 0x10
	}

	return false
}

// isFlagsSet returns true if all the given flags are set.
func (c *CPU) isFlagsSet(flags ...Flag) bool {
	for _, flag := range flags {
		if !c.isFlagSet(flag) {
			return false
		}
	}
	return true
}

// isFlagsNotSet returns true if all the given flags are not set.
func (c *CPU) isFlagsNotSet(flags ...Flag) bool {
	return !c.isFlagsSet(flags...)
}

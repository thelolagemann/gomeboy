package cpu

import (
	"github.com/thelolagemann/go-gameboy/internal/types"
)

type Flag = uint8

const (
	FlagZero      Flag = types.Bit7
	FlagSubtract  Flag = types.Bit6
	FlagHalfCarry Flag = types.Bit5
	FlagCarry     Flag = types.Bit4
)

// clearFlag clears a flag from the F register.
func (c *CPU) clearFlag(flag Flag) {
	c.F = c.F &^ flag
	// c.F &= 0xF0
}

// setFlag sets a flag to the given value.
func (c *CPU) setFlag(flag Flag) {
	c.F = c.F | flag
	//c.F &= 0xF0 // the lower 4 bits of the F register are always 0
}

// isFlagSet returns true if the given flag is set.
func (c *CPU) isFlagSet(flag Flag) bool {
	return c.F&flag != 0
}

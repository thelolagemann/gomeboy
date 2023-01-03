package cpu

type Flag = uint8

const (
	FlagZero      Flag = 7
	FlagSubtract  Flag = 6
	FlagHalfCarry Flag = 5
	FlagCarry     Flag = 4
)

// clearFlag clears a flag from the F register.
func (c *CPU) clearFlag(flag Flag) {
	c.F = c.clearBit(c.F, flag)
}

// setFlag sets a flag to the given value.
func (c *CPU) setFlag(flag Flag) {
	c.F = c.setBit(c.F, flag)
}

// isFlagSet returns true if the given flag is set.
func (c *CPU) isFlagSet(flag Flag) bool {
	return c.F&(1<<flag) != 0
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

// isFlagNotSet returns true if the given flag is not set.
func (c *CPU) isFlagNotSet(flag Flag) bool {
	return !c.isFlagSet(flag)
}

// isFlagsNotSet returns true if all the given flags are not set.
func (c *CPU) isFlagsNotSet(flags ...Flag) bool {
	return !c.isFlagsSet(flags...)
}

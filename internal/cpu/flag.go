package cpu

import (
	"github.com/thelolagemann/gomeboy/internal/types"
)

// Flag represents a flag in the F register, which is
// used to hold the status of various mathematical
// operations.
//
// On the official hardware, the F register is 8 bits
// wide, but only the upper 4 bits are used. The lower
// 4 bits are always 0.
//
// The upper 4 bits are laid out as follows:
//
//	Bit 7 - (Z) FlagZero
//	Bit 6 - (N) FlagSubtract
//	Bit 5 - (H) FlagHalfCarry
//	Bit 4 - (C) FlagCarry
type Flag = uint8

const (
	// FlagZero is set when the result of an operation is 0.
	//
	// Examples:
	//  SUB A, B; A = 0x00, B = 0x00 -> FlagZero is set
	//  SUB A, B; A = 0x02, B = 0x01 -> FlagZero is not set
	//  DEC A; A = 0x01 -> FlagZero is set
	//  DEC A; A = 0x00 -> FlagZero is not set
	//  INC A; A = 0x00 -> FlagZero is not set
	//  INC A; A = 0xFF -> FlagZero is set
	FlagZero = types.Bit7
	// FlagSubtract is set when an operation performs a subtraction.
	//
	// Examples:
	//  SUB A, B; A = 0x00, B = 0x00 -> FlagSubtract is set
	//  SUB A, B; A = 0x02, B = 0x01 -> FlagSubtract is set
	//  ADD A, B; A = 0x00, B = 0x00 -> FlagSubtract is not set
	//  ADD A, B; A = 0x02, B = 0x01 -> FlagSubtract is not set
	//  DEC A; A = 0x01 -> FlagSubtract is set
	//  DEC A; A = 0x00 -> FlagSubtract is set
	//  INC A; A = 0x00 -> FlagSubtract is not set
	//  INC A; A = 0xFF -> FlagSubtract is not set
	FlagSubtract = types.Bit6
	// FlagHalfCarry is set when there is a carry from the lower nibble to
	// the upper nibble, or with 16-bit operations, when there is a carry
	// from the lower byte to the upper byte.
	//
	// Examples:
	//   ADD A, B; A = 0x0F, B = 0x01 -> FlagHalfCarry is set
	//   ADD A, B; A = 0x04, B = 0x01 -> FlagHalfCarry is not set
	//   ADD HL, BC; HL = 0x00FF, BC = 0x0001 -> FlagHalfCarry is set
	//   ADD HL, BC; HL = 0x000F, BC = 0x0001 -> FlagHalfCarry is not set
	FlagHalfCarry = types.Bit5
	// FlagCarry is set when there is a mathematical operation that has a
	// result that is too large to fit in the destination register.
	//
	// Examples:
	//   ADD A, B; A = 0xFF, B = 0x01 -> FlagCarry is set
	//   ADD A, B; A = 0x04, B = 0x01 -> FlagCarry is not set
	//   ADD HL, BC; HL = 0xFFFF, BC = 0x0001 -> FlagCarry is set
	//   ADD HL, BC; HL = 0x00FF, BC = 0x0001 -> FlagCarry is not set
	FlagCarry = types.Bit4
)

// clearFlag clears the given flag in the F register,
// leaving all other flags unchanged. If the flag
// is already cleared, this function does nothing. To
// set a flag, use setFlag.
func (c *CPU) clearFlag(flag Flag) {
	c.F &^= flag
}

// setFlags sets all the flags in the F register,
// as specified by the given arguments.
func (c *CPU) setFlags(Z bool, N bool, H bool, C bool) {
	v := uint8(0)
	if Z {
		v |= FlagZero
	}
	if N {
		v |= FlagSubtract
	}
	if H {
		v |= FlagHalfCarry
	}
	if C {
		v |= FlagCarry
	}
	c.F = v
}

// isFlagSet returns true if the given flag is set,
// false otherwise.
func (c *CPU) isFlagSet(flag Flag) bool {
	return c.F&flag == flag
}

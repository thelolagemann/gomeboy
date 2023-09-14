package types

// Register represents a GB Register which is used to hold an 8-bit value.
// The CPU has 8 registers: A, B, C, D, E, H, L, and F. The F register is
// special in that it is used to hold the flags.
type Register = uint8

// RegisterPair represents a pair of GB Registers which is used to hold a 16-bit
// value. The CPU has 4 register pairs: AF, BC, DE, and HL.
type RegisterPair struct {
	High *Register
	Low  *Register
}

// Uint16 returns the value of the RegisterPair as an uint16.
func (r *RegisterPair) Uint16() uint16 {
	return uint16(*r.High)<<8 | uint16(*r.Low)
}

// SetUint16 sets the value of the RegisterPair to the given value.
func (r *RegisterPair) SetUint16(value uint16) {
	*r.High = uint8(value >> 8)
	*r.Low = uint8(value)
}

// Registers represents the GB CPU registers.
type Registers struct {
	A Register
	B Register
	C Register
	D Register
	E Register
	F Register
	H Register
	L Register

	BC *RegisterPair
	DE *RegisterPair
	HL *RegisterPair
	AF *RegisterPair
}

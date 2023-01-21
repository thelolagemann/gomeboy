package registers

// CPU represents a Game Boy CPU register. The CPU
// has 8 registers: A, B, C, D, E, H, L, and F. The F register
// is special in that it is used to hold the flags, and only the
// upper 4 bits are used. The lower 4 bits are always 0. The
// CPU also has 4 register pairs: AF, BC, DE, and HL, which are
// used to access the upper and lower registers as a 16-bit value.
// These register pairs are defined in the CPUPair type.
type CPU = uint8

// CPUPair represents a register pair of the Game Boy,
// which is used to access the upper and lower registers as a
// 16-bit value. The CPU has 4 register pairs: AF, BC, DE, and HL.
type CPUPair struct {
	High *CPU
	Low  *CPU
}

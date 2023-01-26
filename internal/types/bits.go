package types

// Bit is used to represent a single bit within
// a bitfield, most commonly used to access
// the specific bit within a byte, setting,
// clearing, or testing the bit.
type Bit uint8

const (
	// Bit0 is the first bit.
	//
	//  0b0000_0001
	//            ^
	Bit0 Bit = 1 << iota
	// Bit1 is the second bit.
	//
	//  0b0000_0010
	//           ^
	Bit1
	// Bit2 is the third bit.
	//
	//  0b0000_0100
	//          ^
	Bit2
	// Bit3 is the fourth bit.
	//
	//  0b0000_1000
	//         ^
	Bit3
	// Bit4 is the fifth bit.
	//
	//  0b0001_0000
	//       ^
	Bit4
	// Bit5 is the sixth bit.
	//
	//  0b0010_0000
	//      ^
	Bit5
	// Bit6 is the seventh bit.
	//
	//  0b0100_0000
	//     ^
	Bit6
	// Bit7	is the eighth bit.
	//
	//  0b1000_0000
	//    ^
	Bit7
)

func SetBit(b uint8, bit Bit) uint8 {
	return b | uint8(bit)
}

func ResetBit(b uint8, bit Bit) uint8 {
	return b &^ uint8(bit)
}

func TestBit(b uint8, bit Bit) bool {
	return (b & uint8(bit)) != 0
}

package types

// Bit is used to represent a single bit within
// a bitfield, most commonly used to access
// the specific bit within a byte, setting,
// clearing, or testing the bit.
type Bit = uint8

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

type Mask = uint8

const (
	// Mask0 is a mask that will clear the first bit.
	//
	//  0b1111_1110
	//            ^
	Mask0 Mask = ^Bit0
	// Mask1 is a mask that will clear the second bit.
	//
	//  0b1111_1101
	//           ^
	Mask1 Mask = ^Bit1
	// Mask2 is a mask that will clear the third bit.
	//
	//  0b1111_1011
	//          ^
	Mask2 Mask = ^Bit2
	// Mask3 is a mask that will clear the fourth bit.
	//
	//  0b1111_0111
	//         ^
	Mask3 Mask = ^Bit3
	// Mask4 is a mask that will clear the fifth bit.
	//
	//  0b1110_1111
	//       ^
	Mask4 Mask = ^Bit4
	// Mask5 is a mask that will clear the sixth bit.
	//
	//  0b1101_1111
	//      ^
	Mask5 Mask = ^Bit5
	// Mask6 is a mask that will clear the seventh bit.
	//
	//  0b1011_1111
	//     ^
	Mask6 Mask = ^Bit6
	// Mask7 is a mask that will clear the eighth bit.
	//
	//  0b0111_1111
	//    ^
	Mask7 Mask = ^Bit7
)

// CombineMasks combines multiple bit masks into a single byte, where
// each bit is cleared if any of the masks clear that bit.
func CombineMasks(masks ...Mask) Mask {
	var mask Mask = 0xFF
	for _, m := range masks {
		// apply the mask to clear the bit
		mask &= m
	}
	return mask
}

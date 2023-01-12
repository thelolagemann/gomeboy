package palette

const (
	// Greyscale is the default greyscale palette.
	Greyscale = iota
	// Green is the green palette which attempts to emulate
	// the original colour palette as it would have appeared
	// on the original Game Boy.
	Green
	// Red is a red palette.
	Red
	// Yellow is a yellow palette.
	Yellow
)

// Palette represents a palette. A palette is an array of 4 RGB values,
// that can be used to represent a colour.
type Palette struct {
	Colors [4][3]uint8
}

// Current is the currently selected palette.
var Current = Greyscale

// Palettes is a list of all available palettes.
var Palettes = []Palette{
	// Greyscale
	{
		Colors: [4][3]uint8{
			{0xFF, 0xFF, 0xFF},
			{0xCC, 0xCC, 0xCC},
			{0x77, 0x77, 0x77},
			{0x00, 0x00, 0x00},
		},
	},
	// Green
	{
		Colors: [4][3]uint8{
			{0x9B, 0xBC, 0x0F},
			{0x8B, 0xAC, 0x0F},
			{0x30, 0x62, 0x30},
			{0x0F, 0x38, 0x0F},
		},
	},
	// Red
	{
		Colors: [4][3]uint8{
			{0xFF, 0x00, 0x00},
			{0xCC, 0x00, 0x00},
			{0x77, 0x00, 0x00},
			{0x00, 0x00, 0x00},
		},
	},
	// Yellow
	{
		Colors: [4][3]uint8{
			{0xFF, 0xFF, 0x00},
			{0xCC, 0xCC, 0x00},
			{0x77, 0x77, 0x00},
			{0x00, 0x00, 0x00},
		},
	},
}

// GetColour returns the colour based on the colour index and the
// Current palette.
func GetColour(index uint8) [3]uint8 {
	return Palettes[Current].Colors[index]
}

// ByteToPalette creates a new palette from a byte, using the
// selected palette as a base.
func ByteToPalette(b byte) Palette {
	var palette Palette
	palette.Colors[0] = Palettes[Current].Colors[b&0x03]
	palette.Colors[1] = Palettes[Current].Colors[(b>>2)&0x03]
	palette.Colors[2] = Palettes[Current].Colors[(b>>4)&0x03]
	palette.Colors[3] = Palettes[Current].Colors[(b>>6)&0x03]
	return palette
}

// ToByte converts a palette to a byte, using the
// selected palette as a base.
func (p Palette) ToByte() byte {
	var b byte
	b |= paletteToByte(p.Colors[0])
	b |= paletteToByte(p.Colors[1]) << 2
	b |= paletteToByte(p.Colors[2]) << 4
	b |= paletteToByte(p.Colors[3]) << 6
	return b
}

func (p Palette) GetColour(index uint8) [3]uint8 {
	return p.Colors[index]
}

func paletteToByte(color [3]uint8) byte {
	for i, p := range Palettes[Current].Colors {
		if p == color {
			return byte(i)
		}
	}
	return 0
}

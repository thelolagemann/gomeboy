package palette

const (
	// Greyscale is the default greyscale palette.
	Greyscale = iota
	// Green is the green palette which attempts to emulate
	// the original colour palette.
	Green
)

// Palette represents a palette. A palette is an array of 4 RGB values,
// that can be used to represent a colour.
type Palette struct {
	// The palette's colors.
	Colors [4][3]uint8
}

// Current is the currently selected palette.
var Current = Palettes[Greyscale]

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
}

// GetColour returns the colour based on the colour index and the
// Current palette.
func GetColour(index uint8) [3]uint8 {
	return Current.Colors[index]
}

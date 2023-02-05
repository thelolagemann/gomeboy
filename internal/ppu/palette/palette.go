package palette

import "image/color"

const (
	// Greyscale is the default greyscale palette.
	Greyscale = iota
	// Green is the green palette which attempts to emulate
	// the original colour palette as it would have appeared
	// on the original Game Boy.
	Green
)

// Palette represents a palette. A palette is an array of 4 different
// shades, represented as 2 bits each. When GetColour is called, the
// 2 bits are mapped to the currently selected ColourPalette.
//
// For example, if the palette is [0, 1, 2, 3], and the current
// ColourPalette is the default greyscale palette, the colours will be
// [0, 85, 170, 255].
//
// To give another example, if the palette is [3, 1, 0, 2], and the
// current ColourPalette is the green palette, the colours will be
// [0, 255, 0, 255].
type Palette [4]uint8

type Colour [4]color.RGBA

// Current is the currently selected palette.
var Current = Greyscale

var latchedPalette = Greyscale
var requestPaletteChange = false

// ColourPalettes maps the shades of a palette to their RGBA values.
var ColourPalettes = []Colour{
	// Greyscale
	{
		{0xFF, 0xFF, 0xFF, 0xFF},
		{0xCC, 0xCC, 0xCC, 0xFF},
		{0x77, 0x77, 0x77, 0xFF},
		{0x00, 0x00, 0x00, 0xFF},
	},
	// Green (mimics original)
	{
		{0x9B, 0xBC, 0x0F, 0xFF},
		{0x8B, 0xAC, 0x0F, 0xFF},
		{0x30, 0x62, 0x30, 0xFF},
		{0x0F, 0x38, 0x0F, 0xFF},
	},
}

// GetColour returns the colour based on the colour index and the
// Current palette.
func GetColour(index uint8) [3]uint8 {
	return [3]uint8{ColourPalettes[Current][index].R, ColourPalettes[Current][index].G, ColourPalettes[Current][index].B}
}

// ByteToPalette creates a new palette from a byte, using the
// selected palette as a base.
func ByteToPalette(b byte) Palette {
	var palette Palette
	palette[0] = b & 0x03
	palette[1] = (b >> 2) & 0x03
	palette[2] = (b >> 4) & 0x03
	palette[3] = (b >> 6) & 0x03
	return palette
}

// ToByte converts a palette to a byte, using the
// selected palette as a base.
func (p *Palette) ToByte() byte {
	var b byte

	b |= p[0]
	b |= p[1] << 2
	b |= p[2] << 4
	b |= p[3] << 6

	return b
}

func (p *Palette) GetColour(index uint8) [3]uint8 {
	// map provided index to the current palette
	return GetColour(p[index])
}

func toRGB(c [3]uint8) color.RGBA {
	return color.RGBA{R: c[0], G: c[1], B: c[2], A: 0xFF}
}

func CyclePalette() {
	latchedPalette++
	if latchedPalette >= len(ColourPalettes) {
		latchedPalette = Greyscale
	}
	requestPaletteChange = true
}

func UpdatePalette() {
	if requestPaletteChange {
		Current = latchedPalette
		requestPaletteChange = false
	}
}

package palette

import (
	"github.com/thelolagemann/gomeboy/internal/types"
	"image/color"
)

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
type Palette [4][3]uint8

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
		{0xAA, 0xAA, 0xAA, 0xAA},
		{0x55, 0x55, 0x55, 0x55},
		{0x00, 0x00, 0x00, 0x00},
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
	// get the first 2 bits
	palette[0] = GetColour(b & 0x03)
	// get the second 2 bits
	palette[1] = GetColour((b >> 2) & 0x03)
	// get the third 2 bits
	palette[2] = GetColour((b >> 4) & 0x03)
	// get the fourth 2 bits
	palette[3] = GetColour((b >> 6) & 0x03)
	return palette
}

// ToByte converts a palette to a byte, using the
// selected palette as a base.
//
// TODO The logic here is incredibly inefficient. It should be
// possible to do this in a single loop.
func (p Palette) ToByte() byte {
	var b byte

	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			if p[i] == GetColour(uint8(j)) {
				b |= uint8(j) << (i * 2)
			}
		}
	}

	return b
}

func (p Palette) GetColour(index uint8) [3]uint8 {
	// map provided index to the current palette
	return p[index]
}

func (p Palette) Save(s *types.State) {
	for _, pal := range p {
		for _, col := range pal {
			s.Write8(col)
		}
	}
}

func LoadPaletteFromState(s *types.State) Palette {
	p := Palette{}
	for i := 0; i < 4; i++ {
		for j := 0; j < 3; j++ {
			p[i][j] = s.Read8()
		}
	}
	return p
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

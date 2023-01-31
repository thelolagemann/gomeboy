package palette

// CGBPalette is a palette used by the CGB to provide
// up to 32768 colors.
type CGBPalette struct {
	palettes     [8][4][3]uint8
	Index        byte
	Incrementing bool
}

// SetIndex updates the index of the palette.
func (p *CGBPalette) SetIndex(value byte) {
	p.Index = value & 0x3F
	p.Incrementing = value&0x80 != 0 // if bit 7 is set, incrementing is true
}

// GetIndex returns the index of the palette.
func (p *CGBPalette) GetIndex() byte {
	if p.Incrementing {
		return p.Index | 0x80
	} else {
		return p.Index
	}
}

// Read returns the value of the palette at the specified index.
func (p *CGBPalette) Read() byte {
	paletteIndex := p.Index >> 3
	colourIndex := p.Index & 0x7 >> 1

	colour := uint16(
		uint16(p.palettes[paletteIndex][colourIndex][0]>>3)<<0 |
			uint16(p.palettes[paletteIndex][colourIndex][1]>>3)<<5 |
			uint16(p.palettes[paletteIndex][colourIndex][2]>>3)<<10,
	)

	if p.Index&0x1 == 0 {
		return uint8(colour & 0xFF)
	} else {
		return uint8(colour >> 8)
	}
}

// Write writes the value to the palette at the specified index.
func (p *CGBPalette) Write(value byte) {
	paletteIndex := p.Index >> 3
	colourIndex := p.Index & 0x7 >> 1

	colour := uint16(p.palettes[paletteIndex][colourIndex][0]>>3)<<0 |
		uint16(p.palettes[paletteIndex][colourIndex][1]>>3)<<5 |
		uint16(p.palettes[paletteIndex][colourIndex][2]>>3)<<10

	if p.Index&0x1 == 0 {
		colour = colour&0xFF00 | uint16(value)
	} else {
		colour = colour&0x00FF | uint16(value)<<8
	}

	p.palettes[paletteIndex][colourIndex][0] = uint8(colour>>0) & 0x1F << 3
	p.palettes[paletteIndex][colourIndex][1] = uint8(colour>>5) & 0x1F << 3
	p.palettes[paletteIndex][colourIndex][2] = uint8(colour>>10) & 0x1F << 3

	if p.Incrementing {
		p.Index = (p.Index + 1) & 0x3F
	}
}

// GetColour returns the colour for a given palette index,
// and colour index.
func (p *CGBPalette) GetColour(paletteIndex byte, colourIndex byte) [3]uint8 {
	return p.palettes[paletteIndex][colourIndex]
}

func NewCGBPallette() *CGBPalette {
	pal := [8][4][3]uint8{}
	for i := 0; i < 8; i++ {
		for j := 0; j < 4; j++ {
			pal[i][j] = [3]uint8{0xFF, 0xFF, 0xFF}
		}
	}

	return &CGBPalette{
		palettes: pal,
	}
}

package palette

import (
	"github.com/thelolagemann/gomeboy/internal/types"
	"image"
	"image/color"
	"image/png"
	"os"
)

// CGBPalette is a palette used by the CGB to provide
// up to 32768 colors.
type CGBPalette struct {
	Palettes     [8]Palette
	Index        byte
	Incrementing bool
}

// SetIndex updates the index of the palette.
func (p *CGBPalette) SetIndex(value byte) {
	p.Index = value & 0x3F
	p.Incrementing = value&types.Bit7 != 0
}

// GetIndex returns the index of the palette.
func (p *CGBPalette) GetIndex() byte {
	if p.Incrementing {
		return p.Index | types.Bit7
	} else {
		return p.Index
	}
}

// TODO save compatibility palette
// - load game with boot ROM enabled
// - save colour palette to file (bgp = index 0 of colour palette, obp1 = index 0 of sprite palette, obp2 = index 1 of sprite palette)
// - encoded filename as hash of palette

type CompatibilityPalette struct {
	BGP  [4][3]uint8 `json:"bgp"`
	OBP0 [4][3]uint8 `json:"obp1"`
	OBP1 [4][3]uint8 `json:"obp2"`
}

// Read returns the value of the palette at the specified index.
func (p *CGBPalette) Read() byte {
	paletteIndex := p.Index >> 3
	colourIndex := (p.Index & 0x7) >> 1

	colour := (uint16(p.Palettes[paletteIndex][colourIndex][0]>>3) << 0) |
		(uint16(p.Palettes[paletteIndex][colourIndex][1]>>3) << 5) |
		(uint16(p.Palettes[paletteIndex][colourIndex][2]>>3) << 10)

	if p.Index&1 == 0 {
		return uint8(colour) & 0xFF
	} else {
		return uint8(colour >> 8)
	}
}

// Write writes the value to the palette at the specified index.
func (p *CGBPalette) Write(value byte) {
	paletteIndex := p.Index >> 3
	colourIndex := (p.Index & 0x7) >> 1

	colour := (uint16(p.Palettes[paletteIndex][colourIndex][0]>>3) << 0) |
		(uint16(p.Palettes[paletteIndex][colourIndex][1]>>3) << 5) |
		(uint16(p.Palettes[paletteIndex][colourIndex][2]>>3) << 10)

	if p.Index&0x1 == 0 {
		colour = (colour & 0xFF00) | uint16(value)
	} else {
		colour = (colour & 0x00FF) | uint16(value)<<8
	}

	p.Palettes[paletteIndex][colourIndex][0] = (uint8(colour>>0)&0x1F)<<3 | (uint8(colour>>0)&0x1F)>>2
	p.Palettes[paletteIndex][colourIndex][1] = (uint8(colour>>5)&0x1F)<<3 | (uint8(colour>>5)&0x1F)>>2
	p.Palettes[paletteIndex][colourIndex][2] = (uint8(colour>>10)&0x1F)<<3 | (uint8(colour>>10)&0x1F)>>2

	if p.Incrementing {
		p.Index = (p.Index + 1) & 0x3F
	}
}

// GetColour returns the colour for a given palette index,
// and colour index.
func (p *CGBPalette) GetColour(paletteIndex byte, colourIndex byte) [3]uint8 {
	return p.Palettes[paletteIndex][colourIndex]
}

func NewCGBPallette() *CGBPalette {
	pal := [8]Palette{}
	for i := 0; i < 8; i++ {
		for j := 0; j < 4; j++ {
			pal[i][j] = [3]uint8{0xFF, 0xFF, 0xFF}
		}
	}

	return &CGBPalette{
		Palettes: pal,
	}
}

// SaveExample saves an example of the currently available Palettes,
// by drawing a grid of all available colours.
func (p *CGBPalette) SaveExample(imgOutput string) {
	// open output file
	out, err := os.Create(imgOutput)
	if err != nil {
		panic(err)
	}

	// create a new image
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))

	// draw the grid
	for i := 0; i < 8; i++ {
		for j := 0; j < 4; j++ {
			for x := 0; x < 32; x++ {
				for y := 0; y < 32; y++ {
					img.Set(32*i+x, 32*j+y, color.RGBA{
						R: p.Palettes[i][j][0],
						G: p.Palettes[i][j][1],
						B: p.Palettes[i][j][2],
						A: 0xFF,
					})
				}
			}
		}
	}

	// encode the image
	err = png.Encode(out, img)
	if err != nil {
		panic(err)
	}

	// close the file
	err = out.Close()
	if err != nil {
		panic(err)
	}
}

func (p *CGBPalette) Load(s *types.State) {
	for i := range p.Palettes {
		p.Palettes[i] = LoadPaletteFromState(s)
	}
	p.Index = s.Read8()
}

func (p *CGBPalette) Save(s *types.State) {
	for _, pa := range p.Palettes {
		pa.Save(s)
	}
	s.Write8(p.Index)
}

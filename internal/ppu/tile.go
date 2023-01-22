package ppu

import (
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"image"
	"image/color"
)

// Tile represents a tile. Each tile has a size of 8x8 pixels and a color
// depth of 4 colors/gray shades. Tiles can be displayed as sprites or as
// background/window tiles.
type Tile [8][8]int

type TileAttributes struct {
	// UseBGPriority is the BG Priority bit. When set, the tile is displayed
	// behind the background and window. Otherwise, it is displayed in front
	// of the background and window.
	UseBGPriority bool
	// YFlip is the Y Flip bit. When set, the tile is flipped vertically.
	YFlip bool
	// XFlip is the X Flip bit. When set, the tile is flipped horizontally.
	XFlip bool
	// PaletteNumber is the Palette Number bit. It specifies the palette
	// number (0-7) that is used to determine the tile's colors.
	PaletteNumber uint8
	// VRAMBank is the VRAM Bank bit. It specifies the VRAM bank (0-1) that
	// is used to store the tile's data.
	VRAMBank uint8
}

func NewTile(b [16]uint8) *Tile {
	t := Tile{}
	for tileY := 0; tileY < 8; tileY++ {
		lo, hi := int(b[tileY*2]), int(b[tileY*2+1])
		for tileX := 0; tileX < 8; tileX++ {
			t[tileY][tileX] = (lo >> (7 - tileX) & 1) | (hi>>(7-tileX)&1)<<1
		}
	}

	return &t
}

// Read returns the byte of the tile at the given address.
func (t *Tile) Read(address uint16) uint8 {
	var tileY = int(address) / 2
	var tileX = int(address) % 2
	return uint8(t[tileY][tileX])
}

// Write writes the given value to the tile at the given address.
func (t *Tile) Write(address uint16, value uint8) {
	var tileY = int(address) / 2
	var tileX = int(address) % 2
	t[tileY][tileX] = int(value)
}

// Draw draws the tile to the given image at the given position.
func (t *Tile) Draw(img *image.RGBA, i int, i2 int) {
	for tileY := 0; tileY < 8; tileY++ {
		for tileX := 0; tileX < 8; tileX++ {
			var x = i + tileX
			var y = i2 + tileY
			var colourNum = t[tileY][tileX]
			rgb := palette.GetColour(uint8(colourNum))
			img.Set(x, y, color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 0xff})
		}
	}
}

type TileMap struct {
	// The tile map's tiles.
	Tiles [32][32]Tile
}

func (t *TileAttributes) Read(address uint16) uint8 {
	var val uint8
	if t.UseBGPriority {
		val |= 0x80
	}
	if t.YFlip {
		val |= 0x40
	}
	if t.XFlip {
		val |= 0x20
	}
	val |= t.PaletteNumber & 0b111
	val |= t.VRAMBank << 3
	return val
}

func (t *TileAttributes) Write(address uint16, value uint8) {
	t.UseBGPriority = value&0x80 != 0
	t.YFlip = value&0x40 != 0
	t.XFlip = value&0x20 != 0
	t.PaletteNumber = value & 0b111
	t.VRAMBank = value & 0x8 >> 3

	// fmt.Printf("updated tile with attributes: %v %v %v %v %v\n", t.UseBGPriority, t.YFlip, t.XFlip, t.PaletteNumber, t.VRAMBank)
}

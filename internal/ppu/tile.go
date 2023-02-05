package ppu

import (
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"image"
	"image/color"
)

// Tile represents a tile. Each tile has a size of 8x8 pixels and a color
// depth of 4 colors/gray shades. Tiles can be displayed as sprites or as
// background/window tiles.
type Tile [8][2]uint8

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

// TileMap represents a tile map. A tile map is a 32x32 array of tiles,
// each tile being 8x8 pixels. The tile map is used to determine which
// tiles are displayed in the background and window. There are two tile
// maps, located at 0x9800 and 0x9C00, and each tile map can be used for
// the background or window. In CGB mode, there are two tile maps for
// each background and window, located at 0x9800 and 0x9C00 for bank 0,
// and at 0x9C00 and 0xA000 for bank 1.
type TileMap [32][32]TileMapEntry

// NewTileMap returns a new tile map.
func NewTileMap() TileMap {
	var tileMap = TileMap{}
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			tileMap[y][x] = TileMapEntry{
				TileID:     0,
				Attributes: &TileAttributes{},
			}
		}
	}
	return tileMap
}

type TileMapEntry struct {
	TileID     uint8
	Attributes *TileAttributes
}

func (t *TileMapEntry) GetID(signed bool) int {
	id := int(t.TileID)
	if signed {
		if id < 128 {
			id += 256
		}
	}
	return id
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

func (t *TileAttributes) Write(value uint8) {
	t.UseBGPriority = value&0x80 != 0
	t.YFlip = value&0x40 != 0
	t.XFlip = value&0x20 != 0
	t.PaletteNumber = value & 0b111
	t.VRAMBank = value >> 3 & 0x1
}

// Draw draws the bank number over the tile map.
func (t *TileAttributes) Draw(img *image.RGBA, i int, i2 int) {
	for tileY := 0; tileY < 8; tileY++ {
		for tileX := 0; tileX < 8; tileX++ {
			var x = i + tileX
			var y = i2 + tileY
			var colourNum = int(t.VRAMBank)
			rgb := palette.GetColour(uint8(colourNum))

			// mix with current colour
			currentColor := img.At(x, y)
			r, g, b, a := currentColor.RGBA()
			rgb[0] = uint8((rgb[0] + uint8(r)) / 2)
			rgb[1] = uint8((rgb[1] + uint8(g)) / 2)
			rgb[2] = uint8((rgb[2] + uint8(b)) / 2)
			img.Set(x, y, color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: uint8(a)})

		}
	}
}

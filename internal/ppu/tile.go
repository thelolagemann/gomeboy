package ppu

import (
	"github.com/thelolagemann/gomeboy/internal/ppu/palette"
	"image"
)

// Tile represents a tile. Each tile has a size of 8x8 pixels and a color
// depth of 4 colors/gray shades. Tiles can be displayed as sprites or as
// background/window tiles.
type Tile [16]uint8

type TileAttributes struct {
	// BGPriority is the BG priority bit. When set, the tile is displayed
	// behind the background and window. Otherwise, it is displayed in front
	// of the background and window.
	BGPriority bool
	// YFlip is the Y Flip bit. When set, the tile is flipped vertically.
	YFlip bool
	// XFlip is the X Flip bit. When set, the tile is flipped horizontally.
	XFlip bool
	// CGBPaletteNumber is the Palette Number bit. It specifies the palette
	// number (0-7) that is used to determine the tile's colors.
	CGBPaletteNumber uint8
	// VRAMBank is the VRAM Bank bit. It specifies the VRAM bank (0-1) that
	// is used to store the tile's data.
	VRAMBank uint8
}

// Draw draws the tile to the given image at the given position.
func (t Tile) Draw(img *image.RGBA, i int, i2 int, pal palette.Palette) {
	for tileY := 0; tileY < 8; tileY++ {
		for tileX := 0; tileX < 8; tileX++ {
			var x = i + tileX
			var y = i2 + tileY
			high, low := t[tileY], t[tileY+8]
			var colourNum = int((high >> (7 - tileX)) & 1)
			colourNum |= int((low>>(7-tileX))&1) << 1
			rgb := pal.GetColour(uint8(colourNum))
			img.Pix[(y*img.Stride)+(x*4)] = rgb[0]
			img.Pix[(y*img.Stride)+(x*4)+1] = rgb[1]
			img.Pix[(y*img.Stride)+(x*4)+2] = rgb[2]
			img.Pix[(y*img.Stride)+(x*4)+3] = 0xff
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
				id: 0,
			}
		}
	}
	return tileMap
}

type TileMapEntry struct {
	id         uint16
	Attributes TileAttributes

	Tile
}

// GetID returns the tile ID of the entry according to the
// current addressing mode.
func (t TileMapEntry) GetID(addressingMode bool) uint16 {
	id := t.id
	if addressingMode {
		if id < 128 {
			id += 256
		}
	}
	return id
}

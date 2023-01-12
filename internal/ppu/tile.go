package ppu

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

func NewTile(b [16]uint8) Tile {
	t := Tile{}
	for tileY := 0; tileY < 8; tileY++ {
		lo, hi := int(b[tileY*2]), int(b[tileY*2+1])
		for tileX := 0; tileX < 8; tileX++ {
			t[tileY][tileX] = (lo >> (7 - tileX) & 1) | (hi>>(7-tileX)&1)<<1
		}
	}

	return t
}

type TileMap struct {
	// The tile map's tiles.
	Tiles [32][32]Tile
}

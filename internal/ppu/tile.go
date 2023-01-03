package ppu

// Tile represents a tile. Each tile has a size of 8x8 pixels and a color
// depth of 4 colors/gray shades. Tiles can be displayed as sprites or as
// background/window tiles.
type Tile struct {
	// The tile's pixels. Each tile occupies 16 bytes of memory, which
	// corresponds to 2 bytes per row. For each line, the first byte
	// specifies the low bits of the color index, and the second byte
	// specifies the high bits of the color index. In both bytes,
	// the most significant bit is the leftmost pixel, and the least
	// significant bit is the rightmost pixel.
	Pixels [8][2]byte
}

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

func NewTile(b []byte) Tile {
	t := Tile{}
	for i := 0; i < 16; i++ {
		t.Pixels[i/2][i%2] = b[i]
	}

	return t
}

type TileMap struct {
	// The tile map's tiles.
	Tiles [32][32]Tile
}

package ppu

// Sprite is the 8x8 default sprite.
type Sprite struct {
	X      uint8
	Y      uint8
	TileID uint8
	spriteAttributes
}

// spriteAttributes represents the Attributes of a sprite.
type spriteAttributes struct {
	// Bit 7 - OBJ-to-BG priority (0=OBJ Above BG, 1=OBJ Behind BG color 1-3)
	// (Used for both BG and Window. BG color 0 is always behind OBJ)
	priority bool
	// Bit 6 - Y flip          (0=Normal, 1=Vertically mirrored)
	flipY bool
	// Bit 5 - X flip          (0=Normal, 1=Horizontally mirrored)
	flipX bool
	// Bit 3 - Tile VRAM-Bank  **CGB mode Only**     (0=Bank 0, 1=Bank 1)
	vRAMBank uint8
	// Bit 0-2 - Palette Number or Bit 4 - Palette Mode
	paletteNumber uint8
}

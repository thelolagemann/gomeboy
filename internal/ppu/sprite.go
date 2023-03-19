package ppu

// Sprite is the 8x8 default sprite.
type Sprite struct {
	X      uint8
	Y      uint8
	TileID uint8
	spriteAttributes
}

// spriteAttributes represents the attributes of a sprite.
type spriteAttributes struct {
	// Bit 7 - OBJ-to-BG priority (0=OBJ Above BG, 1=OBJ Behind BG color 1-3)
	// (Used for both BG and Window. BG color 0 is always behind OBJ)
	priority bool
	// Bit 6 - Y flip          (0=Normal, 1=Vertically mirrored)
	flipY bool
	// Bit 5 - X flip          (0=Normal, 1=Horizontally mirrored)
	flipX bool
	// Bit 4 - Palette number  **Non CGB mode Only** (0=OBP0, 1=OBP1)
	useSecondPalette uint8
	// Bit 3 - Tile VRAM-Bank  **CGB mode Only**     (0=Bank 0, 1=Bank 1)
	vRAMBank uint8
	// Bit 0-2 - Palette number  **CGB mode Only**     (OBP0-7)
	cgbPalette uint8
}

func (s *Sprite) Update(address uint16, value uint8) {
	byteIndex := address % 4
	if byteIndex == 0 {
		s.Y = value
	} else if byteIndex == 1 {
		s.X = value
	} else if byteIndex == 2 {
		s.TileID = value
	} else if byteIndex == 3 {
		s.priority = value&0x80 == 0
		s.flipY = value&0x40 != 0
		s.flipX = value&0x20 != 0
		s.useSecondPalette = value & 0x10 >> 4
		s.vRAMBank = (value >> 3) & 0x01
		s.cgbPalette = value & 0x07
	}
}

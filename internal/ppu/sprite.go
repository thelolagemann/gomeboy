package ppu

// Sprite is the 8x8 default sprite.
type Sprite struct {
	X      uint8
	Y      uint8
	TileID uint8
	SpriteAttributes
}

// SpriteAttributes represents the attributes of a sprite.
type SpriteAttributes struct {
	// Bit 7 - OBJ-to-BG Priority (0=OBJ Above BG, 1=OBJ Behind BG color 1-3)
	// (Used for both BG and Window. BG color 0 is always behind OBJ)
	Priority bool
	// Bit 6 - Y flip          (0=Normal, 1=Vertically mirrored)
	FlipY bool
	// Bit 5 - X flip          (0=Normal, 1=Horizontally mirrored)
	FlipX bool
	// Bit 4 - Palette number  **Non CGB Mode Only** (0=OBP0, 1=OBP1)
	UseSecondPalette uint8
	// Bit 3 - Tile VRAM-Bank  **CGB Mode Only**     (0=Bank 0, 1=Bank 1)
	VRAMBank uint8
	// Bit 0-2 - Palette number  **CGB Mode Only**     (OBP0-7)
	CGBPalette uint8

	// raw data
	value uint8
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
		s.Priority = value&0x80 == 0
		s.FlipY = value&0x40 != 0
		s.FlipX = value&0x20 != 0
		s.UseSecondPalette = value & 0x10 >> 4
		s.VRAMBank = (value >> 3) & 0x01
		s.CGBPalette = value & 0x07
	}
	s.value = value
}

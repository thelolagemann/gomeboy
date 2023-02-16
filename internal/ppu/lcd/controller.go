package lcd

import (
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/thelolagemann/go-gameboy/pkg/utils"
)

// Controller is the LCD controller. It is responsible for controlling various
// aspects of the LCD, such as enabling the background and window display.
//
// Its value is stored in the LCD Control Register (0xFF40) as follows:
//
//	Bit 7 - LCD Enable             (0=Off, 1=On)
//	Bit 6 - Window Tile Map Display Select (0=9800-9BFF, 1=9C00-9FFF)
//	Bit 5 - Window Display Enable          (0=Off, 1=On)
//	Bit 4 - BG & Window Tile Data Select   (0=8800-97FF, 1=8000-8FFF)
//	Bit 3 - BG Tile Map Display Select     (0=9800-9BFF, 1=9C00-9FFF)
//	Bit 2 - OBJ (Sprite) Size              (0=8x8, 1=8x16)
//	Bit 1 - OBJ (Sprite) Display Enable    (0=Off, 1=On)
//	Bit 0 - BG/Window Display/Priority     (0=Off, 1=On)
type Controller struct {
	// Enabled is the LCD Enable bit. When set, the LCD is enabled.
	Enabled bool
	// WindowTileMapAddress represents the Window Tile Map Display Select bit.
	// When set, the window tile map is located at 0x9C00-0x9FFF. Otherwise, it
	// is located at 0x9800-0x9BFF. For convenience, this is stored as an uint16
	// depicting the start address of the tile map.
	//	(0=9800-9BFF)
	//  (1=9C00-9FFF)
	WindowTileMapAddress uint16
	// WindowEnabled is the Window Display Enable bit. When set, the window is
	// enabled.
	WindowEnabled bool
	// TileDataAddress represents the BG & Window Tile Data Select bit. When
	// set, the tile data is located at 0x8000-0x8FFF. Otherwise, it is located
	// at 0x8800-0x97FF (signed). For convenience, this is stored as an uint16
	// depicting the start address of the tile data.
	//	(0=8800-97FF)
	//  (1=8000-8FFF)
	TileDataAddress uint16
	// BackgroundTileMapAddress represents the BG Tile Map Display Select bit.
	// When set, the background tile map is located at 0x9C00-0x9FFF. Otherwise,
	// it is located at 0x9800-0x9BFF. For convenience, this is stored as an
	// uint16 depicting the start address of the tile map.
	//	(0=9800-9BFF)
	//  (1=9C00-9FFF)
	BackgroundTileMapAddress uint16
	// SpriteSize is the OBJ (Sprite) Size bit. It is 8x8 when the bit is
	// reset, and 8x16 when the bit is set.
	SpriteSize uint8
	// SpriteEnabled is the OBJ (Sprite) Display Enable bit. When set, sprites
	// are enabled.
	SpriteEnabled bool
	// BackgroundEnabled is the BG/Window Display/Priority bit. When set, the
	// background and window are enabled.
	BackgroundEnabled bool

	cleared  bool
	isSigned bool
}

func (c *Controller) init(onWrite types.WriteHandler) {
	types.RegisterHardware(
		types.LCDC,
		func(v uint8) {
			// detect a rising edge on the LCD enable bit
			if !c.Enabled && utils.Test(v, 7) {
				c.cleared = false
			}
			c.Enabled = utils.Test(v, 7)
			if utils.Test(v, 6) {
				c.WindowTileMapAddress = 0x9C00
			} else {
				c.WindowTileMapAddress = 0x9800
			}
			c.WindowEnabled = utils.Test(v, 5)
			if utils.Test(v, 4) {
				c.TileDataAddress = 0x8000
				c.isSigned = false
			} else {
				c.TileDataAddress = 0x8800
				c.isSigned = true
			}
			if utils.Test(v, 3) {
				c.BackgroundTileMapAddress = 0x9C00
			} else {
				c.BackgroundTileMapAddress = 0x9800
			}
			c.SpriteSize = 8 + uint8(utils.Val(v, 2))*8
			c.SpriteEnabled = utils.Test(v, 1)
			c.BackgroundEnabled = utils.Test(v, 0)
		}, func() uint8 {
			var value uint8
			if c.Enabled {
				value |= 1 << 7
			}
			if c.WindowTileMapAddress == 0x9C00 {
				value |= 1 << 6
			}
			if c.WindowEnabled {
				value |= 1 << 5
			}
			if c.TileDataAddress == 0x8000 {
				value |= 1 << 4
			}
			if c.BackgroundTileMapAddress == 0x9C00 {
				value |= 1 << 3
			}
			if c.SpriteSize == 16 {
				value |= 1 << 2
			}
			if c.SpriteEnabled {
				value |= 1 << 1
			}
			if c.BackgroundEnabled {
				value |= 1 << 0
			}
			return value
		},
		types.WithWriteHandler(onWrite),
	)
}

// NewController returns a new LCD controller.
func NewController(writeHandler types.WriteHandler) *Controller {
	c := &Controller{
		WindowTileMapAddress:     0x9800,
		BackgroundTileMapAddress: 0x9800,
		TileDataAddress:          0x8800,
		SpriteSize:               8,
		BackgroundEnabled:        false,
		SpriteEnabled:            false,
		WindowEnabled:            false,
		Enabled:                  false,
	}
	c.init(writeHandler)
	return c
}

// UsingSignedTileData returns true if the LCD controller is using signed tile
// data.
func (c *Controller) UsingSignedTileData() bool {
	return c.isSigned
}

func (c *Controller) Clear() {
	c.cleared = true
}

func (c *Controller) Cleared() bool {
	return c.cleared
}

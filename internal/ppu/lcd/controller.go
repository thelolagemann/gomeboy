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
//	Bit 0 - BG/Window Display/priority     (0=Off, 1=On)
type Controller struct {
	// Enabled is the LCD Enable bit. When set, the LCD is enabled.
	Enabled bool
	// WindowTileMap represents the Window Tile Map Display Select bit.
	// When set, the window tile map is located at 0x9C00-0x9FFF. Otherwise, it
	// is located at 0x9800-0x9BFF. For convenience, this is stored as an uint16
	// depicting the start address of the tile map.
	//	(0=9800-9BFF)
	//  (1=9C00-9FFF)
	WindowTileMap uint8
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
	// BackgroundTileMap represents the BG Tile Map Display Select bit.
	// When set, the background tile map is located at 0x9C00-0x9FFF. Otherwise,
	// it is located at 0x9800-0x9BFF. For convenience, this is stored as an
	// uint16 depicting the start address of the tile map.
	//	(0=9800-9BFF)
	//  (1=9C00-9FFF)
	BackgroundTileMap uint8
	// SpriteSize is the OBJ (Sprite) Size bit. It is 8x8 when the bit is
	// reset, and 8x16 when the bit is set.
	SpriteSize uint8
	// SpriteEnabled is the OBJ (Sprite) Display Enable bit. When set, sprites
	// are enabled.
	SpriteEnabled bool
	// BackgroundEnabled is the BG/Window Display/priority bit. When set, the
	// background and window are enabled.
	BackgroundEnabled bool

	Raw      uint8
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
				c.WindowTileMap = 1
			} else {
				c.WindowTileMap = 0
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
				c.BackgroundTileMap = 1
			} else {
				c.BackgroundTileMap = 0
			}
			c.SpriteSize = 8 + uint8(utils.Val(v, 2))*8
			c.SpriteEnabled = utils.Test(v, 1)
			c.BackgroundEnabled = utils.Test(v, 0)
			c.Raw = v
		}, func() uint8 {
			return c.Raw
		},
		types.WithWriteHandler(onWrite),
	)
}

// NewController returns a new LCD controller.
func NewController(writeHandler types.WriteHandler) *Controller {
	c := &Controller{
		WindowTileMap:     0,
		BackgroundTileMap: 0,
		TileDataAddress:   0x8800,
		SpriteSize:        8,
		BackgroundEnabled: false,
		SpriteEnabled:     false,
		WindowEnabled:     false,
		Enabled:           false,
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

var _ types.Stater = (*Controller)(nil)

func (c *Controller) Load(state *types.State) {
	v := state.Read8()
	if !c.Enabled && utils.Test(v, 7) {
		c.cleared = false
	}
	c.Enabled = utils.Test(v, 7)
	if utils.Test(v, 6) {
		c.WindowTileMap = 1
	} else {
		c.WindowTileMap = 0
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
		c.BackgroundTileMap = 1
	} else {
		c.BackgroundTileMap = 0
	}
	c.SpriteSize = 8 + uint8(utils.Val(v, 2))*8
	c.SpriteEnabled = utils.Test(v, 1)
	c.BackgroundEnabled = utils.Test(v, 0)
	c.Raw = v
}

func (c *Controller) Save(s *types.State) {
	s.Write8(c.Raw)
}

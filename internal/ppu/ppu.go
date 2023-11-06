// Package ppu provides a programmable pixel unit for the DMG and CGB.
package ppu

import (
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/ppu/lcd"
	"github.com/thelolagemann/gomeboy/internal/ppu/palette"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/utils"
	"image"
	"image/color"
	"unsafe"
)

const (
	// ScreenWidth is the width of the screen in pixels.
	ScreenWidth = 160
	// ScreenHeight is the height of the screen in pixels.
	ScreenHeight = 144
)

type PPU struct {
	// LCDC register
	Enabled                                         bool
	BackgroundEnabled, WindowEnabled, SpriteEnabled bool
	BackgroundTileMap, WindowTileMap                uint8
	TileDataAddress                                 uint16
	SpriteSize                                      uint8
	isSigned, cleared                               bool

	BGColourisationPalette   *palette.Palette
	OBJ0ColourisationPalette *palette.Palette
	OBJ1ColourisationPalette *palette.Palette

	ColourPalettes []palette.Palette

	// Window
	windowInternal uint8

	mode   lcd.Mode
	status byte

	oam                 *OAM
	ColourPalette       *palette.CGBPalette
	ColourSpritePalette *palette.CGBPalette

	TileChanged [2][384]bool // used for debug views (tile viewer)
	TileData    [2][384]Tile // 384 tiles, 8x8 pixels each (double in CGB mode)
	TileMaps    [2]TileMap   // 32x32 tiles, 8x8 pixels each

	PreparedFrame [ScreenHeight][ScreenWidth][3]uint8
	colorNumber   [ScreenWidth]uint8

	backgroundLineRendered [ScreenHeight]bool

	b                  *io.Bus
	statInterruptDelay bool
	RefreshScreen      bool

	tileBgPriority [ScreenHeight][ScreenWidth]bool

	// various LUTs
	colourNumberLUT         [256][256][8]uint8
	reversedColourNumberLUT [256][256][8]uint8
	modeInterruptLUT        [4][256]bool

	notifyFrame func()

	// debug
	Debug struct {
		SpritesDisabled    bool
		BackgroundDisabled bool
		WindowDisabled     bool
	}
	backgroundDirty bool

	dirtiedLog [65536]dirtyEvent
	lastDirty  uint16

	s                     *scheduler.Scheduler
	lyForComparison       uint16
	lycInterruptLine      bool
	statInterruptLine     bool
	modeToInterrupt       uint8
	currentLine           uint8
	backgroundLineChanged [256]bool
	lyCompare             byte
}

func New(b *io.Bus, s *scheduler.Scheduler) *PPU {
	oam := NewOAM()
	p := &PPU{
		TileData: [2][384]Tile{},

		b:   b,
		s:   s,
		oam: oam,
		ColourPalettes: []palette.Palette{
			// Greyscale
			{
				{0xFF, 0xFF, 0xFF},
				{0xAA, 0xAA, 0xAA},
				{0x55, 0x55, 0x55},
				{0x00, 0x00, 0x00},
			},
			// Green (mimics original)
			{
				{0x9B, 0xBC, 0x0F},
				{0x8B, 0xAC, 0x0F},
				{0x30, 0x62, 0x30},
				{0x0F, 0x38, 0x0F},
			},
		},
	}

	b.ReserveAddress(types.LCDC, func(v byte) byte {
		// is the screen turning off?
		if p.Enabled && v&types.Bit7 == 0 {
			// turn off the screen
			p.Enabled = false

			// the screen should only be turned off in VBlank
			if p.mode != lcd.VBlank {
				// warn user of incorrect behaviour TODO
			}

			// deschedule all PPU events
			for i := scheduler.PPUHBlank; i <= scheduler.PPUOAMInterrupt; i++ {
				p.s.DescheduleEvent(i)
			}

			// clear the screen
			p.renderBlank()

			// when the LCD is off, LY reads 0, and STAT mode reads 0 (HBlank)
			p.b.Set(types.LY, 0)
			p.currentLine = 0

			p.mode = lcd.HBlank

			// unlock OAM/VRAM
			p.b.Unlock(io.OAM)
			p.b.Unlock(io.VRAM)
		} else if !p.Enabled && v&types.Bit7 != 0 {
			// turn on the screen
			p.Enabled = true
			// reset LYC to compare against and clear coincidence flag
			p.lyForComparison = 0
			p.b.Set(types.LY, 0)
			p.currentLine = 0

			// perform STAT check
			p.modeToInterrupt = 255
			p.statUpdate()

			// schedule end of first glitched line
			p.s.ScheduleEvent(scheduler.PPUStartGlitchedLine0, 76)

			p.cleared = false
		}

		if utils.Test(v, 6) {
			p.WindowTileMap = 1
		} else {
			p.WindowTileMap = 0
		}
		p.WindowEnabled = utils.Test(v, 5)
		if utils.Test(v, 4) {
			p.TileDataAddress = 0x8000
			p.isSigned = false // TODO rename to something more appropriate
		} else {
			p.TileDataAddress = 0x8800
			p.isSigned = true
		}

		if utils.Test(v, 3) {
			p.BackgroundTileMap = 1
		} else {
			p.BackgroundTileMap = 0
		}

		p.SpriteSize = 8 + uint8(utils.Val(v, 2))*8
		p.SpriteEnabled = utils.Test(v, 1)
		p.BackgroundEnabled = utils.Test(v, 0)

		return v
	})
	b.ReserveAddress(types.STAT, func(v byte) byte {
		// writing to STAT briefly enables all STAT interrupts
		// but only on DMG. Road Rash relies on this bug, so
		// maybe add a warning for users trying to play Road
		// Rash in CGB mode
		if !p.b.IsGBC() {
			oldStat := p.status
			p.status = 0xff
			p.statUpdate()
			p.status = oldStat
		}

		// clear INT bits from stat
		p.status = p.status&0b1000_0111 | v&0b0111_1000
		p.statUpdate()

		return types.Bit7 | p.status | p.mode
	})
	b.ReserveLazyReader(types.STAT, func() byte {
		//fmt.Printf("%02x %d\n", types.Bit7|p.status|p.mode, p.s.Until(scheduler.PPUVRAMTransfer))
		return types.Bit7 | p.status | p.mode
	})
	b.ReserveAddress(types.SCY, func(v byte) byte {
		// do we need to force a re-render on background
		if b.Get(types.SCY) != v {
			p.dirtyBackground(scy)
		}

		return v
	})
	b.ReserveAddress(types.SCX, func(v byte) byte {
		// do we need to force a re-render on background
		if b.Get(types.SCX) != v {
			p.dirtyBackground(scx)
		}

		return v
	})
	b.ReserveAddress(types.LY, func(v byte) byte {
		// any write to LY resets to 0
		return 0
	})
	b.ReserveSetAddress(types.LY, func(v any) {
		p.currentLine = v.(uint8)
		p.b.Set(types.LY, v.(uint8))
	})
	b.ReserveAddress(types.LYC, func(v byte) byte {
		p.lyCompare = v
		// do we need to force a re-render on background
		if b.Get(types.LYC) != v {
			p.dirtyBackground(lyc)
		}
		p.statUpdate()
		return v
	})
	b.ReserveAddress(types.BGP, func(v byte) byte {
		// do we need to force a re-render on background?
		if v == b.Get(types.BGP) {
			return v
		}

		p.dirtyBackground(bgp)

		// if launched without a boot ROM, then check to see if a colourisation palette is loaded
		if p.BGColourisationPalette != nil {
			p.ColourPalette.Palettes[0] = palette.ByteToPalette(*p.BGColourisationPalette, v)
		} else {
			p.ColourPalette.Palettes[0] = palette.ByteToPalette(p.ColourPalettes[palette.Greyscale], v)
		}

		return v
	})
	b.ReserveAddress(types.OBP0, func(v byte) byte {
		// do we need to force a re-render on background?
		if v == b.Get(types.OBP0) {
			return v
		}
		p.dirtyBackground(obp0)

		if p.OBJ0ColourisationPalette != nil {
			p.ColourSpritePalette.Palettes[0] = palette.ByteToPalette(*p.OBJ0ColourisationPalette, v)
		} else {
			p.ColourSpritePalette.Palettes[0] = palette.ByteToPalette(p.ColourPalettes[palette.Greyscale], v)
		}

		return v
	})
	b.ReserveAddress(types.OBP1, func(v byte) byte {
		// do we need to force a re-render on background?
		if v == b.Get(types.OBP1) {
			return v
		}

		p.dirtyBackground(obp1)

		if p.OBJ1ColourisationPalette != nil {
			p.ColourSpritePalette.Palettes[1] = palette.ByteToPalette(*p.OBJ1ColourisationPalette, v)
		} else {
			p.ColourSpritePalette.Palettes[1] = palette.ByteToPalette(p.ColourPalettes[palette.Greyscale], v)
		}

		return v
	})
	b.ReserveAddress(types.WY, func(v byte) byte {
		return v
	})
	b.ReserveAddress(types.WX, func(v byte) byte {
		return v
	})

	b.WhenGBC(func() {
		// special address handler for colourisation (not on real hardware)
		b.ReserveAddress(0xFF7F, func(b byte) byte {
			if p.BGColourisationPalette != nil {
				return 0xff
			}
			p.BGColourisationPalette = &palette.Palette{}
			p.OBJ0ColourisationPalette = &palette.Palette{}
			p.OBJ1ColourisationPalette = &palette.Palette{}
			*p.BGColourisationPalette = p.ColourPalette.Palettes[0]
			*p.OBJ0ColourisationPalette = p.ColourSpritePalette.Palettes[0]
			*p.OBJ1ColourisationPalette = p.ColourSpritePalette.Palettes[1]
			return 0xff
		})
		// setup CGB only registers
		b.ReserveAddress(types.BCPS, func(v byte) byte {
			if p.b.IsGBCCart() || p.b.IsBooting() {
				p.ColourPalette.SetIndex(v)
				p.dirtyBackground(bcps)
				return p.ColourPalette.GetIndex() | 0x40
			}

			return 0xff

		})
		b.ReserveSetAddress(types.BCPS, func(a any) {
			p.ColourPalette.SetIndex(a.(byte))
			p.b.Set(types.BCPS, p.ColourPalette.GetIndex()|0x40)
		})
		b.ReserveAddress(types.BCPD, func(v byte) byte {
			if p.b.IsGBCCart() || p.b.IsBooting() {
				p.ColourPalette.Write(v)
				p.dirtyBackground(bcpd)

				// update bcps
				p.b.Set(types.BCPS, p.ColourPalette.GetIndex()|0x40)
				return p.ColourPalette.Read()
			}

			return 0xff

		})
		b.Set(types.BCPD, p.ColourPalette.Read())
		b.ReserveLazyReader(types.BCPD, func() byte {
			if !p.b.IsGBCCart() || p.mode == lcd.VRAM {
				return 0xff
			}

			return p.ColourPalette.Read()
		})
		b.ReserveAddress(types.OCPS, func(v byte) byte {
			if p.mode != lcd.VRAM {
				p.ColourSpritePalette.SetIndex(v)
				p.dirtyBackground(ocps)
				p.b.Set(types.OCPD, p.ColourSpritePalette.Read())
				return p.ColourSpritePalette.GetIndex() | 0x40
			}
			return 0xFF
		})
		b.ReserveSetAddress(types.OCPS, func(a any) {
			// bc some models boot into VRAM mode
			p.ColourSpritePalette.SetIndex(a.(byte))
			p.dirtyBackground(ocps)
			p.b.Set(types.OCPD, p.ColourSpritePalette.Read())
			p.b.Set(types.OCPS, a.(byte)|0x40)
		})
		b.ReserveAddress(types.OCPD, func(v byte) byte {
			if p.b.IsGBCCart() || p.b.IsBooting() {
				p.ColourSpritePalette.Write(v)
				p.dirtyBackground(ocpd)
				p.b.Set(types.OCPS, p.ColourSpritePalette.GetIndex()|0x40)
				return p.ColourSpritePalette.Read()
			}

			return 0xff

		})
		b.Set(types.OCPD, p.ColourSpritePalette.Read())

	})

	s.RegisterEvent(scheduler.PPUStartHBlank, p.startHBlank)
	s.RegisterEvent(scheduler.PPUStartGlitchedLine0, p.startGlitchedFirstLine)
	s.RegisterEvent(scheduler.PPUMiddleGlitchedLine0, p.middleGlitchedLine0)
	s.RegisterEvent(scheduler.PPUContinueGlitchedLine0, p.continueGlitchedFirstLine)
	s.RegisterEvent(scheduler.PPUEndGlitchedLine0, p.endGlitchedLine0)
	s.RegisterEvent(scheduler.PPUHBlank, p.endHBlank)
	s.RegisterEvent(scheduler.PPUVRAMTransfer, p.endVRAMTransfer)
	s.RegisterEvent(scheduler.PPUStartOAMSearch, p.startOAM)
	s.RegisterEvent(scheduler.PPUContinueOAMSearch, p.continueOAM)
	s.RegisterEvent(scheduler.PPUPrepareEndOAMSearch, func() {
		p.b.RLock(io.VRAM)
		p.b.WUnlock(io.OAM)

		// schedule end of OAM search for (4 cycles later)
		p.s.ScheduleEvent(scheduler.PPUEndOAMSearch, 4)
	})
	s.RegisterEvent(scheduler.PPUEndOAMSearch, p.endOAM)
	s.RegisterEvent(scheduler.PPUStartVBlank, p.startVBlank)
	s.RegisterEvent(scheduler.PPUContinueVBlank, p.continueVBlank)
	s.RegisterEvent(scheduler.PPULine153Continue, p.continueLine153)
	s.RegisterEvent(scheduler.PPULine153End, p.endLine153)
	s.RegisterEvent(scheduler.PPUEndFrame, p.endFrame)

	s.RegisterEvent(scheduler.PPUHBlankInterrupt, func() {
		p.modeToInterrupt = lcd.HBlank
		p.statUpdate()
		p.modeToInterrupt = lcd.VRAM

		p.s.ScheduleEvent(scheduler.PPUVRAMTransfer, 4)
	})
	s.RegisterEvent(scheduler.HDMA, func() {
		if p.b.IsGBCCart() {
			p.b.HandleHDMA()
		}
	})

	b.ReserveBlockWriter(0x8000, p.writeVRAM)
	b.ReserveBlockWriter(0x9000, p.writeVRAM)

	// initialize tile data
	for i := 0; i < 2; i++ {
		for j := 0; j < len(p.TileData[0]); j++ {
			p.TileData[i][j] = Tile{}
		}
	}

	// initialize tile map
	for i := 0; i < 2; i++ {
		for j := 0; j < len(p.TileMaps); j++ {
			p.TileMaps[i] = NewTileMap()
		}
	}

	// setup LUTs
	for b1 := 0; b1 <= 255; b1++ {
		for b2 := 0; b2 <= 255; b2++ {
			for xPos := 0; xPos < 8; xPos++ {
				p.colourNumberLUT[b1][b2][xPos] = uint8((b1 >> (7 - xPos) & 0x1) | ((b2 >> (7 - xPos) & 0x1) << 1))
				p.reversedColourNumberLUT[b1][b2][xPos] = uint8((b1 >> xPos & 0x1) | ((b2 >> xPos & 0x1) << 1))
			}
		}
	}

	// setup line changed
	for i := 0; i < len(p.backgroundLineChanged); i++ {
		p.backgroundLineChanged[i] = true
	}

	p.ColourPalette = palette.NewCGBPallette()
	p.ColourSpritePalette = palette.NewCGBPallette()

	return p
}

// startGlitchedFirstLine is called 76 cycles after the PPU is enabled,
// performing the first line of the screen in a glitched manner, accurate
// to the real hardware.
func (p *PPU) startGlitchedFirstLine() {
	p.statUpdate() // this occurs before the mode change, mode should be 255 here
	p.modeToInterrupt = lcd.VRAM

	p.s.ScheduleEvent(scheduler.PPUContinueGlitchedLine0, 4)
}

// continueGlitchedFirstLine is called 4 cycles after startGlitchedFirstLine,
// continuing the first line of the screen in a glitched manner, accurate
// to the real hardware.
func (p *PPU) continueGlitchedFirstLine() {
	// OAM & VRAM are blocked until the end of VRAM transfer
	p.b.Lock(io.OAM)
	p.b.Lock(io.VRAM)
	p.mode = lcd.VRAM

	p.s.ScheduleEvent(scheduler.PPUMiddleGlitchedLine0, 172)
}

func (p *PPU) middleGlitchedLine0() {
	p.mode = 0
	p.modeToInterrupt = 0
	p.statUpdate()

	p.b.Unlock(io.OAM)
	p.b.Unlock(io.VRAM)

	p.s.ScheduleEvent(scheduler.PPUEndGlitchedLine0, 196)
}

func (p *PPU) endGlitchedLine0() {
	p.modeToInterrupt = 2
	p.s.ScheduleEvent(scheduler.PPUHBlank, 4)
}

func (p *PPU) endHBlank() {
	// increment current scanline
	p.currentLine++
	p.modeToInterrupt = lcd.OAM

	// if we are on line 144, we are entering VBlank
	if p.currentLine == 144 {
		p.RefreshScreen = true
		p.notifyFrame()
		if p.backgroundDirty {
			for i := 0; i < ScreenHeight; i++ {
				p.backgroundLineRendered[i] = false
			}
		}
		p.backgroundDirty = false

		// was the LCD just turned on? (the Game Boy never receives the first frame after turning on the LCD)
		if !p.cleared {
			p.renderBlank()
		}

		p.startVBlank()
	} else {
		// go to OAM search
		p.startOAM()
	}
}

func (p *PPU) endVRAMTransfer() {
	p.mode = lcd.HBlank
	p.modeToInterrupt = lcd.HBlank

	p.b.Unlock(io.OAM)
	p.b.Unlock(io.VRAM)

	p.s.ScheduleEvent(scheduler.PPUStartHBlank, 4)
}

func (p *PPU) startHBlank() {
	p.statUpdate()

	p.renderScanline()

	p.s.ScheduleEvent(scheduler.HDMA, 8)

	// schedule end of HBlank
	p.s.ScheduleEvent(scheduler.PPUHBlank, uint64(scrollXHblank[p.b.Get(types.SCX)&0x7]))
}

// startOAM is performed on the first cycle of lines 0 to 143, and performs
// the OAM search for the current line. The OAM search lasts until cycle 88,
// when Mode 3 (VRAM) is entered.
//
// Lines 0 - 144:
//
//	OAM Search: 4 -> 84
func (p *PPU) startOAM() {
	p.b.Set(types.LY, p.currentLine) // update LY

	p.mode = lcd.HBlank
	// OAM STAT int occurs 1-M cycle before STAT changes, except on line 0
	if p.currentLine != 0 {
		p.modeToInterrupt = 2
		p.lyForComparison = 0xffff
	} else { // line 0
		p.lyForComparison = 0
	}

	// update STAT
	p.statUpdate()

	// OAM read is blocked until the end of OAM search,
	// OAM write is not blocked for another 4 cycles
	p.b.RLock(io.OAM)
	p.b.WUnlock(io.OAM)

	p.s.ScheduleEvent(scheduler.PPUContinueOAMSearch, 4)
}

// continueOAM is performed 4 cycles after startOAM, and performs the
// rest of the OAM search.
func (p *PPU) continueOAM() {
	p.mode = lcd.OAM
	p.lyForComparison = uint16(p.currentLine)
	p.modeToInterrupt = lcd.OAM
	p.statUpdate()

	p.modeToInterrupt = 255
	p.statUpdate()

	p.b.WLock(io.OAM)

	p.s.ScheduleEvent(scheduler.PPUPrepareEndOAMSearch, 76)
}

// endOAM is performed 80 cycles after startOAM, and performs the
// rest of the OAM search.
func (p *PPU) endOAM() {
	p.mode = lcd.VRAM
	p.modeToInterrupt = lcd.VRAM
	p.statUpdate()

	p.b.Lock(io.OAM)
	p.b.Lock(io.VRAM)

	// schedule end of VRAM search
	p.s.ScheduleEvent(scheduler.PPUHBlankInterrupt, uint64(scrollXvRAM[p.b.Get(types.SCX)&0x7]))
}

func (p *PPU) WriteCorruptionOAM() {
	/*fmt.Println("corrupting oam")
	return
	// determine which row of the OAM we are on
	// by getting the cycles we have until the end of OAM search
	cyclesUntilEndOAM := 80 - p.s.Until(scheduler.PPUEndOAMSearch)

	// each row is 4 ticks long and made up of 8 bytes (4 words)
	row := cyclesUntilEndOAM / 4

	if row < 2 { // the first 2 rows are not affected by the corruption
		return
	}

	// we need to get the 3 words that make up the corruption
	// the first word is the first word of the current row
	// the second word is the first word in the preceding row
	// the third word is the 3rd word in the preceding row
	// these 3 words then get corrupted by the bitwise glitch
	// and overwrite the first word of the current row
	// the last three words of the current row are then overwritten
	// with the preceding row's last three words
	a := uint16(p.oam.data[row*4]) | uint16(p.oam.data[row*4+1])<<8
	b := uint16(p.oam.data[row*4-8]) | uint16(p.oam.data[row*4-7])<<8
	c := uint16(p.oam.data[row*4-6]) | uint16(p.oam.data[row*4-5])<<8

	// perform the bitwise glitch
	newValue := bitwiseGlitch(a, b, c)

	// replace the first word of the current row with the new value
	p.oam.data[row*4] = byte(newValue)
	p.oam.data[row*4+1] = byte(newValue >> 8)

	// replace the last 3 words of the row from the preceding row
	p.oam.data[row*4-6] = p.oam.data[row*4-2]
	p.oam.data[row*4-5] = p.oam.data[row*4-1]
	p.oam.data[row*4-4] = p.oam.data[row*4]
	p.oam.data[row*4-3] = p.oam.data[row*4+1]
	p.oam.data[row*4-2] = p.oam.data[row*4+2]
	p.oam.data[row*4-1] = p.oam.data[row*4+3]

	//panic(fmt.Sprintf("OAM corruption: row %d %d cycles until end of OAM search %s", row, cyclesUntilEndOAM, p.s.String()))*/
}

func bitwiseGlitch(a, b, c uint16) uint16 {
	return ((a ^ c) & (b ^ c)) ^ c
}

// startVBlank is performed on the first cycle of each line 144 to 152, and
// performs the VBlank period for the current line. The VBlank period lasts
// until for 456 * 10 cycles, when the PPU enters Mode 2 (OAM search) on
// line 153 (PPU be like line 0, no line 153. you know, line 0, not the line 153 it's the next line :)).
func (p *PPU) startVBlank() {
	// should we start line 153?
	if p.currentLine == 153 {
		p.startLine153()
		return
	}

	p.lyForComparison = 0xffff
	p.statUpdate()

	// set the LY register to current scanline
	p.b.Set(types.LY, p.currentLine)

	if p.currentLine == 144 {
		p.modeToInterrupt = lcd.OAM

		// trigger vblank interrupt
		if p.b.Model() != types.CGBABC && p.b.Model() != types.CGB0 {
			p.b.RaiseInterrupt(io.VBlankINT)
		}
	}
	p.statUpdate()

	p.s.ScheduleEvent(scheduler.PPUContinueVBlank, 4)
}

func (p *PPU) continueVBlank() {
	p.lyForComparison = uint16(p.currentLine)
	p.statUpdate()
	if p.currentLine == 144 {
		p.mode = lcd.VBlank

		// trigger vblank interrupt
		if p.b.Model() == types.CGBABC || p.b.Model() == types.CGB0 {
			p.b.RaiseInterrupt(io.VBlankINT)
		}

		// entering vblank also triggers the OAM STAT interrupt if enabled
		if !p.statInterruptLine && p.status&0x20 != 0 {
			p.b.RaiseInterrupt(io.LCDINT)
		}
		p.modeToInterrupt = lcd.VBlank
		p.statUpdate()
	}

	p.s.ScheduleEvent(scheduler.PPUStartVBlank, 452)

	// start vblank for next line
	// line 153 is a special case
	p.currentLine++
}

func (p *PPU) startLine153() {
	p.b.Set(types.LY, 153)
	p.lyForComparison = 0xffff

	p.statUpdate()

	p.s.ScheduleEvent(scheduler.PPULine153Continue, 4)
}

func (p *PPU) continueLine153() {
	p.b.Set(types.LY, 0)
	p.lyForComparison = 153
	p.statUpdate()

	p.s.ScheduleEvent(scheduler.PPULine153End, 4)
}

func (p *PPU) endLine153() {
	p.b.Set(types.LY, 0)
	p.lyForComparison = 0xffff
	p.statUpdate()

	p.s.ScheduleEvent(scheduler.PPUEndFrame, 4)
}

func (p *PPU) endFrame() {
	p.lyForComparison = 0
	p.statUpdate()
	p.currentLine = 0
	p.windowInternal = 0

	p.s.ScheduleEvent(scheduler.PPUStartOAMSearch, 444)
}

func (p *PPU) DumpTileMaps(tileMap1, tileMap2 *image.RGBA, gap int) {
	// draw tilemap (0x9800 - 0x9BFF)
	for i := uint8(0); i < 32; i++ {
		for j := uint8(0); j < 32; j++ {
			tileEntry := p.calculateTileID(j, i, 0)
			// get tile data
			tile := p.TileData[tileEntry.Attributes.VRAMBank][tileEntry.GetID(p.isSigned)]
			tile.Draw(tileMap1, int(i)*(8+gap), int(j)*(8+gap), p.ColourPalette.Palettes[0])
		}
	}

	// draw tilemap (0x9C00 - 0x9FFF)
	for i := uint8(0); i < 32; i++ {
		for j := uint8(0); j < 32; j++ {
			tileEntry := p.calculateTileID(j, i, 1)

			// get tile data
			tile := p.TileData[tileEntry.Attributes.VRAMBank][tileEntry.GetID(p.isSigned)]
			tile.Draw(tileMap2, int(i)*(8+gap), int(j)*(8+gap), p.ColourPalette.Palettes[0])
		}
	}
}

func (p *PPU) writeVRAM(address uint16, value uint8) {
	address &= 0x1FFF

	// are we writing to the tile data?
	if address <= 0x17FF {
		p.updateTile(address, value)
		// update the tile data
	} else if address <= 0x1FFF {
		if p.b.Get(types.VBK)&1 == 0 {
			// which offset are we writing to?
			if address >= 0x1800 && address <= 0x1BFF {
				// tilemap 0
				p.updateTileMap(address, 0, value)
			}
			if address >= 0x1C00 && address <= 0x1FFF {
				// tilemap 1
				p.updateTileMap(address, 1, value)
			}
		}
		if p.b.Get(types.VBK)&1 == 1 {
			// update the tile Attributes
			if address >= 0x1800 && address <= 0x1BFF {
				// tilemap 0
				p.updateTileAttributes(address, 0, value)
			}
			if address >= 0x1C00 && address <= 0x1FFF {
				// tilemap 1
				p.updateTileAttributes(address, 1, value)
			}
		}
	}

}

// updateTile updates the tile at the given address
func (p *PPU) updateTile(address uint16, value uint8) {
	// get the tile address
	index := address & 0x1FFE // only the lower 13 bits are used

	// get the tileID
	tileID := index >> 4 // divide by 16

	// get the tile row
	row := (address >> 1) & 0x7

	p.TileData[p.b.Get(types.VBK)&1][tileID][row+((address%2)*8)] = value
	p.TileChanged[p.b.Get(types.VBK)&1][tileID] = true

	p.dirtyBackground(tile)
	// recache tilemap
	//p.recacheByID(tileID)
}

func (p *PPU) updateTileMap(address uint16, tilemapIndex, value uint8) {
	// determine the y and x position
	y := (address / 32) & 0x1F
	x := address & 0x1F

	p.TileMaps[tilemapIndex][y][x].id = uint16(value)

	p.dirtyBackground(tileMap)
}

func (p *PPU) updateTileAttributes(index uint16, tilemapIndex uint8, value uint8) {
	// panic(fmt.Sprintf("updating tile %x with %b", index, value))
	// determine the y and x position
	y := (index / 32) & 0x1F
	x := index & 0x1F

	// update the tilemap
	t := TileAttributes{}
	t.BGPriority = value&0x80 != 0
	t.YFlip = value&0x40 != 0
	t.XFlip = value&0x20 != 0
	t.CGBPaletteNumber = value & 0b111
	t.VRAMBank = value >> 3 & 0x1

	p.TileMaps[tilemapIndex][y][x].Attributes = t
	p.TileMaps[tilemapIndex][y][x].Tile = p.TileData[t.VRAMBank][p.TileMaps[tilemapIndex][y][x].id]
	// p.recacheTile(x, y, tilemapIndex)

	p.dirtyBackground(tileAttr)
}

func (p *PPU) statUpdate() {
	// do nothing if the LCD is disabled
	if !p.Enabled {
		return
	}

	// TODO handle DMA active here

	// get previous interrupt state
	prevInterruptLine := p.statInterruptLine

	// handle LY=LYC
	if p.lyForComparison != 0xffff && uint8(p.lyForComparison) == p.lyCompare {
		p.lycInterruptLine = true
		p.status |= types.Bit2
	} else {
		if p.lyForComparison != 0xffff {
			p.lycInterruptLine = false
		}
		p.status &^= types.Bit2
	}

	// handle mode to interrupt
	switch p.modeToInterrupt {
	case lcd.HBlank:
		p.statInterruptLine = p.status&0x08 != 0
	case lcd.VBlank:
		p.statInterruptLine = p.status&0x10 != 0
	case lcd.OAM:
		p.statInterruptLine = p.status&0x20 != 0
	default:
		p.statInterruptLine = false
	}

	// LY=LYC interrupt
	if p.status&0x40 != 0 && p.lycInterruptLine {
		p.statInterruptLine = true
	}

	// trigger interrupt if needed
	if p.statInterruptLine && !prevInterruptLine {
		p.b.RaiseInterrupt(io.LCDINT)
	}
}

var (
	scrollXHblank = [8]uint16{196, 192, 192, 192, 192, 188, 188, 188}
	scrollXvRAM   = [8]uint16{168, 172, 172, 172, 172, 176, 176, 176}
)

func (p *PPU) renderScanline() {
	if p.b.Get(types.LY) >= ScreenHeight {
		return
	}
	if (!p.backgroundLineRendered[p.b.Get(types.LY)] || p.oam.dirtyScanlines[p.b.Get(types.LY)] || p.backgroundDirty) && (p.BackgroundEnabled || p.b.IsGBCCart()) {
		p.renderBackgroundScanline()

	}

	if !p.Debug.WindowDisabled {
		if p.WindowEnabled {
			p.renderWindowScanline()
		}
	}

	if !p.Debug.SpritesDisabled {
		if p.SpriteEnabled {
			p.renderSpritesScanline(p.b.Get(types.LY))
		}
	}
}

func (p *PPU) renderBlank() {
	for y := uint8(0); y < ScreenHeight; y++ {
		for x := uint8(0); x < ScreenWidth; x++ {
			p.PreparedFrame[y][x] = p.ColourPalette.Palettes[0][0] // TODO handle GBC
		}
	}
	p.cleared = true
}

func (p *PPU) renderWindowScanline() {
	// do nothing if window is out of bounds
	if p.b.Get(types.LY) < p.b.Get(types.WY) {
		return
	} else if p.b.Get(types.WX) > ScreenWidth {
		return
	} else if p.b.Get(types.WY) > ScreenHeight {
		return
	}

	yPos := p.windowInternal

	// get the initial x pos and pixel pos
	xPos := p.b.Get(types.WX) - 7
	xPixelPos := xPos % 8

	// get the tile map row
	tileMapRow := p.TileMaps[p.WindowTileMap][yPos>>3]

	// get the first tile entry
	tileEntry := tileMapRow[xPos>>3]
	tileID := tileEntry.GetID(p.isSigned)

	yPixelPos := yPos

	// get the first lot of tile data
	tileData := p.TileData[tileEntry.Attributes.VRAMBank][tileID]
	if tileEntry.Attributes.YFlip {
		yPixelPos = 7 - yPixelPos
	}
	yPixelPos %= 8

	// get the 2 bytes that make up a row of 8 pixels
	b1 := tileData[yPixelPos]
	b2 := tileData[yPixelPos+8]

	// create the colour lookup table
	var colourLUT [8]uint8
	if tileEntry.Attributes.XFlip {
		colourLUT = p.reversedColourNumberLUT[b1][b2]
	} else {
		colourLUT = p.colourNumberLUT[b1][b2]
	}

	priority := tileEntry.Attributes.BGPriority

	pal := p.ColourPalette.Palettes[tileEntry.Attributes.CGBPaletteNumber]

	scanline := &p.PreparedFrame[p.b.Get(types.LY)]
	bgPriorityLine := &p.tileBgPriority[p.b.Get(types.LY)]

	for i := uint8(0); i < ScreenWidth; i++ {
		// did we just finish a tile?
		if xPixelPos == 8 { // we don't need to do this for the last pixel
			// increment xPos by the number of pixels we've just rendered
			xPos = i - (p.b.Get(types.WX) - 7)

			// get the next tile entry
			tileEntry = tileMapRow[xPos>>3]
			tileID = tileEntry.GetID(p.isSigned)

			if p.b.IsGBC() {
				pal = p.ColourPalette.Palettes[tileEntry.Attributes.CGBPaletteNumber]
			}

			tileData = p.TileData[tileEntry.Attributes.VRAMBank][tileID]

			if tileEntry.Attributes.YFlip {
				yPixelPos = 7 - yPos
			}
			yPixelPos %= 8
			b1 = tileData[yPixelPos]
			b2 = tileData[yPixelPos+8]
			if tileEntry.Attributes.XFlip {
				colourLUT = p.reversedColourNumberLUT[b1][b2]
			} else {
				colourLUT = p.colourNumberLUT[b1][b2]
			}

			priority = tileEntry.Attributes.BGPriority

			// reset the x pixel pos
			xPixelPos = 0
		}

		// don't render until we're in the window
		if i >= p.b.Get(types.WX)-7 {
			p.colorNumber[i] = colourLUT[xPixelPos]
			scanline[i] = pal[colourLUT[xPixelPos]]
			bgPriorityLine[i] = priority
		}

		xPixelPos++

	}

	p.windowInternal++
}

func (p *PPU) renderBackgroundScanline() {
	// get the initial y pos and pixel pos
	yPos := p.b.Get(types.LY) + p.b.Get(types.SCY)

	// get the initial x pos and pixel pos
	xPos := p.b.Get(types.SCX)
	xPixelPos := xPos & 7

	// get the first tile entry
	tileEntry := p.TileMaps[p.BackgroundTileMap][yPos>>3][xPos>>3]
	tileID := tileEntry.GetID(p.isSigned)

	yPixelPos := yPos
	// get the first lot of tile data
	tileData := p.TileData[tileEntry.Attributes.VRAMBank][tileID]
	if tileEntry.Attributes.YFlip {
		yPixelPos = 7 - yPos
	}
	yPixelPos %= 8

	// get the 2 bytes that make up a row of 8 pixels
	b1 := tileData[yPixelPos]
	b2 := tileData[yPixelPos+8]

	// create the colour lookup table
	var colourLUT [8]uint8
	if tileEntry.Attributes.XFlip {
		colourLUT = p.reversedColourNumberLUT[b1][b2]
	} else {
		colourLUT = p.colourNumberLUT[b1][b2]
	}

	priority := tileEntry.Attributes.BGPriority

	pal := p.ColourPalette.Palettes[tileEntry.Attributes.CGBPaletteNumber]

	bgPriorityLine := &p.tileBgPriority[p.b.Get(types.LY)]
	var scanline [ScreenWidth][3]uint8

	for i := uint8(0); i < ScreenWidth; i++ {
		// set scanline using unsafe to copy 4 bytes at a time
		//scanline[i] = pal[colourLUT[xPixelPos]]
		*(*uint32)(unsafe.Pointer(&scanline[i])) = *(*uint32)(unsafe.Pointer(&pal[colourLUT[xPixelPos]]))

		bgPriorityLine[i] = priority
		p.colorNumber[i] = colourLUT[xPixelPos]

		xPixelPos++

		// did we just finish a tile?
		if xPixelPos == 8 { // we don't need to do this for the last pixel
			// increment xPos by the number of pixels we've just rendered
			xPos += 8

			// get the next tile entry
			tileEntry = p.TileMaps[p.BackgroundTileMap][yPos>>3][xPos>>3]
			tileID = tileEntry.GetID(p.isSigned)

			if p.b.IsGBC() {
				pal = p.ColourPalette.Palettes[tileEntry.Attributes.CGBPaletteNumber]
			}

			tileData = p.TileData[tileEntry.Attributes.VRAMBank][tileID]

			if tileEntry.Attributes.YFlip {
				yPixelPos = 7 - yPos
			}
			yPixelPos %= 8
			b1 = tileData[yPixelPos]
			b2 = tileData[yPixelPos+8]
			if tileEntry.Attributes.XFlip {
				colourLUT = p.reversedColourNumberLUT[b1][b2]
			} else {
				colourLUT = p.colourNumberLUT[b1][b2]
			}

			priority = tileEntry.Attributes.BGPriority

			// reset the x pixel pos
			xPixelPos = 0
		}
	}

	p.backgroundLineRendered[p.b.Get(types.LY)] = true
	p.oam.dirtyScanlines[p.b.Get(types.LY)] = false

	// update scanline in frame
	p.PreparedFrame[p.b.Get(types.LY)] = scanline
}

// calculateTileID calculates the tile ID for the current scanline
func (p *PPU) calculateTileID(tilemapOffset, lineOffset uint8, mapOffset uint8) TileMapEntry {
	// get the tile entry from the tilemap
	tileEntry := p.TileMaps[mapOffset][tilemapOffset][lineOffset]

	return tileEntry
}

func (p *PPU) renderSpritesScanline(scanline uint8) {
	if p.b.OAMChanged() {
		p.b.OAMCatchup(p.oam.Write)
	}

	spriteXPerScreen := [ScreenWidth]uint8{}
	spriteCount := 0 // number of sprites on the current scanline (max 10)

	for _, sprite := range p.oam.Sprites {
		spriteY := sprite.Y
		spriteX := sprite.X

		if spriteY > scanline || spriteY+p.SpriteSize <= scanline {
			continue
		}
		if spriteCount >= 10 {
			break
		}
		spriteCount++

		tilerowIndex := scanline - spriteY
		if sprite.flipY {
			tilerowIndex = p.SpriteSize - tilerowIndex - 1
		}
		tilerowIndex %= 8
		tileID := uint16(sprite.TileID)
		if p.SpriteSize == 16 {
			if scanline-spriteY < 8 {
				if sprite.flipY {
					tileID |= 0x01
				} else {
					tileID &= 0xFE
				}
			} else {
				if sprite.flipY {
					tileID &= 0xFE
				} else {
					tileID |= 0x01
				}
			}
		}

		// get the 2 bytes of data that make up the row of the tile
		b1 := p.TileData[sprite.vRAMBank][tileID][tilerowIndex]
		b2 := p.TileData[sprite.vRAMBank][tileID][tilerowIndex+8]

		// get the colour lut
		var colourLUT [8]uint8
		if sprite.flipX {
			colourLUT = p.reversedColourNumberLUT[b1][b2]
		} else {
			colourLUT = p.colourNumberLUT[b1][b2]
		}
		var pal palette.Palette
		pal = p.ColourSpritePalette.Palettes[sprite.useSecondPalette]

		if p.b.IsGBCCart() {
			pal = p.ColourSpritePalette.Palettes[sprite.cgbPalette]
		}

		for x := uint8(0); x < 8; x++ {
			// skip if the sprite is out of bounds
			pixelPos := spriteX + x
			if pixelPos < 0 || pixelPos >= ScreenWidth {
				continue
			}

			// get the color of the pixel using the sprite palette
			color := colourLUT[x]

			// skip if the color is transparent
			if color == 0 {
				continue
			}

			// skip if the sprite doesn't have priority and the background is not transparent
			if !p.b.IsGBCCart() || p.BackgroundEnabled {
				if !(sprite.priority && !p.tileBgPriority[scanline][pixelPos]) {
					if p.colorNumber[pixelPos] != 0 {
						continue
					}
				}
			}

			if p.b.IsGBCCart() {
				// skip if the sprite doesn't have priority and the background is not transparent
				if spriteXPerScreen[pixelPos] != 0 {
					continue
				}
			} else {
				// skip if pixel is occupied by sprite with lower x coordinate
				if spriteXPerScreen[pixelPos] != 0 && spriteXPerScreen[pixelPos] <= spriteX+10 {
					continue
				}
			}

			p.PreparedFrame[scanline][pixelPos] = pal[color]

			// mark the pixel as occupied
			spriteXPerScreen[pixelPos] = spriteX + 10
		}
	}
}

func (p *PPU) ClearRefresh() {
	p.RefreshScreen = false
}

var _ types.Stater = (*PPU)(nil)

func (p *PPU) Load(s *types.State) {
	p.windowInternal = s.Read8()
	// load the vRAM data
	p.RefreshScreen = s.ReadBool()
	p.statInterruptDelay = s.ReadBool()
	p.ColourPalette.Load(s)
	p.ColourSpritePalette.Load(s)
}

func (p *PPU) Save(s *types.State) {
	s.Write8(p.windowInternal) // 1 byte
	s.WriteBool(p.RefreshScreen)
	s.WriteBool(p.statInterruptDelay)

	p.ColourPalette.Save(s)
	p.ColourSpritePalette.Save(s)
}

func (p *PPU) DumpRender(img *image.RGBA) {
	for y := 0; y < ScreenHeight; y++ {
		for x := 0; x < ScreenWidth; x++ {
			// draw the frame
			col := p.PreparedFrame[y][x]
			img.Set(x, y, color.RGBA{col[0], col[1], col[2], 255})

			if !p.backgroundLineRendered[y] {
				// mix RED with frame
				img.Set(x, y, combine(img.At(x, y), color.RGBA{255, 0, 0, 128}))
			} else if p.oam.spriteScanlinesColumn[y][x] {
				img.Set(x, y, color.RGBA{0, 255, 0, 128})
			} else if p.oam.spriteScanlines[y] {
				img.Set(x, y, color.RGBA{0, 0, 255, 128})
			}
		}
	}
}

func combine(c1, c2 color.Color) color.Color {
	r, g, b, a := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()

	return color.RGBA{
		uint8((r + r2) >> 9), // div by 2 followed by ">> 8"  is ">> 9"
		uint8((g + g2) >> 9),
		uint8((b + b2) >> 9),
		uint8((a + a2) >> 9),
	}
}

func (p *PPU) AttachNotifyFrame(fn func()) {
	p.notifyFrame = fn
}

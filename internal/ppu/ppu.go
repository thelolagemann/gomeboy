// Package ppu provides a programmable pixel unit for the DMG and CGB.
package ppu

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/ppu/lcd"
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"github.com/thelolagemann/go-gameboy/internal/ram"
	"github.com/thelolagemann/go-gameboy/internal/scheduler"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"image"
	"image/color"
)

const (
	// ScreenWidth is the width of the screen in pixels.
	ScreenWidth = 160
	// ScreenHeight is the height of the screen in pixels.
	ScreenHeight = 144
)

type PPU struct {
	*lcd.Controller
	CurrentScanline uint8
	lyCompare       uint8
	SpritePalettes  [2]palette.Palette

	// LCD
	status uint8
	mode   lcd.Mode

	// Background
	// It is made up of a 256x256 pixel map
	// of tiles. The map is divided into 32x32 tiles. Each tile is 8x8 pixels. As the
	// display only has 160x144 pixels, the background is scrolled to display
	// different parts of the map.

	// scrollX is the X position of the background.
	scrollX uint8
	// scrollY is the Y position of the background.
	scrollY              uint8
	Palette              palette.Palette
	compatibilityPalette palette.Palette

	// Window
	windowX        uint8
	windowY        uint8
	windowInternal uint8

	// CGB registers
	vRAMBank uint8

	oam                 *OAM
	vRAM                [2]*ram.RAM // Second bank only exists on CGB
	ColourPalette       *palette.CGBPalette
	ColourSpritePalette *palette.CGBPalette

	TileData [2][384]Tile // 384 tiles, 8x8 pixels each (double in CGB mode)
	tileMaps [2]TileMap   // 32x32 tiles, 8x8 pixels each

	irq *interrupts.Service

	PreparedFrame [ScreenHeight][ScreenWidth][3]uint8
	colorNumber   [ScreenWidth]uint8

	backgroundLineRendered [ScreenHeight]bool

	bus                *mmu.MMU
	statInterruptDelay bool
	RefreshScreen      bool
	DMA                *DMA
	delayedTick        bool

	tileBgPriority [ScreenHeight][ScreenWidth]bool
	isGBC          bool
	isGBCCompat    bool

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

	dirtiedLog   [65536]dirtyEvent
	lastDirty    uint16
	CurrentCycle uint64
	lastLine     bool

	hdma                              *HDMA
	s                                 *scheduler.Scheduler
	oamReadBlocked, oamWriteBlocked   bool
	vramReadBlocked, vramWriteBlocked bool
	lyForComparison                   uint8
	lycInterruptLine                  bool
	statInterruptLine                 bool
	modeToInterrupt                   uint8
	currentLine                       uint8
	vblankCounts                      uint8
	cyclesPassed                      uint64
	cyclesStart                       uint64
}

func (p *PPU) init() {
	// setup components
	p.Controller = lcd.NewController(func(writeFn func()) {
		// TODO lcdon_timing
		// cycle 0x11 (17) currently reports 0x87 (0b10000111) instead of 0x84 (0b10000100)
		// so we're entering mode 0 (hblank) too late?
		// "lcd-on is special, the oam scan period is 4 cycles shorter in the first line (and doesn't actually scan oam, the mode reads as 0)
		//[13:31]
		//or I guess it effectively starts at cycle 4 in the line " - @calc
		old := p.Raw
		wasOn := p.Enabled
		writeFn()

		// if the screen was turned off, clear the screen
		if wasOn && !p.Enabled {
			// the screen should not be turned off unless in vblank
			if p.mode != lcd.VBlank {
				panic("PPU: Screen was turned off while not in VBlank")
			}

			// deschedule all PPU events
			for i := scheduler.PPUHBlank; i < scheduler.PPUGlitchedLine0; i++ {
				p.s.DescheduleEvent(i)
			}

			// clear the screen
			p.renderBlank()

			// when the LCD is off, LY is 0, and STAT mode is 0
			p.CurrentScanline = 0
			p.currentLine = 0
			p.mode = lcd.HBlank

			// unblock OAM and VRAM reads/writes
			p.oamReadBlocked = false
			p.vramReadBlocked = false
			p.oamWriteBlocked = false
			p.vramWriteBlocked = false
		} else if !wasOn && p.Enabled {

			// reset LYC to compare against and clear coincidence flag
			p.lyForComparison = 0
			p.CurrentScanline = 0
			p.currentLine = 0

			p.modeToInterrupt = 255
			// perform STAT interrupt check
			p.statUpdate()

			p.s.EnableDebugLogging()
			// schedule the end of first line
			p.s.ScheduleEvent(scheduler.PPUStartGlitchedLine0, 76)
		}
		if old != p.Raw {
			p.dirtyBackground(lcdc)
		}

	})

	// setup registers
	types.RegisterHardware(
		types.STAT,
		func(v uint8) {
			p.status = v&0b0111_1000 | types.Bit7
			p.statUpdate()
		},
		func() uint8 {
			return p.status | p.mode
		},
		types.WithSet(func(v interface{}) {
			p.status = v.(uint8)&0b0111_1000 | types.Bit7
			p.mode = v.(uint8) & 0b11
		}))
	types.RegisterHardware(
		types.SCY,
		func(v uint8) {
			if p.scrollY != v {
				p.dirtyBackground(scy)
				p.scrollY = v
			}
		},
		func() uint8 {
			return p.scrollY
		},
	)
	types.RegisterHardware(
		types.SCX,
		func(v uint8) {
			if p.scrollX != v {
				p.dirtyBackground(scx)
				p.scrollX = v
			}
		},
		func() uint8 {
			return p.scrollX
		},
	)
	types.RegisterHardware(
		types.LY,
		func(v uint8) {
			// any write to LY resets the value to 0
			p.CurrentScanline = 0
		},
		func() uint8 {
			return p.CurrentScanline
		},
		types.WithSet(func(v interface{}) {
			p.CurrentScanline = v.(uint8)
		}),
	)
	types.RegisterHardware(
		types.LYC,
		func(v uint8) {
			p.lyCompare = v
			if p.lyCompare != v {
				p.dirtyBackground(lyc)

			}
			p.statUpdate()

		},
		func() uint8 {
			return p.lyCompare
		},
	)

	types.RegisterHardware(
		types.BGP,
		func(v uint8) {
			pal := palette.ByteToPalette(v)
			if p.Palette != pal {
				p.Palette = pal
				if p.isGBCCompat && !p.isGBC {
					// rewrite the palette
					for i := 0; i < 4; i++ {
						// get colour number from value
						palNum := v >> (i * 2) & 0x3
						p.compatibilityPalette[palNum] = p.ColourPalette.Palettes[0].GetColour(uint8(i))
					}
				}
				p.dirtyBackground(bgp)
			}
		},
		func() uint8 {
			return p.Palette.ToByte()
		},
		types.WithSet(func(v interface{}) {
			p.Palette = palette.ByteToPalette(v.(uint8)) // avoid the write operation above
		}),
	)
	types.RegisterHardware(
		types.OBP0,
		func(v uint8) {
			pal := palette.ByteToPalette(v)
			if p.SpritePalettes[0] != pal {
				p.SpritePalettes[0] = pal
				if p.isGBCCompat && !p.isGBC {
					// rewrite the palette
					for i := 0; i < 4; i++ {
						// get colour number from value
						palNum := v >> (i * 2) & 0x3
						p.SpritePalettes[0][palNum] = p.ColourSpritePalette.Palettes[0].GetColour(uint8(i))
					}
				}
				p.dirtyBackground(obp0)
			}
		},
		func() uint8 {
			return p.SpritePalettes[0].ToByte()
		},
	)
	types.RegisterHardware(
		types.OBP1,
		func(v uint8) {
			pal := palette.ByteToPalette(v)
			if p.SpritePalettes[1] != pal {
				p.SpritePalettes[1] = pal
				if p.isGBCCompat && !p.isGBC {
					// rewrite the palette
					for i := 0; i < 4; i++ {
						// get colour number from value
						palNum := v >> (i * 2) & 0x3
						p.SpritePalettes[1][palNum] = p.ColourSpritePalette.Palettes[1].GetColour(uint8(i))
					}
				}
				p.dirtyBackground(obp1)
			}
		},
		func() uint8 {
			return p.SpritePalettes[1].ToByte()
		},
	)
	types.RegisterHardware(
		types.WX,
		func(v uint8) {
			if p.windowX != v {
				p.windowX = v
			}
		},
		func() uint8 {
			return p.windowX
		},
	)
	types.RegisterHardware(
		types.WY,
		func(v uint8) {
			if p.windowY != v {
				p.windowY = v
			}
		},
		func() uint8 {
			return p.windowY
		},
	)

	// CGB registers
	types.RegisterHardware(
		types.VBK,
		func(v uint8) {
			if p.isGBCCompat {
				p.vRAMBank = v & types.Bit0
				// p.isDirty = true
			}
		},
		func() uint8 {
			if p.isGBCCompat {
				return p.vRAMBank | 0xFE
			}
			return 0xFF
		},
	)
	types.RegisterHardware(
		types.BCPS,
		func(v uint8) {
			if p.isGBCCompat {
				if p.colorPaletteUnlocked() {
					p.ColourPalette.SetIndex(v)
					p.dirtyBackground(bcps)
				}
			}
		},
		func() uint8 {
			if p.isGBCCompat {
				return p.ColourPalette.GetIndex() | types.Bit6
			}
			return 0xFF
		},
		types.WithSet(func(v interface{}) {
			p.ColourPalette.SetIndex(v.(uint8))
			for i := 0; i < 4; i++ {
				// get colour number from value
				palNum := v.(uint8) >> (i * 2) & 0x3
				p.compatibilityPalette[palNum] = p.ColourPalette.Palettes[0].GetColour(uint8(i))
			}
		}),
	)
	types.RegisterHardware(
		types.BCPD,
		func(v uint8) {
			if p.isGBCCompat && p.colorPaletteUnlocked() {
				p.ColourPalette.Write(v)
				p.dirtyBackground(bcpd)
			}
		},
		func() uint8 {
			if p.isGBC && p.colorPaletteUnlocked() {
				return p.ColourPalette.Read()
			}
			return 0xFF
		},
	)
	types.RegisterHardware(
		types.OCPS,
		func(v uint8) {
			if p.isGBCCompat && p.colorPaletteUnlocked() {
				p.ColourSpritePalette.SetIndex(v)
				p.dirtyBackground(ocps)
			}
		},
		func() uint8 {
			if p.isGBCCompat {
				return p.ColourSpritePalette.GetIndex() | 0x40
			}
			return 0xFF
		},
		types.WithSet(func(v interface{}) {
			p.ColourSpritePalette.SetIndex(v.(uint8))
		}),
	)
	types.RegisterHardware(
		types.OCPD,
		func(v uint8) {
			if p.isGBCCompat && p.colorPaletteUnlocked() {
				p.ColourSpritePalette.Write(v)
				p.dirtyBackground(ocpd)
			}
		},
		func() uint8 {
			if p.isGBC && p.colorPaletteUnlocked() {
				return p.ColourSpritePalette.Read()
			}
			return 0xFF
		},
	)

	// initialize tile data
	for i := 0; i < 2; i++ {
		for j := 0; j < len(p.TileData[0]); j++ {
			p.TileData[i][j] = Tile{}
		}
	}

	// initialize tile map
	for i := 0; i < 2; i++ {
		for j := 0; j < len(p.tileMaps); j++ {
			p.tileMaps[i] = NewTileMap()
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

	p.ColourPalette = palette.NewCGBPallette()
	p.ColourSpritePalette = palette.NewCGBPallette()
}

// TODO pass channel to send frame to
func (p *PPU) StartRendering() {
	p.isGBC = p.bus.IsGBC()
	p.isGBCCompat = p.bus.IsGBCCompat()
}

func New(mmu *mmu.MMU, irq *interrupts.Service, s *scheduler.Scheduler) *PPU {
	oam := NewOAM()
	p := &PPU{
		TileData: [2][384]Tile{},

		bus: mmu,
		irq: irq,
		oam: oam,
		vRAM: [2]*ram.RAM{
			ram.NewRAM(8192),
			ram.NewRAM(8192),
		},
		DMA: NewDMA(mmu, oam, s),
		s:   s,
	}
	p.hdma = NewHDMA(mmu, p.writeVRAM, s)

	p.init()

	s.RegisterEvent(scheduler.PPUStartGlitchedLine0, p.startGlitchedFirstLine)
	s.RegisterEvent(scheduler.PPUContinueGlitchedLine0, p.continueGlitchedFirstLine)
	s.RegisterEvent(scheduler.PPUHBlank, p.endHBlank)
	s.RegisterEvent(scheduler.PPUVRAMTransfer, p.endVRAMTransfer)
	s.RegisterEvent(scheduler.PPUStartOAMSearch, p.startOAM)
	s.RegisterEvent(scheduler.PPUContinueOAMSearch, p.continueOAM)
	s.RegisterEvent(scheduler.PPUEndOAMSearch, p.endOAM)
	s.RegisterEvent(scheduler.PPUStartVBlank, p.startVBlank)
	s.RegisterEvent(scheduler.PPUContinueVBlank, p.continueVBlank)
	s.RegisterEvent(scheduler.PPULine153Start, p.startLine153)
	s.RegisterEvent(scheduler.PPULine153Continue, p.continueLine153)
	s.RegisterEvent(scheduler.PPULine153End, p.endLine153)
	s.RegisterEvent(scheduler.PPUEndFrame, p.endFrame)
	s.RegisterEvent(scheduler.PPUVRAMLocked, func() {
		p.vramReadBlocked = true
		p.vramWriteBlocked = true // TODO make separate
	})
	s.RegisterEvent(scheduler.PPUVRAMUnlocked, func() {
		p.vramReadBlocked = false
		p.vramWriteBlocked = false // TODO make separate
	})
	s.RegisterEvent(scheduler.PPUOAMLocked, func() {
		p.oamReadBlocked = true
		p.oamWriteBlocked = true
	})
	s.RegisterEvent(scheduler.PPUOAMUnlocked, func() {
		p.oamWriteBlocked = false
	})

	s.RegisterEvent(scheduler.PPUHBlankInterrupt, func() {
		p.modeToInterrupt = lcd.HBlank
		p.statUpdate()
		p.modeToInterrupt = lcd.VRAM
	})

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
	p.oamReadBlocked = true
	p.vramReadBlocked = true
	p.oamWriteBlocked = true
	p.vramWriteBlocked = true

	p.mode = lcd.VRAM

	p.s.ScheduleEvent(scheduler.PPUVRAMTransfer, 172)
}

func (p *PPU) endHBlank() {
	// increment current scanline
	p.currentLine++
	p.modeToInterrupt = lcd.OAM

	if p.currentLine > 3 {
		p.s.DisableDebugLogging()
	}

	p.printStage("endHBlank")

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
		if !p.Cleared() {
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
	p.statUpdate()

	p.oamReadBlocked = false
	p.vramReadBlocked = false
	p.oamWriteBlocked = false
	p.vramWriteBlocked = false
	p.renderScanline()

	// schedule end of HBlank
	p.s.ScheduleEvent(scheduler.PPUHBlank, uint64(scrollXHblank[p.scrollX&0x7]))

	p.printStage("endVRAMTransfer")
}

// startOAM is performed on the first cycle of lines 0 to 143, and performs
// the OAM search for the current line. The OAM search lasts until cycle 88,
// when mode 3 (VRAM) is entered.
//
// Lines 0 - 144:
//
//	OAM Search: 4 -> 84
func (p *PPU) startOAM() {
	p.CurrentScanline = p.currentLine // update LY

	p.mode = lcd.HBlank
	// OAM STAT int occurs 1-M cycle before STAT changes, except on line 0
	if p.currentLine != 0 {
		p.modeToInterrupt = 2
		p.lyForComparison = 255
	} else { // line 0
		p.lyForComparison = 0
	}

	// update STAT
	p.statUpdate()

	// OAM read is blocked until the end of OAM search,
	// OAM write is not blocked for another 4 cycles
	p.oamReadBlocked = true
	p.oamWriteBlocked = false

	p.s.ScheduleEvent(scheduler.PPUContinueOAMSearch, 4)

	p.cyclesPassed = 0
	p.cyclesStart = p.s.Cycle()
	p.printStage("Prepare OAM")
}

// continueOAM is performed 4 cycles after startOAM, and performs the
// rest of the OAM search.
func (p *PPU) continueOAM() {
	p.mode = lcd.OAM
	p.lyForComparison = p.currentLine
	p.modeToInterrupt = lcd.OAM
	p.statUpdate()

	p.modeToInterrupt = 255
	p.statUpdate()

	p.oamWriteBlocked = true

	p.s.ScheduleEvent(scheduler.PPUVRAMLocked, 76)
	p.s.ScheduleEvent(scheduler.PPUOAMUnlocked, 76)

	// schedule end of OAM search for (80 cycles later)
	p.s.ScheduleEvent(scheduler.PPUEndOAMSearch, 80)

	p.printStage("Start OAM")
}

// endOAM is performed 80 cycles after startOAM, and performs the
// rest of the OAM search.
func (p *PPU) endOAM() {
	p.mode = lcd.VRAM
	p.modeToInterrupt = lcd.VRAM
	p.statUpdate()

	p.oamReadBlocked = true
	p.vramReadBlocked = true
	p.oamWriteBlocked = true
	p.vramWriteBlocked = true

	// schedule end of VRAM search
	p.s.ScheduleEvent(scheduler.PPUHBlankInterrupt, uint64(scrollXvRAM[p.scrollX&0x7])-4)
	p.s.ScheduleEvent(scheduler.PPUVRAMTransfer, uint64(scrollXvRAM[p.scrollX&0x7]))

	p.printStage("End OAM")
}

func (p *PPU) printStage(str string) {
	p.cyclesPassed = p.s.Cycle() - p.cyclesStart
	p.bus.Log.Debugf("%s: %d (line %d)", str, p.cyclesPassed, p.currentLine)
}

// startVBlank is performed on the first cycle of each line 144 to 152, and
// performs the VBlank period for the current line. The VBlank period lasts
// until for 456 * 10 cycles, when the PPU enters mode 2 (OAM search) on
// line 153 (PPU be like line 0, no line 153. you know, line 0, not the line 153 it's the next line :)).
func (p *PPU) startVBlank() {
	// should we start line 153?
	if p.currentLine == 153 {
		p.startLine153()
		return
	}

	p.lyForComparison = 255
	p.statUpdate()

	// set the LY register to current scanline
	p.CurrentScanline = p.currentLine

	if p.currentLine == 144 {
		p.modeToInterrupt = lcd.OAM

		// trigger vblank interrupt (according to mooneye's test, this is triggered on the first cycle of line 144 for DMG, but
		// is delayed by 4 cycles for CGB)
		if !p.isGBCCompat {
			p.irq.Request(interrupts.VBlankFlag)
		}

	}
	p.statUpdate()

	p.s.ScheduleEvent(scheduler.PPUContinueVBlank, 4)

	if p.currentLine == 144 {
		p.cyclesStart = p.s.Cycle()
		p.cyclesPassed = 0
	}
	p.printStage("Prepare VBlank")
}

func (p *PPU) continueVBlank() {
	p.lyForComparison = p.currentLine
	p.statUpdate()
	if p.currentLine == 144 {
		p.mode = lcd.VBlank // entering vblank

		// trigger vblank interrupt
		if p.isGBCCompat {
			p.irq.Request(interrupts.VBlankFlag)
		}

		// entering vblank also triggers the OAM STAT interrupt if enabled
		if !p.statInterruptLine && p.status&0x20 != 0 {
			p.irq.Request(interrupts.LCDFlag)
		}
		p.modeToInterrupt = lcd.VBlank
		p.statUpdate()
	}

	p.s.ScheduleEvent(scheduler.PPUStartVBlank, 452)

	p.printStage("Start VBlank")
	// start vblank for next line
	// line 153 is a special case
	p.currentLine++
}

func (p *PPU) startLine153() {
	p.CurrentScanline = 153
	p.lyForComparison = 255

	p.statUpdate()

	p.s.ScheduleEvent(scheduler.PPULine153Continue, 4)
	p.printStage("Prepare Line 153")
}

func (p *PPU) continueLine153() {
	p.CurrentScanline = 0
	p.lyForComparison = 153
	p.statUpdate()

	p.s.ScheduleEvent(scheduler.PPULine153End, 4)
	p.printStage("Start Line 153")
}

func (p *PPU) endLine153() {
	p.CurrentScanline = 0
	p.lyForComparison = 255
	p.statUpdate()

	p.s.ScheduleEvent(scheduler.PPUEndFrame, 4)
	p.printStage("End Line 153")
}

func (p *PPU) endFrame() {
	p.lyForComparison = 0
	p.statUpdate()
	p.currentLine = 0
	p.windowInternal = 0

	p.s.ScheduleEvent(scheduler.PPUStartOAMSearch, 444)
	p.printStage("End Frame")
}

// TODO save compatibility palette
// - load game with boot ROM Enabled
// - save colour palette to file (bgp = index 0 of colour palette, obp1 = index 0 of sprite palette, obp2 = index 1 of sprite palette)
// - encoded filename as hash of palette

func (p *PPU) LoadCompatibilityPalette() {
	if p.bus.BootROM != nil {
		return // don't load compatibility palette if boot ROM is Enabled (as the boot ROM will setup the palette)
	}

	hash := p.bus.Cart.Header().TitleChecksum()
	entryWord := uint16(hash) << 8
	if p.bus.Cart.Header().Title != "" {
		entryWord |= uint16(p.bus.Cart.Header().Title[3])
	}
	paletteEntry, ok := palette.GetCompatibilityPaletteEntry(entryWord)
	if !ok {
		// load default palette
		paletteEntry = palette.CompatibilityPalettes[0x1C][0x03]
	}

	// set the palette
	for i := 0; i < 4; i++ {
		p.ColourPalette.Palettes[0][i] = paletteEntry.BG[i]
		p.compatibilityPalette[i] = paletteEntry.BG[i]
		p.ColourSpritePalette.Palettes[0][i] = paletteEntry.OBJ0[i]
		p.ColourSpritePalette.Palettes[1][i] = paletteEntry.OBJ1[i]
	}

}

func (p *PPU) Read(address uint16) uint8 {
	// read from VRAM
	if address >= 0x8000 && address <= 0x9FFF {
		if !p.vramReadBlocked {
			return p.vRAM[p.vRAMBank].Read(address - 0x8000)
		} else {
			return 0xFF
		}
	}

	// read from OAM
	if address >= 0xFE00 && address <= 0xFE9F {
		if !p.oamReadBlocked && !p.DMA.IsTransferring() {
			return p.oam.Read(address - 0xFE00)
		}
		return 0xff
	}

	// illegal read
	panic(fmt.Sprintf("PPU: Read from invalid address: %X", address))
}

func (p *PPU) vramUnlocked() bool {
	return p.mode != lcd.VRAM
}

func (p *PPU) DumpTileMaps(tileMap1, tileMap2 *image.RGBA) {
	// draw tilemap (0x9800 - 0x9BFF)
	for i := uint8(0); i < 32; i++ {
		for j := uint8(0); j < 32; j++ {
			tileEntry := p.calculateTileID(j, i, 0)
			// get tile data
			tile := p.TileData[tileEntry.attributes.vRAMBank][tileEntry.GetID(p.UsingSignedTileData())]
			tile.Draw(tileMap1, int(i*8), int(j*8))
		}
	}

	// draw tilemap (0x9C00 - 0x9FFF)
	for i := uint8(0); i < 32; i++ {
		for j := uint8(0); j < 32; j++ {
			tileEntry := p.calculateTileID(j, i, 1)

			// get tile data
			tile := p.TileData[tileEntry.attributes.vRAMBank][tileEntry.GetID(p.UsingSignedTileData())]
			tile.Draw(tileMap2, int(i)*8, int(j)*8)
		}
	}
}

func (p *PPU) DumpTiledata() image.Image {
	// 3 tilesets of 128 tiles each = 384 tiles total (CGB doubles everything)
	// 1 tile = 8x8 pixels
	// 384 * 64 = 24576 pixels total
	// 256 * 96 = 24576 pixels total
	// CGB = 768 * 64 = 49152 pixels total
	// CGB = 512 * 96 = 49152 pixels total
	var img *image.RGBA
	if p.isGBCCompat {
		img = image.NewRGBA(image.Rect(0, 0, 256, 192))
	} else {
		img = image.NewRGBA(image.Rect(0, 0, 256, 96))
	}

	for i, tile := range p.TileData[0] {
		// calculate the x and y position of the tile
		x := (i % 32) * 8
		y := (i / 32) * 8

		tile.Draw(img, x, y)
	}

	if p.isGBCCompat {
		for i, tile := range p.TileData[1] {
			// calculate the x and y position of the tile
			x := (i % 32) * 8
			y := (i/32)*8 + 96

			tile.Draw(img, x, y)
		}
	}

	return img
}

func (p *PPU) colorPaletteUnlocked() bool {
	return p.mode != lcd.VRAM
}

func (p *PPU) writeVRAM(address uint16, value uint8) {

	if address <= 0x2000 {
		// write to the current VRAM bank
		p.vRAM[p.vRAMBank].Write(address, value)

		// are we writing to the tile data?
		if address <= 0x17FF {
			p.updateTile(address, value)
			// update the tile data
		} else if address <= 0x1FFF {
			if p.vRAMBank == 0 {
				// which offset are we writing to?
				if address >= 0x1800 && address <= 0x1BFF {
					// tilemap 0
					p.updateTileMap(address, 0)
				}
				if address >= 0x1C00 && address <= 0x1FFF {
					// tilemap 1
					p.updateTileMap(address, 1)
				}
			}
			if p.vRAMBank == 1 {
				// update the tile attributes
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
		return
	}

	// out of bounds
	panic(fmt.Sprintf("ppu: write to out of bounds VRAM address %04X", address))
}

func (p *PPU) Write(address uint16, value uint8) {
	// VRAM (0x8000 - 0x9FFF)
	if address >= 0x8000 && address <= 0x9FFF {
		// is the VRAM currently locked?
		if !p.vramUnlocked() {
			return
		}
		p.writeVRAM(address-0x8000, value)
		return
	}
	// OAM (0xFE00 - 0xFE9F)
	if address >= 0xFE00 && address <= 0xFE9F {
		if !p.oamWriteBlocked && !p.DMA.IsTransferring() {
			p.oam.Write(address-0xFE00, value)
		}
		return
	}

	// illegal writes
	panic(fmt.Sprintf("ppu: illegal write to address %04X", address))
}

// updateTile updates the tile at the given address
func (p *PPU) updateTile(address uint16, value uint8) {
	// get the tile address
	index := address & 0x1FFE // only the lower 13 bits are used

	// get the tileID
	tileID := index >> 4 // divide by 16

	// get the tile row
	row := (address >> 1) & 0x7

	// set the tile data
	p.TileData[p.vRAMBank][tileID][row+((address%2)*8)] = value

	p.dirtyBackground(tile)
	// recache tilemap
	//p.recacheByID(tileID)
}

func (p *PPU) updateTileMap(address uint16, tilemapIndex uint8) {
	// determine the y and x position
	y := (address / 32) & 0x1F
	x := address & 0x1F

	// update the tilemap
	p.tileMaps[tilemapIndex][y][x].id = uint16(p.vRAM[0].Read(address))

	p.dirtyBackground(tileMap)
}

func (p *PPU) updateTileAttributes(index uint16, tilemapIndex uint8, value uint8) {
	// panic(fmt.Sprintf("updating tile %x with %b", index, value))
	// determine the y and x position
	y := (index / 32) & 0x1F
	x := index & 0x1F

	// update the tilemap
	t := TileAttributes{}
	t.bgPriority = value&0x80 != 0
	t.yFlip = value&0x40 != 0
	t.xFlip = value&0x20 != 0
	t.paletteNumber = value & 0b111
	t.vRAMBank = value >> 3 & 0x1

	p.tileMaps[tilemapIndex][y][x].attributes = t
	p.tileMaps[tilemapIndex][y][x].Tile = p.TileData[t.vRAMBank][p.tileMaps[tilemapIndex][y][x].id]
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
	if p.lyForComparison == p.lyCompare {
		p.lycInterruptLine = true
		p.status |= 0x04
	} else {
		if p.lyForComparison != 255 {
			p.lycInterruptLine = false
		}
		p.status &^= 0x04
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
		p.irq.Request(interrupts.LCDFlag)
	}
}

var (
	scrollXHblank = [8]uint16{200, 196, 196, 196, 196, 192, 192, 192}
	scrollXvRAM   = [8]uint16{172, 176, 176, 176, 176, 180, 180, 180}
)

func (p *PPU) renderScanline() {
	if p.CurrentScanline >= ScreenHeight {
		return
	}
	if !p.Debug.BackgroundDisabled {
		if (!p.backgroundLineRendered[p.CurrentScanline] || p.oam.spriteScanlines[p.CurrentScanline] || p.oam.dirtyScanlines[p.CurrentScanline] || p.backgroundDirty) && (p.BackgroundEnabled || p.isGBC) {
			p.renderBackgroundScanline()
		}
	}

	if !p.Debug.WindowDisabled {
		if p.WindowEnabled {
			p.renderWindowScanline()
		}
	}

	if !p.Debug.SpritesDisabled {
		if p.SpriteEnabled {
			p.renderSpritesScanline(p.CurrentScanline)
		}
	}
}

func (p *PPU) renderBlank() {
	for y := uint8(0); y < ScreenHeight; y++ {
		for x := uint8(0); x < ScreenWidth; x++ {
			p.PreparedFrame[y][x] = p.Palette.GetColour(0) // TODO handle GBC
		}
	}
	p.Clear()
}

func (p *PPU) renderBlankLine() {
	for x := uint8(0); x < ScreenWidth; x++ {
		p.PreparedFrame[p.CurrentScanline][x] = p.Palette.GetColour(0)
	}
}

func (p *PPU) renderWindowScanline() {
	// do nothing if window is out of bounds
	if p.CurrentScanline < p.windowY {
		return
	} else if p.windowX > ScreenWidth {
		return
	} else if p.windowY > ScreenHeight {
		return
	}

	yPos := p.windowInternal

	// get the initial x pos and pixel pos
	xPos := p.windowX - 7
	xPixelPos := xPos % 8

	// get the tile map row
	tileMapRow := p.tileMaps[p.WindowTileMap][yPos>>3]

	// get the first tile entry
	tileEntry := tileMapRow[xPos>>3]
	tileID := tileEntry.GetID(p.UsingSignedTileData())

	yPixelPos := yPos

	// get the first lot of tile data
	tileData := p.TileData[tileEntry.attributes.vRAMBank][tileID]
	if tileEntry.attributes.yFlip {
		yPixelPos = 7 - yPixelPos
	}
	yPixelPos %= 8

	// get the 2 bytes that make up a row of 8 pixels
	b1 := tileData[yPixelPos]
	b2 := tileData[yPixelPos+8]

	// create the colour lookup table
	var colourLUT [8]uint8
	if tileEntry.attributes.xFlip {
		colourLUT = p.reversedColourNumberLUT[b1][b2]
	} else {
		colourLUT = p.colourNumberLUT[b1][b2]
	}

	priority := tileEntry.attributes.bgPriority

	// assign the palette to use
	var pal palette.Palette
	if p.isGBC || p.bus.BootROM != nil && !p.bus.IsBootROMDone() {
		pal = p.ColourPalette.Palettes[tileEntry.attributes.paletteNumber]
	} else if p.isGBCCompat {
		// we use the last palette for compatibility mode (not actually representative of
		// the actual hardware)
		pal = p.compatibilityPalette
	} else {
		pal = p.Palette
	}

	scanline := &p.PreparedFrame[p.CurrentScanline]
	bgPriorityLine := &p.tileBgPriority[p.CurrentScanline]

	for i := uint8(0); i < ScreenWidth; i++ {
		// did we just finish a tile?
		if xPixelPos == 8 { // we don't need to do this for the last pixel
			// increment xPos by the number of pixels we've just rendered
			xPos = i - (p.windowX - 7)

			// get the next tile entry
			tileEntry = tileMapRow[xPos>>3]
			tileID = tileEntry.GetID(p.UsingSignedTileData())

			if p.isGBC || p.bus.BootROM != nil && !p.bus.IsBootROMDone() {
				pal = p.ColourPalette.Palettes[tileEntry.attributes.paletteNumber]
			} else if p.isGBCCompat {
				// we use the last palette for compatibility mode (not actually representative of
				// the actual hardware)
				pal = p.compatibilityPalette
			} else {
				pal = p.Palette
			}
			tileData = p.TileData[tileEntry.attributes.vRAMBank][tileID]

			if tileEntry.attributes.yFlip {
				yPixelPos = 7 - yPos
			}
			yPixelPos %= 8
			b1 = tileData[yPixelPos]
			b2 = tileData[yPixelPos+8]
			if tileEntry.attributes.xFlip {
				colourLUT = p.reversedColourNumberLUT[b1][b2]
			} else {
				colourLUT = p.colourNumberLUT[b1][b2]
			}

			priority = tileEntry.attributes.bgPriority

			// reset the x pixel pos
			xPixelPos = 0
		}

		// don't render until we're in the window
		if i >= p.windowX-7 {
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
	yPos := p.CurrentScanline + p.scrollY

	// get the initial x pos and pixel pos
	xPos := p.scrollX
	xPixelPos := xPos & 7

	// get the first tile entry
	tileEntry := p.tileMaps[p.BackgroundTileMap][yPos>>3][xPos>>3]
	tileID := tileEntry.GetID(p.UsingSignedTileData())

	yPixelPos := yPos
	// get the first lot of tile data
	tileData := p.TileData[tileEntry.attributes.vRAMBank][tileID]
	if tileEntry.attributes.yFlip {
		yPixelPos = 7 - yPixelPos
	}
	yPixelPos %= 8

	// get the 2 bytes that make up a row of 8 pixels
	b1 := tileData[yPixelPos]
	b2 := tileData[yPixelPos+8]

	// create the colour lookup table
	var colourLUT [8]uint8
	if tileEntry.attributes.xFlip {
		colourLUT = p.reversedColourNumberLUT[b1][b2]
	} else {
		colourLUT = p.colourNumberLUT[b1][b2]
	}

	priority := tileEntry.attributes.bgPriority

	// assign the palette to use
	var pal palette.Palette
	if p.isGBC || p.bus.BootROM != nil && !p.bus.IsBootROMDone() {
		pal = p.ColourPalette.Palettes[tileEntry.attributes.paletteNumber]
	} else if p.isGBCCompat {
		// we use the last palette for compatibility mode (not actually representative of
		// the actual hardware)
		pal = p.compatibilityPalette
	} else {
		pal = p.Palette
	}

	bgPriorityLine := &p.tileBgPriority[p.CurrentScanline]
	scanline := &p.PreparedFrame[p.CurrentScanline]

	for i := uint8(0); i < ScreenWidth; i++ {
		// set scanline using unsafe to copy 4 bytes at a time
		//*(*uint32)(unsafe.Pointer(&scanline[i])) = *(*uint32)(unsafe.Pointer(&pal[colourLUT[xPixelPos]]))
		scanline[i] = pal[colourLUT[xPixelPos]]
		bgPriorityLine[i] = priority
		p.colorNumber[i] = colourLUT[xPixelPos]

		xPixelPos++

		// did we just finish a tile?
		if xPixelPos == 8 { // we don't need to do this for the last pixel
			// increment xPos by the number of pixels we've just rendered
			xPos += 8

			// get the next tile entry
			tileEntry = p.tileMaps[p.BackgroundTileMap][yPos>>3][xPos>>3]
			tileID = tileEntry.GetID(p.UsingSignedTileData())

			if p.isGBC || p.bus.BootROM != nil && !p.bus.IsBootROMDone() {
				pal = p.ColourPalette.Palettes[tileEntry.attributes.paletteNumber]
			} else if p.isGBCCompat {
				// we use the last palette for compatibility mode (not actually representative of
				// the actual hardware)
				pal = p.compatibilityPalette
			} else {
				pal = p.Palette
			}
			tileData = p.TileData[tileEntry.attributes.vRAMBank][tileID]

			if tileEntry.attributes.yFlip {
				yPixelPos = 7 - yPos
			}
			yPixelPos %= 8
			b1 = tileData[yPixelPos]
			b2 = tileData[yPixelPos+8]
			if tileEntry.attributes.xFlip {
				colourLUT = p.reversedColourNumberLUT[b1][b2]
			} else {
				colourLUT = p.colourNumberLUT[b1][b2]
			}

			priority = tileEntry.attributes.bgPriority

			// reset the x pixel pos
			xPixelPos = 0
		}
	}

	p.backgroundLineRendered[p.CurrentScanline] = true
	p.oam.dirtyScanlines[p.CurrentScanline] = false
}

// calculateTileID calculates the tile ID for the current scanline
func (p *PPU) calculateTileID(tilemapOffset, lineOffset uint8, mapOffset uint8) TileMapEntry {
	// get the tile entry from the tilemap
	tileEntry := p.tileMaps[mapOffset][tilemapOffset][lineOffset]

	return tileEntry
}

func (p *PPU) renderSpritesScanline(scanline uint8) {
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
		if p.isGBC {
			pal = p.ColourSpritePalette.Palettes[sprite.cgbPalette]
		} else {
			pal = p.SpritePalettes[sprite.useSecondPalette]
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
			if !p.isGBC || p.BackgroundEnabled {
				if !(sprite.priority && !p.tileBgPriority[scanline][pixelPos]) {
					if p.colorNumber[pixelPos] != 0 {
						continue
					}
				}
			}

			if p.isGBC {
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

			// has the sprite changed the background?
			if p.PreparedFrame[scanline][pixelPos] != pal[color] {
				p.backgroundLineRendered[scanline] = false
				// draw the pixel
				p.PreparedFrame[scanline][pixelPos] = pal[color]
			}

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
	p.Controller.Load(s)
	p.CurrentScanline = s.Read8()
	p.lyCompare = s.Read8()
	p.windowX = s.Read8()
	p.windowY = s.Read8()
	p.windowInternal = s.Read8()
	for i := uint16(0); i < 0x2000; i++ {
		p.vRAMBank = 0
		p.writeVRAM(i, s.Read8())
	}
	for i := uint16(0); i < 0x2000; i++ {
		p.vRAMBank = 1
		p.writeVRAM(i, s.Read8())
	}
	// load the vRAM data
	p.vRAMBank = s.Read8()
	p.DMA.Load(s)
	p.RefreshScreen = s.ReadBool()
	p.statInterruptDelay = s.ReadBool()
	p.delayedTick = s.ReadBool()
	p.Palette = palette.LoadPaletteFromState(s)
	p.SpritePalettes[0] = palette.LoadPaletteFromState(s)
	p.SpritePalettes[1] = palette.LoadPaletteFromState(s)
	p.ColourPalette.Load(s)
	p.ColourSpritePalette.Load(s)
}

func (p *PPU) Save(s *types.State) {
	p.Controller.Save(s)        // 1 byte
	s.Write8(p.CurrentScanline) // 1 byte
	s.Write8(p.lyCompare)       // 1 byte
	s.Write8(p.windowX)         // 1 byte
	s.Write8(p.windowY)         // 1 byte
	s.Write8(p.windowInternal)  // 1 byte
	p.vRAM[0].Save(s)           // 8192 bytes
	p.vRAM[1].Save(s)           // 8192 bytes
	s.Write8(p.vRAMBank)        // 1 byte
	p.DMA.Save(s)
	s.WriteBool(p.RefreshScreen)
	s.WriteBool(p.statInterruptDelay)
	s.WriteBool(p.delayedTick)
	p.Palette.Save(s)
	p.SpritePalettes[0].Save(s)
	p.SpritePalettes[1].Save(s)
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

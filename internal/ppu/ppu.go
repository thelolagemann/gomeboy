//go:generate go run golang.org/x/tools/cmd/stringer -type=GlitchedLineState,LineState,OffscreenLineState,FetcherState,ObjectFetcherState -output=ppu_string.go
package ppu

import (
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/utils"
	"math/bits"
	"sort"
)

const (
	// ScreenWidth is the width of the screen in pixels.
	ScreenWidth = 160
	// ScreenHeight is the height of the screen in pixels.
	ScreenHeight = 144
)

const (
	// ModeHBlank (Mode 0) - Horizontal Blanking Period
	//
	// 	Duration 87 - 204 dots (variable per line)
	//	- Allows CPU access to VRAM/OAM
	// 	- STAT interrupt available if enabled via STAT.3
	ModeHBlank = iota

	// ModeVBlank (Mode 1) - Vertical Blanking Period
	//
	//	Duration 4560 dots (10 lines)
	//	- Allows full CPU access to VRAM/OAM
	//	- VBlank interrupt available if enabled via IF.0
	//	- STAT interrupt available if enabled via STAT.4 or STAT.5 (bug?)
	//	- Active during LY 144-153
	ModeVBlank

	// ModeOAM (Mode 2) - OAM Scan
	//
	//	Duration: 80 dots (fixed)
	//	- Locks OAM bus
	//	- STAT interrupt available if enabled via STAT.5 (only during mode transition)
	//	- PPU searches OAM for visible sprites
	//	- Occurs at start of each line
	ModeOAM

	// ModeVRAM (Mode 3) - Pixel Transfer
	//
	//	Duration: 172-289 dots (variable depending on objects and window)
	//	- Locks both OAM and VRAM buses
	//	- No STAT interrupts available
	//	- Active during visible pixel rendering
	//	- Pixel FIFOs are continually clocked until LX=168
	ModeVRAM
)

// PPU implements the Game Boy's (P)ixel (P)rocessing (U)nit.
//
// References:
//   - [Pan Docs](https://gbdev.io/pandocs/Graphics.html)
//   - [Hacktix GBEDG](https://hacktix.github.io/GBEDG/ppu/)
//   - [Mooneye test suite](https://github.com/Gekkio/mooneye-test-suite)
type PPU struct {
	// LCDC register
	enabled     bool  // LCDC.7 - LCD Enable
	bgEnabled   bool  // LCDC.0 - BG Display
	winEnabled  bool  // LCDC.5 - Window Enable
	objEnabled  bool  // LCDC.1 - OBJ Enable
	bgTileMap   uint8 // LCDC.3 - BG Tile Map Select (0=9800-9BFF, 1=9C00-9FFF)
	winTileMap  uint8 // LCDC.6 - Window Tile Map Select
	objSize     uint8 // LCDC.2 - OBJ Size (0=8x8, 1=8x16)
	addressMode uint8 // LCDC.4 - BG/Win Tile Data Select (0=8800-97FF, 1=8000-8FFF)

	// Rendering state
	mode      uint8 // Mode reported to STAT register
	modeToInt uint8 // Mode used to raise STAT interrupt
	ly        uint8 // Current line (0-153)
	lx        uint8 // Current dot within line (0-167)
	status    uint8 // Local copy of STAT register

	// Window rendering state
	winFetcherX  uint8 // Window-specific LX position counter
	wly          uint8 // Window line counter
	winTriggerWy bool  // Window Y-position trigger
	winTriggerWx bool  // Window X-position trigger

	// Scroll discard state
	scxDiscarded uint8 // Number of pixels discarded for SCX alignment
	scxToDiscard uint8 // Target number of pixels to discard based on SCX % 8

	// Scroll registers
	scy, scx uint8 // Background viewport position
	wy, wx   uint8 // Window Position

	// LY comparison state
	lyCompare       uint8  // LYC register value
	lyForComparison uint16 // Effective LY value used for LYC checks

	// Interrupt lines
	lycInt  bool // Current LYC interrupt line
	statInt bool // Current STAT interrupt line

	// Palette configuration
	cRAM                     [128]uint8    // CGB palette RAM (64 BG + 64 OBJ entries)
	ColourPalette            ColourPalette // Active BG Palette data
	ColourOBJPalette         ColourPalette // Active OBJ Palette data
	BGColourisationPalette   Palette       // BG Palette used for DMG->CGB colourisation
	OBJ0ColourisationPalette Palette       // OBJ0 Palette used for DMG->CGB colourisation
	OBJ1ColourisationPalette Palette       // OBJ1 Palette used for DMG->CGB colourisation

	// External components
	b *io.Bus
	s *scheduler.Scheduler

	// Frame buffers
	PreparedFrame [ScreenHeight][ScreenWidth][3]uint8

	// Pixel slice fetcher
	bgFIFO               *utils.FIFO[FIFOEntry] // Background/Window pixel FIFO
	objFIFO              *utils.FIFO[FIFOEntry] // OBJ pixel fifo
	fetcherState         FetcherState           // Current pixel slice fetcher phase
	fetcherTileNo        uint8                  // Current tile index from tile map
	fetcherTileAttr      uint8                  // Current attributes
	fetcherData          [2]uint8               // Tile pattern data (low + high bytes)
	fetcherTileNoAddress uint16                 // VRAM address of current tile map entry

	// Object fetcher
	objectFetcherState ObjectFetcherState // Current object fetcher phase
	objFetcherTileNo   uint8              // Current OBJ tile index from tile map
	objFetcherTileAttr uint8              // Current attributes for OBJ
	objFetcherData     [2]uint8           // OBJ tile pattern data (low + high bytes)
	fetcherObj         bool               // True when fetching an OBJ instead of BG/Window
	fetchingObj        Object             // Active OBJ being fetched

	// Internal rendering state
	lineState          LineState          // Visible line progression
	offscreenLineState OffscreenLineState // VBlank handling
	glitchedLineState  GlitchedLineState  // First-line startup behaviour
	objBuffer          []Object           // Scanline object buffer

	// Timing counters
	lineDot  uint64 // Cycle at which the current line began
	frameDot uint64 // Cycle at which the current frame began

	// CGB-specific features
	cgbMode       bool  // CGB mode
	bcpsIndex     uint8 // BG palette index
	ocpsIndex     uint8 // OBJ palette index
	bcpsIncrement bool  // BG palette index auto-increment
	ocpsIncrement bool  // OBJ palette index auto-increment

	// Debug controls
	Debug struct {
		OBJDisabled        bool // Force disable OBJ rendering
		BackgroundDisabled bool // Force disable BG layer
		WindowDisabled     bool // Force disable window layer
	}
}

// A FIFOEntry represents a single pixel entry in a FIFO.
type FIFOEntry struct {
	Color      uint8
	Attributes uint8
	OBJIndex   uint8
	Palette    uint8
}

// Object is used to define the attributes of an object in OAM.
type Object struct {
	x, y  uint8
	attr  uint8
	id    uint8
	index uint8
}

// New creates and initializes a PPU instance ready to be used.
func New(b *io.Bus, s *scheduler.Scheduler) *PPU {
	p := &PPU{
		b: b,
		s: s,

		bgFIFO:  utils.NewFIFO[FIFOEntry](8),
		objFIFO: utils.NewFIFO[FIFOEntry](8),
	}

	for pal := 0; pal < 8; pal++ {
		for c := 0; c < 4; c++ {
			p.ColourPalette[pal][c] = [3]uint8{0xff, 0xff, 0xff}
			p.ColourOBJPalette[pal][c] = [3]uint8{0xff, 0xff, 0xff}
		}
	}

	b.ReserveAddress(types.LCDC, func(v byte) byte {
		// is the screen turning off?
		if p.enabled && v&types.Bit7 == 0 {
			p.enabled = false

			// the screen should only be turned off in VBlank
			if p.mode != ModeVBlank {
				// warn user of incorrect behaviour TODO
			}

			// deschedule all PPU events
			for i := scheduler.PPUHandleVisualLine; i <= scheduler.PPUHandleOffscreenLine; i++ {
				p.s.DescheduleEvent(i)
			}

			// when the LCD is off, LY reads 0, and STAT mode reads 0 (HBlank)
			p.b.Set(types.LY, 0)
			p.ly, p.mode = 0, 0
			p.lyForComparison = 0
			p.wly = 255

			// OAM & VRAM bus is released
			p.b.Unlock(io.OAM | io.VRAM)
			p.resetFetcher()
		} else if !p.enabled && v&types.Bit7 != 0 {
			p.enabled = true
			p.cgbMode = b.IsGBCCart() && b.IsGBC()

			p.frameDot = p.s.Cycle()

			p.glitchedLineState = StartGlitchedLine
			p.statUpdate()

			if p.cgbMode {
				p.handleGlitchedLine0()
			} else {
				p.s.ScheduleEvent(scheduler.PPUHandleGlitchedLine0, 1)
			}
		}

		p.winTileMap = v >> 6 & 1
		p.winEnabled = v&types.Bit5 > 0
		p.addressMode = 1 &^ (v >> 4 & 1)
		p.bgTileMap = v >> 3 & 1
		p.objSize = 8 + (v & types.Bit2 << 1)
		p.objEnabled = v&types.Bit1 > 0
		p.bgEnabled = v&types.Bit0 > 0

		return v
	})
	b.ReserveAddress(types.STAT, func(v byte) byte {
		//p.b.Debugf("writing stat %08b\n", v)
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
		return types.Bit7 | p.status | p.mode
	})
	b.ReserveAddress(types.SCY, func(v byte) byte {
		p.scy = v
		return v
	})
	b.ReserveAddress(types.SCX, func(v byte) byte {
		p.scx = v
		return v
	})
	b.ReserveAddress(types.LY, func(v byte) byte {
		// any write to LY resets to 0
		p.ly = 0
		return 0
	})
	b.ReserveAddress(types.LYC, func(v byte) byte {
		p.lyCompare = v
		p.statUpdate()
		return v
	})
	b.ReserveAddress(types.BGP, func(v byte) byte {
		if p.b.IsGBCCart() && p.b.IsGBC() {
			return v
		}
		p.ColourPalette[0] = p.BGColourisationPalette.remap(v)
		return v
	})
	b.ReserveAddress(types.OBP0, func(v byte) byte {
		if v == b.Get(types.OBP0) || p.b.IsGBCCart() && p.b.IsGBC() {
			return v
		}
		p.ColourOBJPalette[0] = p.OBJ0ColourisationPalette.remap(v)

		return v
	})
	b.ReserveAddress(types.OBP1, func(v byte) byte {
		if v == b.Get(types.OBP1) || p.b.IsGBCCart() && p.b.IsGBC() {
			return v
		}
		p.ColourOBJPalette[1] = p.OBJ1ColourisationPalette.remap(v)

		return v
	})
	b.ReserveAddress(types.WY, func(v byte) byte {
		p.wy = v
		return v
	})
	b.ReserveAddress(types.WX, func(v byte) byte {
		p.wx = v
		return v
	})

	b.RegisterBootHandler(func() {
		if b.IsGBC() && !b.IsGBCCart() {
			p.BGColourisationPalette = p.ColourPalette[0]
			p.OBJ0ColourisationPalette = p.ColourOBJPalette[0]
			p.OBJ1ColourisationPalette = p.ColourOBJPalette[1]
		}
	})

	// setup CGB only registers
	b.RegisterGBCHandler(func() {
		b.ReserveAddress(types.BCPS, func(v byte) byte {
			p.bcpsIndex = v & 0x3F
			p.bcpsIncrement = v&types.Bit7 > 0
			return v | types.Bit6
		})
		b.ReserveAddress(types.BCPD, func(v byte) byte {
			if p.b.IsGBCCart() || p.b.IsBooting() {
				if p.mode != ModeVRAM {
					p.cRAM[p.bcpsIndex] = v
					palIndex, colorIndex := p.bcpsIndex>>3&7, p.bcpsIndex&7>>1

					colour := uint16(p.ColourPalette[palIndex][colorIndex][0]>>3) |
						uint16(p.ColourPalette[palIndex][colorIndex][1]>>3)<<5 |
						uint16(p.ColourPalette[palIndex][colorIndex][2]>>3)<<10

					if p.bcpsIndex&1 == 1 {
						colour = (colour & 0x00FF) | uint16(v)<<8
					} else {
						colour = (colour & 0xFF00) | uint16(v)
					}

					p.ColourPalette[palIndex][colorIndex][0] = uint8(colour&0x1f)<<3 | uint8(colour&0x1f)>>2
					p.ColourPalette[palIndex][colorIndex][1] = uint8(colour>>5&0x1f)<<3 | uint8(colour>>5&0x1f)>>2
					p.ColourPalette[palIndex][colorIndex][2] = uint8(colour>>10&0x1f)<<3 | uint8(colour>>10&0x1f)>>2
				}
				if p.bcpsIncrement {
					p.bcpsIndex = (p.bcpsIndex + 1) & 0x3f
					p.b.Set(types.BCPS, p.b.Get(types.BCPS)&0xC0|p.bcpsIndex)
				}

				return v
			}

			return 0xff
		})
		b.ReserveLazyReader(types.BCPD, func() byte {
			if p.mode != ModeVRAM && p.b.IsGBCCart() {
				return p.cRAM[p.bcpsIndex]
			}
			return 0xff
		})
		b.ReserveAddress(types.OCPS, func(v byte) byte {
			p.ocpsIndex = v & 0x3F
			p.ocpsIncrement = v&types.Bit7 > 0
			return v | types.Bit6
		})
		b.ReserveAddress(types.OCPD, func(v byte) byte {
			if p.b.IsGBCCart() || p.b.IsBooting() {
				if p.mode != ModeVRAM {
					p.cRAM[64+p.ocpsIndex] = v
					palIndex, colorIndex := p.ocpsIndex>>3&7, p.ocpsIndex&7>>1

					colour := uint16(p.ColourOBJPalette[palIndex][colorIndex][0]>>3) |
						uint16(p.ColourOBJPalette[palIndex][colorIndex][1]>>3)<<5 |
						uint16(p.ColourOBJPalette[palIndex][colorIndex][2]>>3)<<10

					if p.ocpsIndex&1 == 1 {
						colour = (colour & 0x00FF) | uint16(v)<<8
					} else {
						colour = (colour & 0xFF00) | uint16(v)
					}

					p.ColourOBJPalette[palIndex][colorIndex][0] = uint8(colour&0x1f)<<3 | uint8(colour&0x1f)>>2
					p.ColourOBJPalette[palIndex][colorIndex][1] = uint8(colour>>5&0x1f)<<3 | uint8(colour>>5&0x1f)>>2
					p.ColourOBJPalette[palIndex][colorIndex][2] = uint8(colour>>10&0x1f)<<3 | uint8(colour>>10&0x1f)>>2
				}

				if p.ocpsIncrement {
					p.ocpsIndex = (p.ocpsIndex + 1) & 0x3f
					p.b.Set(types.OCPS, p.b.Get(types.OCPS)&0xC0|p.ocpsIndex)
				}

				return v
			}

			return 0xff
		})
		b.ReserveLazyReader(types.OCPD, func() byte {
			if p.mode != ModeVRAM && p.b.IsGBCCart() {
				return p.cRAM[64+p.ocpsIndex]
			}
			return 0xff
		})
	})

	s.RegisterEvent(scheduler.PPUHandleVisualLine, p.handleVisualLine)
	s.RegisterEvent(scheduler.PPUHandleGlitchedLine0, p.handleGlitchedLine0)
	s.RegisterEvent(scheduler.PPUHandleOffscreenLine, p.handleOffscreenLine)
	return p
}

// GlitchedLineState represents the progression states for handling the peculiar first line
// that occurs immediately after enabling the LCD. This glitched line has different timings and
// behaviour compared to normal lines, requiring special state management to accurately emulate
// the hardware behaviour.
//
// The states model the sequence of events that occur during this glitched line period,
// including OAM/VRAM locking, mode transitions and the shortened line length.
type GlitchedLineState int

const (
	// StartGlitchedLine is the initial state when beginning the glitched first line.
	// Occurs after a 1-dot delay from the PPU being enabled on DMG, and instantly on CGB.
	StartGlitchedLine GlitchedLineState = iota

	// GlitchedLineOAMWBlock represents the PPU acquiring a write lock on the OAM bus shortly
	// before the end of the ModeOAM scanning phase.
	GlitchedLineOAMWBlock

	// GlitchedLineEndOAM marks the end of the glitched ModeOAM scanning phase and transition
	// to ModeVRAM.
	GlitchedLineEndOAM

	// GlitchedLineStartPixelTransfer initiates the beginning of the ModeVRAM pixel transfer phase.
	// The actual pixel transfer processing is skipped, since this first frame will never be displayed.
	GlitchedLineStartPixelTransfer
)

// glitchedLineCycles defines the duration (in dots) for each state in the glitched line
// sequence.
//
//   - StartGlitchedLine:				76 dots (completes at 77)
//   - GlitchedLineOAMWBlock:	 		 2 dots (completes at 79)
//   - GlitchedLineEndOAM:		 		 5 dots (completes at 84)
//   - GlitchedLineStartPixelTransfer:	 0 dots (-> enterPixelTransfer)
var glitchedLineCycles = []uint64{
	StartGlitchedLine:     76,
	GlitchedLineOAMWBlock: 2,
	GlitchedLineEndOAM:    5,
}

// handleGlitchedLine0 handles the very first line after turning the LCD on. Behaviour is
// a bit peculiar, with the line itself being cut short by 4 dots, the initial mode presenting
// as ModeHBlank instead of ModeOAM, and the PPU itself failing to acquire the OAM bus.
//
// As a consequence of the PPU failing to acquire a lock on the OAM bus the subsequent OAM
// scanning will not find any objects. This detail is irrelevant to our implementation as
// the LCD will never receive the first frame anyway, but it's still an interesting detail.
//
//   - [mooneye/acceptance/ppu/lcdon_timing]
//   - [mooneye/acceptance/ppu/lcdon_write_timing]
func (p *PPU) handleGlitchedLine0() {
	switch p.glitchedLineState {
	case StartGlitchedLine:
		// DMG models take a dot to turn the PPU on, but CGB models
		// appear to turn on instantly, therefore we update lineDot
		// here rather than in writes to LCDC. TODO verify how&why
		p.lineDot = p.s.Cycle()
	case GlitchedLineOAMWBlock:
		p.b.WLock(io.OAM)
	case GlitchedLineEndOAM:
		p.mode, p.modeToInt = ModeVRAM, ModeVRAM
		p.b.Lock(io.OAM)
		p.b.Block(io.VRAM, p.s.DoubleSpeed() || !p.cgbMode)
	case GlitchedLineStartPixelTransfer:
		p.b.Lock(io.VRAM)

		// we can just skip the expensive pixel transfer as
		// the first frame will never be displayed anyway
		p.lineState = EnterHBlank
		p.lineDot -= 4
		p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 168)
		return
	}

	p.s.ScheduleEvent(scheduler.PPUHandleGlitchedLine0, glitchedLineCycles[p.glitchedLineState])
	p.glitchedLineState++
}

// LineState represents the progression states of a standard visible line (LY=0-143) on
// the PPU. It attempts to model the ModeOAM, ModeHBlank and ModeVRAM phases with relatively
// accurate timing.
//
// The state machine coordinates memory bus locking, pixel pipeline operation, and synchronization
// with other PPU components. States transition through:
//
//	ModeOAM -> ModeVRAM -> ModeHBlank -> LY++ ; Repeat
//
// Timing values are derived from hardware measurements where possible and validated against
// test ROMs.
type LineState int

const (
	// StartOAMScan initiates ModeOAM search phase. Locks OAM bus access, initializes
	// LY comparison for LYC checks, checks for the WY==LY condition and handles ModeOAM
	// STAT interrupts.
	StartOAMScan LineState = iota

	// ReleaseOAMBus handles the OAM bus being briefly released on dot 76 of ModeOAM search.
	ReleaseOAMBus

	// StartPixelTransfer enter ModeVRAM pixel transfer phase. Initializes FIFOs and
	// manages the initial pipeline priming for visible pixel generation.
	StartPixelTransfer

	// PixelTransferDummy handles initial pipeline setup. Prepares pixel transfer phase
	// for either direct pixel output or SCX alignment.
	PixelTransferDummy

	// PixelTransferSCXDiscard processes horizontal scroll (SCX) offset adjustment by
	// discarding initial pixels until proper alignment is reached.
	PixelTransferSCXDiscard

	// PixelTransferLX handles actual visible pixel generation (LX = 8-167). Blends
	// BG and OBJ pixels as necessary.
	PixelTransferLX

	// EnterHBlank marks transition to ModeHBlank. Unlocks memory buses, handles HDMA
	// transfers (CGB), and schedules end of ModeHBlank
	EnterHBlank

	// HBlankUpdateLY increments internal LY register and checks for ModeVBlank transition.
	// Manages STAT mode transitions and ModeOAM preparation for the next line.
	HBlankUpdateLY

	// HBlankUpdateOAM configures OAM bus access for next line.
	HBlankUpdateOAM

	// HBlankUpdateVisibleLY finally updates the memory-mapped LY register (visible to the CPU)
	HBlankUpdateVisibleLY

	// HBlankEnd completes the ModeHBlank period and prepares to handle the next line's
	// ModeOAM search phase.
	HBlankEnd
)

// stateCycles defines the duration (in dots) for each visible line state phase.
//   - StartOAMScan:          		76 dots (completes at 76)
//   - ReleaseOAMBus:          		 4 dots (completes at 80)
//   - StartPixelTransfer:     		 5 dots (completes at 85)
//   - PixelTransferDummy:     		 0 dots (completes at 85)
//   - PixelTransferSCXDiscard:	 1 - 7 dots (variable)
//   - PixelTransferLX:      172 - 289 dots (variable)
//   - EnterHBlank:  452 - (80 + mode3Dots) (variable)
//   - HBlankUpdateLY:				 1 dots (completes at 452)
//   - HBlankUpdateOAM:				 2 dots (completes at 454)
//   - HBlankUpdateVisibleLY:		 1 dots (completes at 455)
//   - HBlankEnd:					 0 dots (-> enterOAMScan)
var stateCycles = []uint64{
	StartOAMScan:            76,
	ReleaseOAMBus:           4,
	StartPixelTransfer:      5,
	PixelTransferDummy:      1,
	PixelTransferSCXDiscard: 1,
	PixelTransferLX:         1,
	// EnterHBlank is variable and is manually scheduled
	HBlankUpdateLY:        1,
	HBlankUpdateOAM:       2,
	HBlankUpdateVisibleLY: 1,
	HBlankEnd:             0,
}

// handleVisualLine manages the complete lifecycle of the visible lines (LY = 0-143)
// progressing through ModeOAM, ModeVRAM and ModeHBlank phases with the best effort attempt
// at hardware accurate timings.
//
//   - [mooneye/acceptance/ppu/intr_2_0_timing]
//   - [mooneye/acceptance/ppu/intr_2_mode_0_timing]
//   - [mooneye/acceptance/ppu/intr_2_mode3_timing]
//   - [mooneye/acceptance/ppu/intr_2_oam_ok_timing]
func (p *PPU) handleVisualLine() {
	switch p.lineState {
	case StartOAMScan:
		p.b.Lock(io.OAM)
		p.checkWindowTriggerWY()

		p.lineDot = p.s.Cycle()
		p.lyForComparison = uint16(p.ly)
		p.mode, p.modeToInt = ModeOAM, ModeOAM
		p.statUpdate()
		p.modeToInt = 255
		p.statUpdate()
	case ReleaseOAMBus: // on dot 76
		p.b.RBlock(io.VRAM, !p.cgbMode)
		p.b.WBlock(io.OAM, p.cgbMode)
		p.b.WUnlock(io.VRAM)
	case StartPixelTransfer: // on dot 80
		// fill obj
		if p.objEnabled && !p.Debug.OBJDisabled {
			// fill obj buffer
			p.objBuffer = []Object{}
			for i := uint16(0); i < 0xa0 && len(p.objBuffer) < 10; i += 4 {
				y, x, id, attr := p.b.Get(0xfe00+i), p.b.Get(0xfe00+i+1), p.b.Get(0xfe00+i+2), p.b.Get(0xfe00+i+3)

				if p.ly+16 >= y &&
					p.ly+16 < y+p.objSize {
					spr := Object{
						y:     y,
						x:     x,
						id:    id,
						attr:  attr,
						index: uint8(i) >> 2,
					}

					if p.objSize == 16 {
						if (p.ly+16-y < 8 && spr.attr&types.Bit6 > 0) || (p.ly+16-y >= 8 && spr.attr&types.Bit6 == 0) {
							spr.id |= 1
						} else {
							spr.id &= 0xfe
						}
					}

					p.objBuffer = append(p.objBuffer, spr)
				}
			}

			sort.SliceStable(p.objBuffer, func(i, j int) bool {
				return p.objBuffer[i].x < p.objBuffer[j].x
			})
		}

		p.resetFetcher()
		p.mode, p.modeToInt = ModeVRAM, ModeVRAM
		p.statUpdate() // clear stat line
		p.b.Lock(io.OAM | io.VRAM)
	case PixelTransferDummy: // on dot 85
		p.bgFIFO.Size = 8

		// the first tile fetch is used to handle SCX % 8 > 0, however this data
		// will just be discarded, so we "fill" the BG FIFO with junk data
		p.scxToDiscard = p.scx & 7

		if p.scxToDiscard > 0 {
			p.scxDiscarded = 0
			// window can be activated before BG when SCX % 8 > 0
			p.checkWindowTriggerWX()

			p.lineState = PixelTransferSCXDiscard
		} else {
			p.lineState = PixelTransferLX
		}

		if !p.s.DoubleSpeed() {
			p.handleVisualLine()
		} else {
			p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)
		}
		return
	case PixelTransferSCXDiscard:
		p.stepPixelFetcher()
		if p.bgFIFO.Size > 0 {
			p.bgFIFO.Pop()
			p.scxDiscarded++
			if p.scxDiscarded == p.scxToDiscard {
				// can we finally start transferring pixels yet?
				p.lineState = PixelTransferLX

				p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)
				return
			}

		}

		p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)
		return // remain in this state until we have discarded all SCX % 8 pixels
	case PixelTransferLX:
		p.checkWindowTriggerWX()

		// is the PPU currently fetching an object?
		if p.fetcherObj {
			p.stepObjectFetcher()

			// if still fetching object just wait another dot
			if p.fetcherObj {
				p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)

				// object fetching halts pixel output to the LCD
				return
			}
		}

		// are there any pending objects on this X coordinate?
		if len(p.objBuffer) > 0 && (p.objEnabled || p.cgbMode) && p.objBuffer[0].x == p.lx {
			// do we need to finish the current BG/Window fetch?
			if p.fetcherState < BGWinGetTileDataHighT2 || p.bgFIFO.Size == 0 {
				p.stepPixelFetcher()
			} else {
				p.stepPixelFetcher()
				p.fetcherObj = true
				p.objectFetcherState = OBJGetTileNoT1
				p.fetchingObj = p.objBuffer[0]
				p.objBuffer = p.objBuffer[1:]
			}

			p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)

			return // object fetching immediately halts pushing pixels to the lcd
		}

		p.pushPixel()
		p.stepPixelFetcher()

		// have we reached the end of the line yet?
		if p.lx != 168 {
			p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)
			return // remain in the PixelTransferLX state until we reach LX==168
		}

		if !p.s.DoubleSpeed() {
			p.mode, p.modeToInt = ModeHBlank, ModeHBlank
		}

		p.lineState = EnterHBlank
		p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)
		return
	case EnterHBlank: // variable
		p.mode, p.modeToInt = ModeHBlank, ModeHBlank
		p.statUpdate()
		p.b.Unlock(io.OAM | io.VRAM)

		dotsPassed := p.s.Cycle() - p.lineDot

		if p.s.DoubleSpeed() {
			dotsPassed >>= 1
		}

		// TODO verify timing
		if p.cgbMode {
			p.b.HandleHDMA()
		}

		p.lineState = HBlankUpdateLY
		p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 452-dotsPassed)
		return
	case HBlankUpdateLY: // dot 452
		if p.ly != 143 {
			p.modeToInt = ModeOAM
		}
		p.ly++

		// are we about to enter vblank?
		if p.ly == 144 {
			// the LCD never receives the first frame after turning the PPU on,
			// or rather the LCD doesn't starting clocking any data in until the
			// first vblank has been reached
			// [little-things-gb/first-white]
			if p.s.Cycle()-p.frameDot < 65664 { // shorter frame is probably the glitched line so we're turning on
				p.renderBlank()
			}

			// move to vblank
			p.offscreenLineState = StartVBlank
			p.handleOffscreenLine()
			return
		}
		// we are just continuing OAM -> VRAM -> HBlank loop
		p.lineState = HBlankUpdateOAM
		fallthrough
	case HBlankUpdateOAM: // dot 452 these 4 dots overlap with the beginning of OAM scan (to handle STAT delay)
		p.b.WBlock(io.OAM, p.cgbMode && !p.s.DoubleSpeed())
	case HBlankUpdateVisibleLY: // dot 454
		p.b.Set(types.LY, p.ly)
	case HBlankEnd: // dot 455
		p.b.RBlock(io.OAM, !p.s.DoubleSpeed())
		if p.ly != 0 {
			p.lyForComparison = 0xffff
		} else {
			p.lyForComparison = 0
		}

		// OAM stat int fires 1 dot before STAT changes, except on line 0
		if p.ly != 0 {
			p.modeToInt = ModeOAM
			p.mode = ModeHBlank
		} else if !p.cgbMode {
			p.mode = ModeHBlank
		}
		p.statUpdate()

		p.lineState = StartOAMScan
		p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)
		return
	}

	p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, stateCycles[p.lineState])
	p.lineState++
}

// OffscreenLineState represents the progression states during vertical blanking
// period (ModeVBlank, LY = 144-153) for the PPU.
//
// The state machine models both standard ModeVBlank lines (144-152) and the peculiar
// line 153 behaviour.
type OffscreenLineState int

const (
	// StartVBlank initiates the vertical blanking period. Disables LY comparison
	// and updates the STAT mode to ModeVBlank.
	StartVBlank OffscreenLineState = iota

	// VBlankUpdateLY writes current LY value to memory-mapped register.
	// Handles CGB-specific OAM stat interrupt timing differences.
	VBlankUpdateLY

	// VBlankUpdateLYC updates LY comparison status and triggers
	// LY=LYC interrupt if enabled.
	VBlankUpdateLYC

	// VBlankHandleInt manages VBlank interrupt at LY=144 as well as handling
	// LY increments and the beginning of line 153.
	VBlankHandleInt

	// StartVBlankLastLine prepares for line 153 handling with special LY
	// comparison reset.
	StartVBlankLastLine

	// Line153LYUpdate updates the LY register to 153
	Line153LYUpdate

	// Line153LY0 resets the LY register to 0 but maintains the LY to compare
	// against being 153 still
	Line153LY0

	// Line153LYC disables the LY comparison and processes STAT updates.
	Line153LYC

	// Line153LYC0 finalizes the LY 153 -> 0 transition during line 153.
	Line153LYC0

	// EndFrame completes frame processing and resets PPU state for
	// new frame.
	EndFrame
)

// offscreenLineCycles defines timing for VBlank period states.
//   - StartVBlank:         2 dots (completes at 2)
//   - VBlankUpdateLY:      2 dots (completes at 4)
//   - VBlankUpdateLYC:     1 dot  (completes at 5)
//   - VBlankHandleInt:   451 dots (completes at 456)
//   - StartVBlankLastLine: 2 dots (completes at 2)
//   - Line153LYUpdate:		4 dots (completes at 6)
//   - Line153LY0: 			2 dots (completes at 8)
//   - Line153LYC:			4 dots (completes at 12)
//   - Line153LYC0:		  444 dots (completes at 456)
//   - EndFrame:			0 dot  (-> enterNewFrame)
//
// These timings maintain the 456 dots/line cadence during VBlank.
var offscreenLineCycles = []uint64{
	StartVBlank:         2,
	VBlankUpdateLY:      2,
	VBlankUpdateLYC:     1,
	VBlankHandleInt:     451,
	StartVBlankLastLine: 2,
	Line153LYUpdate:     4,
	Line153LY0:          2,
	Line153LYC:          4,
	Line153LYC0:         444,
}

// handleOffscreenLine manages the ModeVBlank period (LY = 144 - 153) maintaining
// the 456 dots/line cadence. This also includes handling the erratic behaviour that
// occurs on line 153 in regard to the LY/LYC register and STAT comparison checks.
//
//   - [mooneye/acceptance/ppu/intr_1_2_timing-GS]
//   - [mooneye/acceptance/ppu/vblank_stat_intr-GS]
//   - [mooneye/misc/ppu/vblank_stat_intr-C]
func (p *PPU) handleOffscreenLine() {
	switch p.offscreenLineState {
	case StartVBlank: // dot 0
		p.lineDot = p.s.Cycle()
		p.lyForComparison = 0xffff
		p.statUpdate()
	case VBlankUpdateLY: // dot 2
		p.b.Set(types.LY, p.ly)
		// CGB models appear to trigger the OAM stat int during
		// vblank 2 dots earlier than DMG models?
		// [mooneye/misc/ppu/vblank_stat_intr-C]
		if p.b.Model() >= types.CGB0 && p.ly == 144 && !p.statInt && p.status&0x20 > 0 {
			p.b.RaiseInterrupt(io.LCDINT)
		}
	case VBlankUpdateLYC: // dot 4
		p.lyForComparison = uint16(p.ly)
		p.statUpdate()
	case VBlankHandleInt: // dot 5
		switch p.ly {
		case 144: // entering vblank
			p.mode, p.modeToInt = ModeVBlank, ModeVBlank
			p.b.RaiseInterrupt(io.VBlankINT)

			// entering vblank also triggers the OAM interrupt
			// [mooneye/acceptance/ppu/vblank_stat_intr-GS.gb]
			if p.b.Model() < types.CGB0 && !p.statInt && p.status&0x20 > 0 {
				p.b.RaiseInterrupt(io.LCDINT)
			}
			p.statUpdate()
		case 152: // leaving vblank
			p.ly++
			p.offscreenLineState = StartVBlankLastLine
			p.s.ScheduleEvent(scheduler.PPUHandleOffscreenLine, 451)
			return
		}

		p.ly++
		p.offscreenLineState = StartVBlank
		p.s.ScheduleEvent(scheduler.PPUHandleOffscreenLine, 451)
		return
	case StartVBlankLastLine: // dot 0
		p.lineDot = p.s.Cycle()
		p.lyForComparison = 0xffff
		p.statUpdate()
	case Line153LYUpdate:
		p.b.Set(types.LY, 153)
	case Line153LY0:
		p.b.Set(types.LY, 0)
		p.lyForComparison = 153
		p.statUpdate()
	case Line153LYC:
		p.lyForComparison = 0xffff
		p.statUpdate()
	case Line153LYC0:
		p.lyForComparison = 0
		p.statUpdate()
	case EndFrame:
		p.ly = 0
		p.winTriggerWy = false
		p.wly = 255
		p.frameDot = p.s.Cycle()
		p.lineState = HBlankUpdateOAM
		p.handleVisualLine()
		return
	}

	p.s.ScheduleEvent(scheduler.PPUHandleOffscreenLine, offscreenLineCycles[p.offscreenLineState])
	p.offscreenLineState++
}

// pushPixel attempts to push a pixel out from one of the two FIFOs and into
// the LCD. A few things can prevent the LCD from receiving any data in. The
// BG FIFO must have at least one pixel in it, the PPU cannot be fetching an
// object, and there can't be any pending objects at the current LX position.
// If any of these conditions aren't met then the LCD will not receive any data
// and the LX position will remain the same.
//
// Once all three of these conditions are met, then the BG FIFO will be popped,
// alongside the Object FIFO if it has any pending pixels, and merged according
// to the current mode of operation (DMG vs CGB). If LX < 8 then these pixels
// are simply discarded (but still popped) and the LCD will not receive them.
// Once LX >= 8 then the LCD starts clocking in the data from the FIFOs and LX
// increases as normal.
func (p *PPU) pushPixel() {
	if !p.canPopBG() {
		return
	}

	var color [3]uint8

	var bgPX, objPX *FIFOEntry
	var bgPriority, drawObject bool
	var bgEnabled = true

	bgPX = p.bgFIFO.Pop()
	bgPriority = bgPX.Attributes&types.Bit7 > 0

	// are there any pending object pixels?
	if p.objFIFO.Size > 0 {
		objPX = p.objFIFO.Pop()

		if objPX.Color > 0 && (p.objEnabled || p.cgbMode) {
			color = p.ColourOBJPalette[objPX.Palette][objPX.Color]

			drawObject = true
			if objPX.Attributes&types.Bit7 > 0 {
				bgPriority = true
			}
		}
	}

	// are we currently offscreen? (pixels are simply discarded)
	if p.lx < 8 {
		p.lx++
		return
	}

	// BG_EN bit is different on CGB
	if !p.bgEnabled {
		if p.cgbMode {
			bgPriority = false
		} else {
			bgEnabled = false
		}
	}

	if !drawObject || bgPriority && bgPX.Color > 0 {
		// are we drawing a BG pixel?
		if bgEnabled {
			if (p.winTriggerWx && !p.Debug.WindowDisabled) || (!p.winTriggerWx && !p.Debug.BackgroundDisabled) {
				color = p.ColourPalette[bgPX.Attributes&7][bgPX.Color]
			} else {
				// user has disabled rendering through debugging functions
				color = [3]uint8{0xff, 0xff, 0xff} // white TODO customizable disabled color
			}
		} else {
			color = p.ColourPalette[bgPX.Attributes&7][0]
		}
	}

	// draw pixel to frame
	p.PreparedFrame[p.ly][p.lx-8] = color
	p.lx++
}

// FetcherState represents the internal state machines of the PPU's pixel slice fetcher
// responsible for retrieving tile data from VRAM and converting it into pixel streams.
// It handles rendering the background and window layers, and can be interrupted by
// the object fetching process.
//
// The fetcher progresses through distinct phases for each layer:
//  1. Tile number lookup
//  2. Attribute retrieval (CGB only)
//  3. Tile data fetching (low/high bytes)
//  4. Pushing pixels to FIFO
type FetcherState int

// Background/Window activation states
const (
	BGWinActivating        FetcherState = iota // Initial state selecting BG/Window layer
	BGGetTileNoT1                              // BG: Start tile number lookup (dot 1)
	BGGetTileNoT2                              // BG: Complete tile number lookup (dot 2)
	BGGetTileDataLowT1                         // BG: Start tile data low fetch (dot 1)
	BGGetTileDataLowT2                         // BG: Complete tile data low fetch (dot 2)
	BGGetTileDataHighT1                        // BG: Start tile data high fetch (dot 1)
	BGWinGetTileDataHighT2                     // BG/Window: Complete tile data high fetch and finalize tile data (dot 2)
	BGWinPushPixels                            // Push decoded pixels to BG FIFO
)

// Window-specific states
const (
	WinActivating        = BGWinPushPixels + 1 + iota // Window layer activating
	WinGetTileNoT1                                    // Window: Start tile number lookup (dot 1)
	WinGetTileNoT2                                    // Window: Complete tile number lookup (dot 2)
	WinGetTileDataLowT1                               // Window: Start tile data low fetch (dot 1)
	WinGetTileDataLowT2                               // Window: Complete tile data low fetch (dot 2)
	WinGetTileDataHighT1                              // Window: Start tile data high fetch (dot 1)
)

// stepPixelFetcher advances the pixel slice fetcher forward by 1 step depending
// on the state it's currently in.
func (p *PPU) stepPixelFetcher() {
	// if the pixel slice fetcher is currently fetching a window, it could be aborted
	// due to WIN_EN being reset in LCDC. When this happens the fetcher simply switches
	// to BG fetching rather than restarting the process
	if p.fetcherState >= WinGetTileNoT1 && p.fetcherState <= WinGetTileDataHighT1 {
		if !p.winEnabled {
			p.fetcherState -= 8 // roll back to background fetching and continue on
		}
	}

	switch p.fetcherState {
	case BGWinActivating:
		// are we fetching for the window or BG?
		if p.winTriggerWx {
			if p.winEnabled {

				// move to WinGetTileNoT1
				p.fetcherState = WinGetTileNoT1
				p.stepPixelFetcher()
				return
			}

			// window fetch was aborted, proceed with bg fetch
			p.winTriggerWx = false
		}

		// we're fetching background
		p.fetcherState = BGGetTileNoT1
		p.stepPixelFetcher()
		return
	// Background
	case BGGetTileNoT1:
		address := uint16(0x1800)
		address |= uint16(p.bgTileMap) << 10

		// we should have already shifted out the necessary pixels for SCX
		// adjustment during the PixelTransferLXSCXDiscard stage, therefore
		// we just need to apply the current SCX to the current LX position
		// [age/m3-bg-scx/m3-bg-scx]:CGB
		// [age/m3-bg-scx/m3-bg-scx-ds]:CGB
		// [age/m3-bg-scx/m3-bg-scx-nocgb]:CGB
		// TODO determine timings that alter the CGB expected result
		x := ((int(p.scx) + int(p.lx)) >> 3) & 0x1f

		// determine x,y position on tilemap
		address |= (uint16((p.ly+p.scy)>>3) & 0x1f) << 5 // Y pos
		address |= uint16(x)                             // X pos
		p.fetcherTileNoAddress = address

		// TODO verify when CGB accesses tile attrs
		if p.cgbMode {
			p.fetcherTileAttr = p.b.GetVRAM(p.fetcherTileNoAddress, 1)
		}
	case BGGetTileDataLowT2:
		p.fetcherData[0] = p.b.GetVRAM(p.getBGTileAddress(), p.fetcherTileAttr&types.Bit3>>3)
	case BGWinGetTileDataHighT2:
		p.fetcherData[1] = p.b.GetVRAM(p.getBGTileAddress()|1, p.fetcherTileAttr&types.Bit3>>3)
		if p.fetcherTileAttr&types.Bit5 > 0 {
			p.fetcherData[0] = bits.Reverse8(p.fetcherData[0])
			p.fetcherData[1] = bits.Reverse8(p.fetcherData[1])
		}
		fallthrough
	// TODO handle immediate window push?
	case BGWinPushPixels:
		p.fetcherState = BGWinPushPixels
		// pixels can only be pushed to the BG FIFO when it is empty, otherwise
		// we need to remain in this state until it is
		isEmpty := p.bgFIFO.Size == 0

		if isEmpty {
			// TODO handle LX == WX glitch here?

			attr := p.fetcherTileAttr
			low, high := p.fetcherData[0], p.fetcherData[1]
			p.bgFIFO.Data = [8]FIFOEntry{
				{Attributes: attr, Color: (low>>7)&1 | (high>>7)&1<<1},
				{Attributes: attr, Color: (low>>6)&1 | (high>>6)&1<<1},
				{Attributes: attr, Color: (low>>5)&1 | (high>>5)&1<<1},
				{Attributes: attr, Color: (low>>4)&1 | (high>>4)&1<<1},
				{Attributes: attr, Color: (low>>3)&1 | (high>>3)&1<<1},
				{Attributes: attr, Color: (low>>2)&1 | (high>>2)&1<<1},
				{Attributes: attr, Color: (low>>1)&1 | (high>>1)&1<<1},
				{Attributes: attr, Color: (low)&1 | (high)&1<<1},
			}
			p.bgFIFO.Size = 8

			p.fetcherState = BGWinActivating
		}

		return
	// Window
	case WinActivating:
		p.fetcherState = BGWinActivating
		return
	case WinGetTileNoT1:
		address := uint16(0x1800)

		address |= uint16(p.winTileMap) << 10   // Tilemap
		address |= uint16((p.wly>>3)&0x1f) << 5 // Y pos
		address |= uint16(p.winFetcherX)        // X pos

		// window has its own internal lx counter
		p.winFetcherX++

		p.fetcherTileNoAddress = address

		// TODO verify when CGB accesses tile attrs
		if p.cgbMode {
			p.fetcherTileAttr = p.b.GetVRAM(p.fetcherTileNoAddress, 1)
		}
	case WinGetTileDataLowT1:
		p.fetcherData[0] = p.b.GetVRAM(p.getWinTileAddress(), p.fetcherTileAttr&types.Bit3>>3)
	case WinGetTileDataHighT1:
		p.fetcherData[1] = p.b.GetVRAM(p.getWinTileAddress()|1, p.fetcherTileAttr&types.Bit3>>3)
		if p.fetcherTileAttr&types.Bit5 > 0 {
			p.fetcherData[0] = bits.Reverse8(p.fetcherData[0])
			p.fetcherData[1] = bits.Reverse8(p.fetcherData[1])
		}

		// the push stage is the same as the BG so we move to that state
		p.fetcherState = BGWinPushPixels
		return
	}

	p.fetcherState++
}

// ObjectFetcherState represents the internal state machine of the PPU's object
// fetching pipeline. It handles rendering the object layer and has a few key
// distinctions from the background/window pixel slice fetcher.
//
// One of the main differences is that the object fetcher will simply read the tile
// ID from the object buffer, rather than reading it from the tile map. Alongside the
// ID, the object fetcher also grabs the object attributes from the object buffer,
// whereas only the CGB has tile attributes available to the BG/Window fetcher.
//
// Whilst the BG/Window pixel slice fetcher will wait until the BG FIFO is empty
// in order to start pushing new pixels, the Object fetcher will simply merge
// existing pixels with incoming. The rules for which pixels are kept during this
// merging stage are explained further at the relevant steps. As there are only 6
// steps to the object fetching process, the object fetch will always take 6 dots
// to complete.
//
// An object fetch may interrupt a BG/Window fetch as it's halfway through fetching.
// When this occurs, the object fetcher waits until the BG/Window fetching process is
// complete and then starts the object fetch. So even though object fetching will always
// take 6 dots, the duration of it may appear to have taken longer as it's waiting upon
// a BG/Window fetch to finish.
type ObjectFetcherState int

// Object (sprite) rendering states
const (
	OBJGetTileNoT1       ObjectFetcherState = iota // OBJ: Start obj tile number lookup (dot 1)
	OBJGetTileNoT2                                 // OBJ: Complete obj tile number lookup (or get attr?) (dot 2)
	OBJGetTileDataLowT1                            // OBJ: Start tile data low fetch (dot 1)
	OBJGetTileDataLowT2                            // OBJ: Complete tile data low fetch (dot 2)
	OBJGetTileDataHighT1                           // OBJ: Start tile data high fetch (dot 1)
	OBJGetTileDataHighT2                           // OBJ: Complete tile data high fetch (dot 2)
)

// stepObjectFetcher advances the object fetcher forward by 1 step depending on the
// state it's currently in.
func (p *PPU) stepObjectFetcher() {
	switch p.objectFetcherState {
	case OBJGetTileNoT1:
		p.objFetcherTileNo = p.fetchingObj.id
		p.stepPixelFetcher()
	case OBJGetTileNoT2:
		p.objFetcherTileAttr = p.fetchingObj.attr // TODO verify timings
		p.stepPixelFetcher()
	case OBJGetTileDataLowT2:
		p.objFetcherData[0] = p.b.GetVRAM(p.getObjectTileAddress(), p.objFetcherTileAttr&types.Bit3>>3)
	case OBJGetTileDataHighT2:
		p.objFetcherData[1] = p.b.GetVRAM(p.getObjectTileAddress()|1, p.objFetcherTileAttr&types.Bit3>>3)
		if p.objFetcherTileAttr&types.Bit5 > 0 {
			p.objFetcherData[0] = bits.Reverse8(p.objFetcherData[0])
			p.objFetcherData[1] = bits.Reverse8(p.objFetcherData[1])
		}
		j := uint8(0)
		for i := uint8(0x80); i > 0; i >>= 1 {
			if p.fetchingObj.x+j < 8 {
				j++
				continue // offscreen
			}

			objFIFO := FIFOEntry{
				Attributes: p.fetchingObj.attr,
				Palette:    p.fetchingObj.attr & types.Bit4 >> 4,
				OBJIndex:   p.fetchingObj.index,
			}
			if p.cgbMode {
				objFIFO.Palette = p.fetchingObj.attr & 7
			}
			objFIFO.Color = ((p.objFetcherData[0] & i) >> (7 - j)) | ((p.objFetcherData[1]&i)>>(7-j))<<1

			// on DMG objects with the lowest x wins priority, as we already stable sort by x at the end
			// of the oam phase then we only need to check to see if the existing pixel has a colour ID of
			// 0 (=transparent)
			// on CGB (in CGB mode) objects inserted first into the OAM wins priority, so we have to compare
			// the existing pixels insertion index with the new one
			// in all cases an object pixel with a colour id of 0 will be overwritten
			if int(j) < p.objFIFO.Size {
				if p.objFIFO.GetIndex(int(j)).Color == 0 ||
					p.cgbMode &&
						objFIFO.Color != 0 && p.objFIFO.GetIndex(int(j)).OBJIndex > p.fetchingObj.index {
					p.objFIFO.ReplaceIndex(int(j), objFIFO)
				}
			} else {
				p.objFIFO.Push(objFIFO)
			}

			j++

		}

		// are there any more pending OBJ at this LX?
		if p.canFetchOBJ() {
			p.fetchingObj = p.objBuffer[0]
			p.objBuffer = p.objBuffer[1:]
			p.objectFetcherState = OBJGetTileNoT1
		} else {
			p.fetcherObj = false
		}
		return
	}

	p.objectFetcherState++
}

// getBGTileAddress determines the address to read from VRAM for the tile
// that is currently loaded in the pixel fetcher.
func (p *PPU) getBGTileAddress() uint16 {
	// read tile number from VRAM
	tileNo := p.b.GetVRAM(p.fetcherTileNoAddress, 0)

	yPos := (p.ly + p.scy) & 7
	if p.fetcherTileAttr&types.Bit6 > 0 {
		yPos = ^yPos & 7
	}

	// determine where the tile is in VRAM
	address := uint16(0x0000)
	address |= uint16(tileNo) << 4                      // Tile ID offset
	address |= uint16(p.addressMode&^(tileNo>>7)) << 12 // Negation of ID.7 when LCDC.4 is set
	address |= uint16(yPos) << 1                        // Y pos

	return address
}

// getObjectTileAddress determines the address to read from VRAM for the OBJ tile
// that is currently loaded in the pixel fetcher.
func (p *PPU) getObjectTileAddress() uint16 {
	// handle obj size
	tileY := p.fetchingObj.y - 16
	tileY = (p.ly - tileY) & (p.objSize - 1)

	// handle y pos
	if p.objFetcherTileAttr&types.Bit6 > 0 {
		tileY = ^tileY & (p.objSize - 1)
	}

	// determine where the tile is in VRAM
	address := uint16(0x0000)
	address |= uint16(p.objFetcherTileNo) << 4 // Tile ID offset
	address |= uint16(tileY) << 1              // Y pos

	return address
}

// getWinTileAddress determines the address to read from VRAM for the tile
// that is currently loaded in the pixel fetcher.
func (p *PPU) getWinTileAddress() uint16 {
	// read tile number from vram
	tileNo := p.b.GetVRAM(p.fetcherTileNoAddress, 0)

	yPos := p.wly & 7
	if p.fetcherTileAttr&types.Bit6 > 0 {
		yPos = ^yPos & 7
	}

	// determine where the tile is in VRAM
	address := uint16(0x0000)
	address |= uint16(tileNo) << 4                      // Tile ID offset
	address |= uint16(p.addressMode&^(tileNo>>7)) << 12 // Negation of ID.7 when LCDC.4 is set
	address |= uint16(yPos) << 1                        // Y pos

	return address
}

// statUpdate handles updating the STAT interrupt. As the conditions for
// raising a STAT interrupt are checked every dot, we need to call this
// whenever one of the dependent conditions changes as it would be
// too expensive to use the scheduler.
//
// The OAM STAT interrupt is an exception to this rule, it is only raised
// as a consequence of the HBlank -> OAM transition, and the bugged VBlank
// trigger. Thus writing to STAT's OAM interrupt flag whilst already in
// ModeOAM (or ModeVBlank) does not raise a request.
func (p *PPU) statUpdate() {
	if !p.enabled {
		// STAT & LYC call this but the PPU may be disabled when doing so
		// in which case the STAT line isn't processed
		return
	}

	// update LYC_EQ_LY flag
	if p.lyForComparison != 0xffff || p.b.Model() <= types.CGBABC && !p.s.DoubleSpeed() {
		if uint8(p.lyForComparison) == p.lyCompare {
			p.lycInt = true
			p.status |= types.Bit2
		} else {
			if p.lyForComparison != 0xffff {
				p.lycInt = false
			}
			p.status &^= types.Bit2
		}
	}

	// handle the STAT mode interrupt
	statINT := (p.modeToInt == ModeHBlank && p.status&types.Bit3 != 0) ||
		(p.modeToInt == ModeVBlank && p.status&types.Bit4 != 0) ||
		(p.modeToInt == ModeOAM && p.status&types.Bit5 != 0) ||
		(p.lycInt && p.status&types.Bit6 != 0)

	// did STAT go low -> high
	if !p.statInt && statINT {
		p.b.RaiseInterrupt(io.LCDINT)
	}

	p.statInt = statINT
}

// canFetchOBJ determines whether an OBJ should be fetched.
// There are only two conditions that must be met in order to
// initiate an OBJ fetch.
//   - Pending OBJ at the current LX
//   - OBJ enabled in LCDC
func (p *PPU) canFetchOBJ() bool {
	return len(p.objBuffer) > 0 && p.objBuffer[0].x == p.lx && p.objEnabled
}

// canPopBG determines whether a pixel can be popped from the
// BG FIFO. There are three conditions that must be met in order
// to be able to pop a pixel.
//   - BG FIFO isn't empty
//   - Fetcher isn't fetching an OBJ
//   - There are no more pending OBJs for this LX or OBJ are disabled
func (p *PPU) canPopBG() bool {
	return p.bgFIFO.Size > 0 &&
		!p.fetcherObj &&
		!(p.objEnabled && len(p.objBuffer) > 0 && p.objBuffer[0].x == p.lx)
}

// checkWindowTriggerWY checks to see if the window should be triggered
// for the current LY position. The window is considered to be active for
// the rest of the frame, if during at some point WIN_EN was true whilst
// LY == WY. This doesn't mean the window will always be rendered, as the
// LX == WX condition is checked again during the pixel transfer stage.
func (p *PPU) checkWindowTriggerWY() {
	if p.winEnabled && p.wy == p.ly {
		p.winTriggerWy = true
	}
}

// checkWindowTriggerWX checks to see if the window should be triggered
// for the current LX position.
func (p *PPU) checkWindowTriggerWX() {
	if p.winTriggerWy && !p.winTriggerWx && p.winEnabled {
		// TODO handle WX=0
		if p.wx == 0 {
		} else if p.wx < 166 {
			if p.wx == p.lx-1 { // TODO figure out why the -1?
				// activate window
				p.winTriggerWx = true

				// increase internal window line counter
				p.wly++

				// reset window tile counter
				p.winFetcherX = 0

				p.bgFIFO.Reset()

				// next dot is window activating
				p.fetcherState = WinActivating
			}
		}

	}
}

// renderBlank blanks the current screen.
func (p *PPU) renderBlank() {
	for y := uint8(0); y < ScreenHeight; y++ {
		for x := uint8(0); x < ScreenWidth; x++ {
			p.PreparedFrame[y][x] = p.ColourPalette[0][0]
		}
	}
}

// resetFetcher resets the pixel slice fetcher and the two pixel
// FIFOs so that they can be used for the next line.
func (p *PPU) resetFetcher() {
	p.bgFIFO.Reset()
	p.objFIFO.Reset()

	p.winFetcherX = 0
	p.winTriggerWx = false

	p.fetcherState = BGWinActivating
	p.lx = 0
}

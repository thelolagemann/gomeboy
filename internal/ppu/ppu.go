//go:generate go run golang.org/x/tools/cmd/stringer -type=LineState,OffscreenLineState,FetcherState -output=ppu_string.go
package ppu

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/utils"
	"math/bits"
	"sort"
	"strconv"
	"strings"
)

const (
	// ScreenWidth is the width of the screen in pixels.
	ScreenWidth = 160
	// ScreenHeight is the height of the screen in pixels.
	ScreenHeight = 144
)

const (
	ModeHBlank = iota
	ModeVBlank
	ModeOAM
	ModeVRAM
)

// Sprite is used to define the attributes of a sprite in OAM.
type Sprite struct {
	x, y  uint8
	attr  uint8
	id    uint8
	index uint8
}

type Palette [4][3]uint8 // 4 RGB values

func (p Palette) remap(v uint8) Palette { return Palette{p[v&3], p[v>>2&3], p[v>>4&3], p[v>>6&3]} } // remaps a palette

const (
	Greyscale = iota // default
	Green            // mimics the original green screen
)

var ColourPalettes = []Palette{
	{{0xFF, 0xFF, 0xFF}, {0xAA, 0xAA, 0xAA}, {0x55, 0x55, 0x55}, {0x00, 0x00, 0x00}}, // Greyscale
	{{0x9B, 0xBC, 0x0F}, {0x8B, 0xAC, 0x0F}, {0x30, 0x62, 0x30}, {0x0F, 0x38, 0x0F}}, // Green
}

type ColourPalette [8]Palette

//go:embed palettes.csv
var colourisationPaletteData string

type ColourisationPalette struct {
	BG, OBJ0, OBJ1 Palette
}

var ColourisationPalettes = map[uint16]ColourisationPalette{}

func init() {
	r := csv.NewReader(strings.NewReader(colourisationPaletteData))
	r.TrimLeadingSpace = true
	records, _ := r.ReadAll()

	toRGB := func(s string) [3]uint8 {
		rgb, _ := strconv.ParseUint(s, 16, 24)
		return [3]uint8{uint8(rgb >> 16), uint8(rgb >> 8), uint8(rgb)}
	}

	for _, row := range records {
		pal := ColourisationPalette{}

		for x := 0; x < 4; x++ {
			pal.BG[x] = toRGB(row[2+x])
			pal.OBJ0[x] = toRGB(row[6+x])
			pal.OBJ1[x] = toRGB(row[10+x])
		}

		hash, _ := strconv.ParseUint(row[0], 16, 8)
		var disambiguation uint64 = 0
		if len(row[1]) > 0 {
			disambiguation, _ = strconv.ParseUint(row[1], 16, 8)
		}

		ColourisationPalettes[uint16(hash)|uint16(disambiguation)<<8] = pal
	}
}

type PPU struct {
	// LCDC register
	enabled, cleared                            bool
	bgEnabled, winEnabled, objEnabled           bool
	bgTileMap, winTileMap, objSize, addressMode uint8

	// various internal flags
	mode, ly, lx, wly, status  uint8
	scxDiscarded, scxToDiscard uint8

	lyCompare       uint8
	scy, scx        uint8
	wy, wx          uint8
	bgp, pendingBGP uint8

	cgbMode                      bool
	cRAM                         [128]uint8 // 64 bytes BG+OBJ
	bcpsIncrement, ocpsIncrement bool
	bcpsIndex, ocpsIndex         uint8

	// external components
	b                 *io.Bus
	lineDot, frameDot uint64
	s                 *scheduler.Scheduler

	// palettes used for DMG -> CGB colourisation
	BGColourisationPalette   Palette
	OBJ0ColourisationPalette Palette
	OBJ1ColourisationPalette Palette

	// palettes used by the ppu to display colours
	ColourPalette       ColourPalette
	ColourSpritePalette ColourPalette

	PreparedFrame [ScreenHeight][ScreenWidth][3]uint8

	// debug
	Debug struct {
		SpritesDisabled, BackgroundDisabled, WindowDisabled bool
	}

	bgFIFO, objFIFO *utils.FIFO[FIFOEntry]
	bgFetcherX      uint8
	winFetcherX     uint8 // window has its own internal LX for fetching

	fetcherTileNo, fetcherTileAttr uint8
	fetcherData, fetcherDataCache  [2]uint8
	objFetcherLow, objFetcherHigh  uint8
	fetcherObj                     bool
	fetchingObj                    Sprite
	fetcherWin                     bool
	winTriggerWy, winTriggerWx     bool

	objBuffer []Sprite

	objFetcherTileNo, objFetcherAttr uint8

	fetcherTileNoAddress, fetcherAddress        uint16
	objFetcherLowAddress, objFetcherHighAddress uint16
	lineState                                   LineState
	offscreenLineState                          OffscreenLineState
	fetcherState                                FetcherState

	// various INT lines
	LYC_EQ_LY_INT     bool
	STAT_INT          bool
	firstLine         bool
	fetcherDataCached bool
}

// A FIFOEntry represents a single pixel in a FIFO.
type FIFOEntry struct {
	Color      uint8
	Palette    uint8
	Attributes uint8
	OBJIndex   uint8
}

func New(b *io.Bus, s *scheduler.Scheduler) *PPU {
	p := &PPU{
		b: b,
		s: s,

		bgFIFO:        utils.NewFIFO[FIFOEntry](8),
		objFIFO:       utils.NewFIFO[FIFOEntry](8),
		LYC_EQ_LY_INT: true,
	}

	for pal := 0; pal < 8; pal++ {
		for c := 0; c < 4; c++ {
			p.ColourPalette[pal][c] = [3]uint8{0xff, 0xff, 0xff}
			p.ColourSpritePalette[pal][c] = [3]uint8{0xff, 0xff, 0xff}
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
			p.wly = 255
			p.b.Debugf("turning off ppu\n")

			// OAM & VRAM bus is released
			p.b.Unlock(io.OAM | io.VRAM)
			p.lineState = StartOAMScanAfterTurnOn
			p.resetFetcher()
		} else if !p.enabled && v&types.Bit7 != 0 {
			p.enabled = true
			p.cleared = false
			p.cgbMode = b.IsGBCCart() && b.IsGBC()

			// STAT's LYC_EQ_LY is updated here (but interrupt isn't raised)
			if p.ly == p.lyCompare && p.LYC_EQ_LY_INT {
				p.status |= types.Bit2
			} else {
				p.status &^= types.Bit2
			}
			p.b.Debugf("turning on ppu STAT: %08b\n", p.status)

			p.lineDot = p.s.Cycle()
			p.frameDot = p.s.Cycle()

			p.lineState = StartOAMScanAfterTurnOn

			p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)
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
		p.ColourPalette[0] = p.BGColourisationPalette.remap(p.bgp | v)
		return v
	})
	b.ReserveAddress(types.OBP0, func(v byte) byte {
		if v == b.Get(types.OBP0) || p.b.IsGBCCart() && p.b.IsGBC() {
			return v
		}
		p.ColourSpritePalette[0] = p.OBJ0ColourisationPalette.remap(v)

		return v
	})
	b.ReserveAddress(types.OBP1, func(v byte) byte {
		if v == b.Get(types.OBP1) || p.b.IsGBCCart() && p.b.IsGBC() {
			return v
		}
		p.ColourSpritePalette[1] = p.OBJ1ColourisationPalette.remap(v)

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
			p.OBJ0ColourisationPalette = p.ColourSpritePalette[0]
			p.OBJ1ColourisationPalette = p.ColourSpritePalette[1]
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

					colour := uint16(p.ColourSpritePalette[palIndex][colorIndex][0]>>3) |
						uint16(p.ColourSpritePalette[palIndex][colorIndex][1]>>3)<<5 |
						uint16(p.ColourSpritePalette[palIndex][colorIndex][2]>>3)<<10

					if p.ocpsIndex&1 == 1 {
						colour = (colour & 0x00FF) | uint16(v)<<8
					} else {
						colour = (colour & 0xFF00) | uint16(v)
					}

					p.ColourSpritePalette[palIndex][colorIndex][0] = uint8(colour&0x1f)<<3 | uint8(colour&0x1f)>>2
					p.ColourSpritePalette[palIndex][colorIndex][1] = uint8(colour>>5&0x1f)<<3 | uint8(colour>>5&0x1f)>>2
					p.ColourSpritePalette[palIndex][colorIndex][2] = uint8(colour>>10&0x1f)<<3 | uint8(colour>>10&0x1f)>>2
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
	s.RegisterEvent(scheduler.PPUHandleOffscreenLine, p.handleOffscreenLine)
	return p
}

type LineState int

const (
	StartOAMScanAfterTurnOn LineState = iota
	StartOAMScan
	ReleaseOAMBus
	AcquireOAMBus
	StartPixelTransfer
	PixelTransferDummy
	PixelTransferSCXDiscard
	PixelTransferLX0
	PixelTransferLX8
	EnterHBlank
	HBlankUpdateOAM
	HBlankUpdateLY
	HBlankUpdateLYCInt
	HBlankEnd
	HBlankLastLine
	HBlankLastLineLYCInt
	HBlankLastLineEnd
)

var stateCycles = []uint64{
	StartOAMScanAfterTurnOn: 79,
	StartOAMScan:            76,
	ReleaseOAMBus:           2, // 76 - 78 dots (+2)
	AcquireOAMBus:           2, // 78 - 80 dots (+2)
	StartPixelTransfer:      3, // 80 - 83 dots (+3)
	PixelTransferDummy:      1, // 83 - 84 dots (+2)
	PixelTransferSCXDiscard: 1, // variable
	PixelTransferLX0:        1, // 83 - 91 dots (+8)
	PixelTransferLX8:        1,
	// EnterHBlank is variable and is manually scheduled
	HBlankUpdateOAM:      1,
	HBlankUpdateLY:       1,
	HBlankUpdateLYCInt:   1,
	HBlankEnd:            1,
	HBlankLastLine:       1,
	HBlankLastLineLYCInt: 1,
	// HBlankLastLineEnd:  enters new mode
}

// handleVisualLine handles lines 0 -> 143, stepping from OAM -> VRAM -> HBlank until line 143 is reached.
func (p *PPU) handleVisualLine() {
	if p.lineState != PixelTransferLX8 {
		p.b.Debugf("%-30s : LY: %03d LX: %03d SCX: %03d dot:%03d\n", p.lineState, p.ly, p.lx, p.scx, p.s.Cycle()-p.lineDot)
	}
	switch p.lineState {
	case StartOAMScanAfterTurnOn:
		p.statUpdate()
		p.lineState = StartPixelTransfer
		p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 79)
		return
	case StartOAMScan:
		p.lineDot = p.s.Cycle()
	case ReleaseOAMBus: // on dot 76
		p.b.Unlock(io.OAM)
	case AcquireOAMBus: // on dot 78
		p.b.Lock(io.OAM | io.VRAM)
	case StartPixelTransfer: // on dot 79?

		p.enterPixelTransfer()
		return
	case PixelTransferDummy: // on dot 83?
		// the first tile fetch is used to handle SCX % 8 > 0, however this data
		// will just be discarded, so we "fill" the BG FIFO with junk data
		p.bgFIFO.Size = 8

		p.scxToDiscard = p.scx & 7

		if p.scxToDiscard > 0 {
			p.scxDiscarded = 0
			// window can be activated before BG when SCX % 8 > 0
			//p.checkWindowTrigger()

			p.lineState = PixelTransferSCXDiscard
			p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)

			return
		}
		p.lineState = PixelTransferLX0

		p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)
		return
	// TODO handle SCX discard
	case PixelTransferSCXDiscard:
		if p.canPopBG() {
			p.bgFIFO.Pop()
			p.scxDiscarded++
			if p.scxDiscarded == p.scxToDiscard {
				// can we finally start transferring pixels yet?
				p.lineState = PixelTransferLX0

				p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)
				return
			}

		}
		p.stepPixelFetcher()
		p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)
		return // remain in this state until we have discarded all SCX % 8 pixels
	case PixelTransferLX0: // on dot 83
		incLX := false

		// the first 8 pixels are discarded, so just pop them
		if p.canPopBG() {
			p.bgFIFO.Pop()

			// not sure if this ever happens, but we should still discard them
			if p.objFIFO.Size > 0 {
				p.objFIFO.Pop()
			}

			incLX = true

			// remain in the PixelTransferLX0 state until we have reached LX==8
			if p.lx+1 == 8 {
				p.lineState = PixelTransferLX8
			}

			p.checkWindowTrigger()
		}

		// step the fetcher
		p.stepPixelFetcher()

		// did we inc LX?
		if incLX {
			p.lx++
		}

		// the window is considered to be active for the rest of the frame
		// if at some point during it happens that window is enabled while
		// LY == WY. this doesn't mean that the window will always be rendered,
		// as WX == LX condition is checked again during pixel transfer
		if p.winEnabled && p.ly == p.wy {
			p.winTriggerWy = true
		}

		// remain in the PixelTransferLX0 state until we reach LX==8
		p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, stateCycles[p.lineState])
		return
	case PixelTransferLX8:
		incLX := false

		if p.canPopBG() {
			var color [3]uint8
			var drawOAM, bgPriority bool
			var bgEnabled = true
			bgPixel := p.bgFIFO.Pop()
			bgPriority = bgPixel.Attributes&types.Bit7 > 0

			// TODO handle LCDC delay

			// are there pending OBJ pixels?
			if p.objFIFO.Size > 0 {
				objPixel := p.objFIFO.Pop()

				// in DMG mode three conditions must be met to push an OBJ pixel instead of a BG pixel
				// - OBJ are enabled
				// - OBJ pixel is opaque (color = [1,2,3])
				// - BG_OVER_OBJ is disabled | BG color == 0
				if p.objEnabled && objPixel.Color > 0 {
					color = p.ColourSpritePalette[objPixel.Palette][objPixel.Color]
					drawOAM = true
					if objPixel.Attributes&types.Bit7 > 0 {
						bgPriority = true
					}
				}
			}

			// BG_EN bit is different on CGB
			if !p.bgEnabled {
				if p.cgbMode {
					bgPriority = false
				} else {
					bgEnabled = false
				}
			}

			// did an OBJ pixel get drawn?
			if !drawOAM || bgPriority && bgPixel.Color > 0 {
				if bgEnabled {
					color = p.ColourPalette[bgPixel.Attributes&7][bgPixel.Color]
				} else {
					color = p.ColourPalette[bgPixel.Attributes&7][0]
				}
			}

			if p.ly >= 144 {
				panic(fmt.Sprintf("ROM: %s exceeded line boundaries", p.b.Cartridge().String()))
			}
			// draw pixel to frame
			p.PreparedFrame[p.ly][p.lx-8] = color

			incLX = true

			// have we reached the end of the screen?
			if p.lx+1 == 168 {
				p.lx++

				p.lineState = EnterHBlank
				p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)
				return
			}

			p.checkWindowTrigger()

		}
		p.stepPixelFetcher()

		if incLX {
			p.lx++
		}

		// the window is considered to be active for the rest of the frame
		// if at some point during it happens that window is enabled while
		// LY == WY. this doesn't mean that the window will always be rendered,
		// as WX == LX condition is checked again during pixel transfer
		if p.winEnabled && p.ly == p.wy {
			p.winTriggerWy = true
		}

		// remain in the PixelTransferLX8 state until we reach LX==168
		p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)
		return
	case EnterHBlank: // variable
		p.mode = ModeHBlank
		p.statUpdate()
		p.b.Unlock(io.OAM | io.VRAM)

		dotsPassed := p.s.Cycle() - p.lineDot

		if p.s.DoubleSpeed() {
			dotsPassed >>= 1
		}

		if p.ly == 143 {
			p.lineState = HBlankLastLine
			p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 454-dotsPassed)
			return
		} else {
			p.lineState = HBlankUpdateOAM
		}
		// TODO verify timing
		if p.cgbMode {
			p.b.HandleHDMA()
		}

		p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 452-dotsPassed)
		return

	case HBlankUpdateOAM: // dot 452
		p.updateOAMStat()
		p.statUpdate()
	case HBlankUpdateLY: // dot 453
		p.ly++
		p.b.Set(types.LY, p.ly)

		// LYC_EQ_LY IRQ is disabled on dot 454
		p.LYC_EQ_LY_INT = false
		p.statUpdate()
		p.b.Lock(io.OAM)
	case HBlankUpdateLYCInt: // dot 454
		p.LYC_EQ_LY_INT = true
		p.statUpdate()
	case HBlankEnd: // dot 455
		p.enterOAMScan()
		return
	case HBlankLastLine: // dot 453
		p.ly++
		p.b.Set(types.LY, p.ly)
		p.LYC_EQ_LY_INT = false
		p.statUpdate()
	case HBlankLastLineLYCInt: // 454
		p.LYC_EQ_LY_INT = true
		p.statUpdate()
	case HBlankLastLineEnd:
		p.offscreenLineState = StartVBlank
		p.mode = ModeVBlank
		p.statUpdate()

		// raise vblank int as we're now entering vblank
		p.b.RaiseInterrupt(io.VBlankINT)
		p.handleOffscreenLine()
		return
	}

	p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, stateCycles[p.lineState])
	p.lineState++
}

type OffscreenLineState int

const (
	StartVBlank OffscreenLineState = iota
	VBlankUpdateLY
	StartVBlankLastLine
	VBlankLastLine
	VBlankLastLineUpdateLYCInt
	VBlankLastLineUpdateHBlank
	VBlankEnd
)

var offscreenLineCycles = []uint64{
	StartVBlank:                454, // 454
	VBlankUpdateLY:             2,   // 456
	StartVBlankLastLine:        1,   // 1
	VBlankLastLine:             5,   // 6 (+5)
	VBlankLastLineUpdateLYCInt: 447, // 452
	VBlankLastLineUpdateHBlank: 2,   // 454
	VBlankEnd:                  0,   // 456
}

// handleOffscreenLine handles lines 144 - 153
func (p *PPU) handleOffscreenLine() {
	p.b.Debugf("doing offscreen state: %s dot:%d frame dot: %d %d\n", p.offscreenLineState, p.s.Cycle()-p.lineDot, p.s.Cycle()-p.frameDot, p.ly)

	switch p.offscreenLineState {
	case StartVBlank: // dot 0
		p.lineDot = p.s.Cycle()

	case VBlankUpdateLY: // dot 454
		p.ly++
		p.b.Set(types.LY, p.ly)

		p.LYC_EQ_LY_INT = false
		p.statUpdate()

		// have we reached the end of frame?
		if p.ly == 153 {
			p.offscreenLineState = StartVBlankLastLine
			p.s.ScheduleEvent(scheduler.PPUHandleOffscreenLine, 2)
			return
		}

		p.offscreenLineState = StartVBlank
		// just restart vblank step otherwise
		p.s.ScheduleEvent(scheduler.PPUHandleOffscreenLine, 2)
		return
	case StartVBlankLastLine: // dot 0
		p.lineDot = p.s.Cycle()
	case VBlankLastLine: // dot 1
		p.ly = 0
		p.b.Set(types.LY, p.ly)

		p.LYC_EQ_LY_INT = false
		p.statUpdate()
	case VBlankLastLineUpdateLYCInt: // dot 6
		p.LYC_EQ_LY_INT = true
		p.statUpdate()
	case VBlankLastLineUpdateHBlank: // dot 453
		// STAT mode bits are reset on the last cycle (TODO verify)
		p.mode = ModeHBlank
		p.statUpdate()
	case VBlankEnd: // dot 455
		p.wly = 255
		p.frameDot = p.s.Cycle()
		p.winTriggerWy = false
		// restart frame
		p.enterOAMScan()

		p.updateOAMStat()
		return
	}

	p.s.ScheduleEvent(scheduler.PPUHandleOffscreenLine, offscreenLineCycles[p.offscreenLineState])
	p.offscreenLineState++
}

// enterOAMScan sets up the PPU for ModeOAM scanning.
func (p *PPU) enterOAMScan() {
	// reset OBJs
	p.objBuffer = []Sprite{}
	p.mode = ModeOAM
	p.statUpdate()

	p.b.Lock(io.OAM)

	// next event will be ReleaseOAMBus which should occur in 76 dots from now
	p.lineState = StartOAMScan
	p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)
}

// enterPixelTransfer sets up the PPU for ModeVRAM pixel transfer
// Due to the nature of the scheduler, this should be called 1 cycle
// before actually entering the pixel transfer state.
func (p *PPU) enterPixelTransfer() {
	// fill obj
	if p.objEnabled {
		// fill obj buffer
		p.objBuffer = []Sprite{}
		for i := uint16(0); i < 0xa0 && len(p.objBuffer) < 10; i += 4 {
			y, x, id, attr := p.b.Get(0xfe00+i), p.b.Get(0xfe00+i+1), p.b.Get(0xfe00+i+2), p.b.Get(0xfe00+i+3)

			if p.ly+16 >= y &&
				p.ly+16 < y+p.objSize {
				spr := Sprite{
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
	p.mode = ModeVRAM
	p.statUpdate() // clear stat line
	p.lineState = PixelTransferDummy
	p.b.Lock(io.OAM | io.VRAM)
	p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 3)
}

// enterHBlank sets up the PPU for the ModeHBlank.
func (p *PPU) enterHBlank() {

}

// stepPixelFetcher advances the pixel fetcher forward by 1 step depending
// on what state it's currently in.
func (p *PPU) stepPixelFetcher() {

	//p.b.Debugf("doing state fetch: %d dot:%d %d %d\n", p.FetcherState, p.s.Cycle()-p.lineDot, p.lx, p.bgFIFO.Size)
	// if the pixel slice fetcher is currently fetching a window, it could be aborted
	// due to WIN_EN being reset in LCDC. When this happens the fetcher simply switches
	// to BG fetching rather than restarting the process
	if p.fetcherState >= WinGetTileNoT1 && p.fetcherState <= WinGetTileDataHighT1 {
		if !p.winEnabled {
			// TODO determine which bg step to roll back to
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

		// determine x,y position on tilemap
		address |= (uint16((p.ly+p.scy)>>3) & 0x1f) << 5    // Y pos
		address |= (uint16(p.scx+p.bgFetcherX) >> 3) & 0x1f // X pos
		p.fetcherTileNoAddress = address

		// TODO verify when CGB accesses tile attrs
		if p.cgbMode {
			p.fetcherTileAttr = p.b.GetVRAM(p.fetcherTileNoAddress, 1)
		}
	case BGGetTileDataLowT1:
		p.fetcherData[0] = p.b.GetVRAM(p.getBGTileAddress(), p.fetcherTileAttr&types.Bit3>>3)
	case BGGetTileDataHighT1:
		p.fetcherData[1] = p.b.GetVRAM(p.getBGTileAddress()|1, p.fetcherTileAttr&types.Bit3>>3)
	case BGWinGetTileDataHighT2:
		if p.fetcherTileAttr&types.Bit5 > 0 {
			p.fetcherData[0] = bits.Reverse8(p.fetcherData[0])
			p.fetcherData[1] = bits.Reverse8(p.fetcherData[1])
		}
		p.bgFetcherX += 8

		// if there is a pending OBJ fetch and the BG FIFO isn't empty,
		// start fetching the OBJ and cache the current BG/Window fetch
		if p.canFetchOBJ() && p.bgFIFO.Size > 0 {
			p.cacheFetcher()

			// the first OBJ fetcher step should overlap with this dot,
			// rather than the push
			p.fetcherObj = true
			p.fetchingObj = p.objBuffer[0]
			p.objBuffer = p.objBuffer[1:]
			p.fetcherState = OBJGetTileNoT1
			p.stepPixelFetcher()

			return
		}

	// TODO handle immediate window push?
	case BGWinPushPixels:
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

			if p.winTriggerWx {
				return // OBJ fetches are ignored when fetching window
			}
		}

		// if we have hit an OBJ on the current LX, discard the currently
		// fetched pixels and start the obj
		if p.canFetchOBJ() {
			if !isEmpty {
				p.cacheFetcher()
			}

			p.fetcherObj = true
			p.fetchingObj = p.objBuffer[0]
			p.objBuffer = p.objBuffer[1:]
			p.fetcherState = OBJGetTileNoT1
			p.stepPixelFetcher()
			return
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
		// the push stage is the same as the BG so we reset to that state
		p.fetcherState = BGWinPushPixels
		return
	// OBJ
	case OBJGetTileNoT1:
		// read tile number from OAM
		// if a DMA transfer is in progress, a conflict can occur
		// where the address that the DMA is currently writing to
		// is accessed instead of the current OAM index. But only
		// during the write request stage (T0-T1)
		if p.s.Until(scheduler.DMATransfer) <= 4 && p.s.Until(scheduler.DMATransfer) > 2 {
			//p.objFetcherTileNo = p.b.Get(p.b.DMADestination() - 1)
		} else {
			p.objFetcherTileNo = p.fetchingObj.id
		}
	case OBJGetTileNoT2:
		if p.s.Until(scheduler.DMATransfer) == 3 {
			//p.objFetcherAttr = p.b.Get(p.b.DMADestination() - 1)
		} else {
			// read tile attr from OAM
			// same as above, DMA conflict can occur during the write
			// request stage of a DMA transfer
			p.objFetcherAttr = p.fetchingObj.attr

		}
	case OBJGetTileDataLowT2:
		p.objFetcherLow = p.b.GetVRAM(p.getOBJAddress(), p.objFetcherAttr&types.Bit3>>3)
	case OBJGetTileDataHighT2:
		p.objFetcherHigh = p.b.GetVRAM(p.getOBJAddress()|1, p.objFetcherAttr&types.Bit3>>3)
		if p.objFetcherAttr&types.Bit5 > 0 {
			p.objFetcherLow = bits.Reverse8(p.objFetcherLow)
			p.objFetcherHigh = bits.Reverse8(p.objFetcherHigh)
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
			objFIFO.Color = ((p.objFetcherLow & i) >> (7 - j)) | ((p.objFetcherHigh&i)>>(7-j))<<1

			if int(j) < p.objFIFO.Size {
				if p.objFIFO.GetIndex(int(j)).Color == 0 ||
					p.cgbMode &&
						objFIFO.Color != 0 && p.objFIFO.GetIndex(int(j)).OBJIndex > p.fetchingObj.index || p.objFIFO.GetIndex(int(j)).Color == 0 {
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
			p.fetcherState = OBJGetTileNoT1
		} else {
			p.fetcherObj = false

			// did this OBJ fetch interrupt a BG/Win fetch
			if p.fetcherDataCached {
				p.restoreFetcher()
				p.fetcherState = BGWinPushPixels
			} else {
				p.fetcherState = BGWinActivating
			}
		}
		return
	}

	p.fetcherState++
}

// getBGTileAddress determines the address to read from VRAM for the tile
// that is currently loaded in the pixel fetcher.
func (p *PPU) getBGTileAddress() uint16 {
	// read tile number from VRAM
	tileNo := p.b.GetVRAM(p.fetcherTileNoAddress, 0)

	// determine which tile address to read from VRAM
	address := uint16(0x0000)
	address |= uint16(tileNo) << 4                      // Tile ID offset
	address |= uint16(p.addressMode&^(tileNo>>7)) << 12 // Negation of ID.7 when LCDC.4 is set

	yPos := (p.ly + p.scy) & 7
	if p.fetcherTileAttr&types.Bit6 > 0 {
		yPos = ^yPos & 7
	}
	address |= uint16(yPos) << 1 // Y pos

	return address
}

// getOBJAddress determines the address to read from VRAM for the OBJ tile
// that is currently loaded in the pixel fetcher.
func (p *PPU) getOBJAddress() uint16 {
	address := uint16(0x0000)
	address |= uint16(p.objFetcherTileNo) << 4 // Tile ID offset

	// handle obj size
	tileY := p.fetchingObj.y - 16
	tileY = (p.ly - tileY) & (p.objSize - 1)

	// handle y pos
	if p.objFetcherAttr&types.Bit6 > 0 {
		tileY = ^tileY & (p.objSize - 1)
	}

	if p.objSize == 16 {
		tileY &= 0xfe
	}

	address |= uint16(tileY) << 1

	return address
}

// getWinTileAddress determines the address to read from VRAM for the tile
// that is currently loaded in the pixel fetcher.
func (p *PPU) getWinTileAddress() uint16 {
	// read tile number from vram
	tileNo := p.b.GetVRAM(p.fetcherTileNoAddress, 0)

	// determine where the tile is in VRAM
	address := uint16(0x0000)
	address |= uint16(tileNo) << 4                      // Tile ID offset
	address |= uint16(p.addressMode&^(tileNo>>7)) << 12 // Negation of ID.7 when LCDC.4 is set

	yPos := p.wly & 7
	if p.fetcherTileAttr&types.Bit6 > 0 {
		yPos = ^yPos & 7
	}

	address |= uint16(yPos) << 1

	return address
}

type FetcherState int

const (
	BGWinActivating FetcherState = iota
	BGGetTileNoT1
	BGGetTileNoT2
	BGGetTileDataLowT1
	BGGetTileDataHighT2
	BGGetTileDataHighT1
	BGWinGetTileDataHighT2
	BGWinPushPixels
	WinActivating
	WinGetTileNoT1
	WinGetTileNoT2
	WinGetTileDataLowT1
	WinGetTileDataLowT2
	WinGetTileDataHighT1
	OBJGetTileNoT1
	OBJGetTileNoT2
	OBJGetTileDataLowT1
	OBJGetTileDataLowT2
	OBJGetTileDataHighT1
	OBJGetTileDataHighT2
)

// statUpdate handles updating the STAT interrupt. As the conditions for
// raising a STAT interrupt are checked every dot, we need to call this
// whenever one of the dependent conditions changes otherwise it would
// be too expensive for the scheduler.
//
// The OAM STAT interrupt is an exception to this rule, it is only raised
// when transitioning modes, thus writing to STAT's OAM interrupt flag
// whilst already in OAM mode does not raise a request.
func (p *PPU) statUpdate() {
	if !p.enabled {
		// STAT & LYC call this but the PPU may be disabled when doing so
		// in which case the STAT line isn't processed
		return
	}
	lycEq := (p.ly == p.lyCompare) && p.LYC_EQ_LY_INT
	lycEqInt := lycEq && p.status&types.Bit6 > 0

	hblankInt := p.status&types.Bit3 > 0 && p.mode == ModeHBlank

	// vblank int is raised with both VBlank and OAM flag
	vblankInt := (p.status&types.Bit4 > 0 || p.status&types.Bit5 > 0) && p.mode == ModeVBlank

	p.updateINT(lycEqInt || hblankInt || vblankInt)

	//p.b.Debugf("LYC: %t LYC_INT: %t HBlank: %t VBlank: %t\n", lycEq, lycEqInt, hblankInt, vblankInt)

	// update LYC_EQ_LY flag in STAT register
	if lycEq {
		p.status |= types.Bit2
	} else {
		p.status &^= types.Bit2
	}
	p.b.Debugf("updated stat: %08b\n", p.status|p.mode)
}

// updateINT is used for raising the STAT interrupt on a rising edge.
func (p *PPU) updateINT(interrupt bool) {
	//p.b.Debugf("Updating STAT %t\n", interrupt)
	if !p.STAT_INT && interrupt {
		p.b.RaiseInterrupt(io.LCDINT)
		//p.b.Debugf("Raising INT %08b IE: %08b IF: %08b %d\n", p.status, p.b.Get(types.IE), p.b.Get(types.IF), p.s.Cycle()-p.lineDot)
	}

	p.STAT_INT = interrupt
}

// updateOAMStat handles raising the STAT int for OAM mode.
func (p *PPU) updateOAMStat() {
	p.updateINT((p.status&0x20 > 0) || (p.status&0x40 > 0 && p.ly == p.lyCompare && p.LYC_EQ_LY_INT))
}

// cacheFetcher caches the currently loaded low & high bytes
// in the pixel slice fetcher. Used for when an OBJ interrupts
// a BG/Win fetch.
func (p *PPU) cacheFetcher() {
	p.fetcherDataCached = true
	p.fetcherDataCache = p.fetcherData
}

// restoreFetcher restores the data loaded in cache.
func (p *PPU) restoreFetcher() {
	p.fetcherData = p.fetcherDataCache
	p.fetcherDataCached = false
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

// checkWindowTrigger checks to see if the window should be triggered
// for the current LX position.
func (p *PPU) checkWindowTrigger() {
	if p.winTriggerWy && !p.winTriggerWx && p.winEnabled && p.lx == p.wx {
		// activate window
		p.winTriggerWx = true

		// increase internal window line counter
		p.wly++

		// reset window tile counter
		p.winFetcherX = 0

		p.bgFIFO.Reset()

		// next tick is win prefetcher activating
		p.fetcherState = WinActivating
	}
}

// resetFetcher resets the pixel slice fetcher and the two pixel
// FIFOs so that they can be used for the next line.
func (p *PPU) resetFetcher() {
	p.bgFIFO.Reset()
	p.objFIFO.Reset()

	p.bgFetcherX = 0
	p.winFetcherX = 0
	p.winTriggerWx = false
	p.fetcherDataCached = false
	p.fetcherState = BGWinActivating
	p.lx = 0
}

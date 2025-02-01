package ppu

import (
	_ "embed"
	"encoding/csv"
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
	mode, modeToInt, ly, status uint8
	lx, wly                     int
	lyCompare                   uint8
	scy, scx                    uint8
	wy, wx                      uint8

	lyForComparison uint16
	lycINT, statINT bool

	cgbMode                      bool
	cRAM                         [128]uint8 // 64 bytes BG+OBJ
	bcpsIncrement, ocpsIncrement bool
	bcpsIndex, ocpsIndex         uint8

	// external components
	b         *io.Bus
	fifoCycle uint64
	s         *scheduler.Scheduler

	// palettes used for DMG -> CGB colourisation
	BGColourisationPalette   Palette
	OBJ0ColourisationPalette Palette
	OBJ1ColourisationPalette Palette

	// palettes used by the ppu to display colours
	ColourPalette       ColourPalette
	ColourSpritePalette ColourPalette

	PreparedFrame [ScreenHeight][ScreenWidth][3]uint8
	DebugView     [ScreenHeight][ScreenWidth][3]uint8

	// debug
	Debug struct {
		SpritesDisabled, BackgroundDisabled, WindowDisabled bool
	}

	bgFIFO, objFIFO       *utils.FIFO[FIFOEntry]
	bgFetcher, objFetcher *Fetcher
	bgFetcherStep         FetcherStep
	objStep               objStep

	fetcherTileNo, fetcherTileAttr uint8
	fetcherData                    [2]uint8
	objFetcherLow, objFetcherHigh  uint8
	fetcherObj                     bool
	fetchingObj                    Sprite
	fetcherWin                     bool
	winTriggerWy, winTriggerWx     bool

	objBuffer   []Sprite
	visibleObjs int

	fetcherTileNoAddress, fetcherAddress        uint16
	objFetcherLowAddress, objFetcherHighAddress uint16
	linePos                                     int
	lineState                                   lineState
	glitchedLineState                           glitchedLineState
	offscreenLineState                          offscreenLineState
	winTileX                                    uint8
}

type DebugColor uint32

const (
	BGPalette  DebugColor = 0x0000ffff
	SCXPalette DebugColor = 0xffff00ff
)

func (d DebugColor) RGB() [3]uint8 {
	return [3]uint8{uint8(d >> 24), uint8(d >> 16), uint8(d >> 8)}
}

// A FIFOEntry represents a single pixel in a FIFO.
type FIFOEntry struct {
	Color      uint8
	Palette    uint8
	Attributes uint8
	OBJIndex   uint8
}

type Fetcher struct {
	Step                                  FetcherStep
	FetcherLowAddress, FetcherHighAddress uint16
	FetcherLow, FetcherHigh               uint8
}

type FetcherStep int

const (
	GetTileIDT1 FetcherStep = iota
	GetTileIDT2
	GetTileRowLowT1
	GetTileRowLowT2
	GetTileRowHighT1
	GetTileRowHighT2
	PushPixels
)

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
			p.lyForComparison = 0

			p.b.Unlock(io.OAM | io.VRAM)
			p.lineState = startLine
			p.glitchedLineState = startGlitchedLine
		} else if !p.enabled && v&types.Bit7 != 0 {
			p.enabled = true
			p.cleared = false
			p.linePos = -16
			p.cgbMode = b.IsGBCCart() && b.IsGBC()
			p.glitchedLineState = startGlitchedLine
			p.statUpdate()
			p.s.ScheduleEvent(scheduler.PPUHandleGlitchedLine0, 1)
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
		p.b.Debugf("writing stat %08b\n", v)
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
		if p.ly < 3 {
			p.b.Debugf("reading stat %d %08b\n", p.ly, p.status|p.mode)
		}
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
		if b.IsBooting() {
			p.ly = v
			return v
		}
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
		if v == b.Get(types.BGP) || p.b.IsGBCCart() && p.b.IsGBC() {
			return v
		}
		p.ColourPalette[0] = p.BGColourisationPalette.remap(v)

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
	s.RegisterEvent(scheduler.PPUHandleGlitchedLine0, p.handleGlitchedLine0)
	s.RegisterEvent(scheduler.PPUHandleOffscreenLine, p.handleOffscreenLine)
	return p
}

type glitchedLineState int

const (
	startGlitchedLine glitchedLineState = iota
	glitchedLineOAMWBlock
	glitchedLineEndOAM
	glitchedLineStartVRAM
)

var (
	glitchedLineCycles = []uint64{
		startGlitchedLine:     76, // 77
		glitchedLineOAMWBlock: 2,  // 79
		glitchedLineEndOAM:    5,  // 84
		glitchedLineStartVRAM: 0,  // 83 -
	}
)

// handleGlitchedLine0 handles the very first line after turning the LCD on. Timing is a bit different
// from normal, being cut short by 8 dots, whilst also presenting as ModeHBlank when it would otherwise
// be in ModeOAM. The OAM scanning is also skipped, although this doesn't matter as the LCD doesn't start
// displaying a signal from the PPU until the first VBlank is reached.
func (p *PPU) handleGlitchedLine0() {
	switch p.glitchedLineState {
	case startGlitchedLine:
		p.ly, p.lyForComparison = 0, 0
		p.b.Set(types.LY, 0)
		p.linePos = -16
		p.mode = ModeHBlank
		p.modeToInt = 255
		p.b.Unlock(io.OAM | io.VRAM)
		p.statUpdate()
		p.fifoCycle = p.s.Cycle()
	case glitchedLineOAMWBlock:
		p.b.WLock(io.OAM)
		p.statUpdate()
	case glitchedLineEndOAM:
		p.mode, p.modeToInt = ModeVRAM, ModeVRAM
		p.b.Lock(io.OAM)
		if p.s.DoubleSpeed() {
			p.b.Lock(io.VRAM)
		}
		if !p.cgbMode {
			p.b.Lock(io.VRAM)
		}
	case glitchedLineStartVRAM:
		p.b.Lock(io.VRAM)
		p.lineState = startFifo
		p.fifoCycle -= 8
		p.handleVisualLine()
		return
	}

	p.s.ScheduleEvent(scheduler.PPUHandleGlitchedLine0, glitchedLineCycles[p.glitchedLineState])
	p.glitchedLineState++
}

type lineState int

const (
	startLine lineState = iota
	cgbOAMWBlock
	updateLY
	startOAM
	oamIndex37
	finishOAM
	startFifo
	fifo
	finishVRAM
	startHBlank
	finishHBlank
)

var stateCycles = []uint64{
	startLine:    2,  // 2
	cgbOAMWBlock: 1,  // 3
	updateLY:     1,  // 4
	startOAM:     76, // 78
	oamIndex37:   4,  // 84
	finishOAM:    5,  // 89
	startFifo:    0,
	fifo:         1,   // 256-373 (167.5 - 284.5 cycles)
	finishVRAM:   1,   // 257
	startHBlank:  199, // 456 (depending on how many cycles fifo took)
}

type objStep int

const (
	startObjStep objStep = iota
	getRowLow
	getRowHigh
	pushPixels
)

var objStepCycles = []uint64{
	startObjStep: 2,
	getRowLow:    2,
	getRowHigh:   1,
}

// handleVisualLine handles lines 0 -> 143, stepping from OAM -> VRAM -> HBlank until line 143 is reached.
func (p *PPU) handleVisualLine() {
	if p.lineState != fifo {
		//p.b.Debugf("doing state: %d dot:%d scx:%d\n", p.lineState, p.fifoCycle, p.scx)
	}
	switch p.lineState {
	case startLine: // 0 -> 2
		p.handleWY()
		p.fifoCycle = p.s.Cycle()
		p.b.WBlock(io.OAM, p.cgbMode && !p.s.DoubleSpeed())
	case cgbOAMWBlock: // 2 -> 3
		p.b.WBlock(io.OAM, p.cgbMode)
	case updateLY: // 3 -> 4
		p.b.Set(types.LY, p.ly)
		p.b.RBlock(io.OAM, !p.s.DoubleSpeed())

		if p.ly != 0 {
			p.lyForComparison = 0xffff
		} else {
			p.lyForComparison = 0
		}

		// OAM stat int fires 1 T-Cycle before STAT changes, except on line 0
		if p.ly != 0 {
			p.modeToInt = ModeOAM
			p.mode = ModeHBlank
		} else if !p.cgbMode {
			p.mode = ModeHBlank
		}
		p.statUpdate()
	case startOAM:
		p.b.Lock(io.OAM)
		p.mode, p.modeToInt = ModeOAM, ModeOAM
		p.lyForComparison = uint16(p.ly)
		p.handleWY()
		p.statUpdate()
		p.modeToInt = 255
		p.statUpdate()
	case oamIndex37: //
		p.b.RBlock(io.VRAM, !p.cgbMode)
		p.b.WBlock(io.OAM, p.cgbMode)
		p.b.WUnlock(io.VRAM)
	case finishOAM: // = 84 cycles
		p.mode, p.modeToInt = ModeVRAM, ModeVRAM
		p.b.Lock(io.OAM | io.VRAM)
		p.statUpdate() // no VRAM int so this should clear the stat line

		if p.objEnabled || p.cgbMode {
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
			p.visibleObjs = len(p.objBuffer)
		}
	case startFifo:
		p.bgFIFO.Reset()
		p.objFIFO.Reset()
		p.lx = 0
		p.bgFIFO.Size = 8
		p.lineState = fifo
		p.bgFetcherStep = 0
		p.fetcherObj = false
		fallthrough
	case fifo:
		// is the window trying to piggyback the BG fetcher?
		if !p.winTriggerWx && p.winTriggerWy && p.winEnabled {
			windowActivated := false
			if p.wx == 0 {
				if p.linePos == -7 {
					windowActivated = true
				} else if p.linePos == -16 && p.scx&7 > 0 {
					windowActivated = true
				} else if p.linePos >= -15 && p.linePos <= -8 {
					windowActivated = true
				}
			} else if p.wx < 166 {
				if p.wx == uint8(p.linePos)+7 {
					windowActivated = true
				}
			}

			// have we met the conditions to start fetching a window?
			if windowActivated {
				p.wly++
				p.winTileX = 0

				// fetching the window clears the BG fifo and resets the fetcher
				p.bgFIFO.Reset()
				p.winTriggerWx = true
				p.bgFetcherStep = GetTileIDT1
			}
		}

		// is the PPU currently fetching an OBJ?
		if p.fetcherObj {
			switch p.objStep {
			case startObjStep:
				p.stepPixelFetcher()
			case getRowLow:
				p.stepPixelFetcher()
				p.objFetcherLow = p.b.GetVRAM(p.getOBJRow(p.fetchingObj.id), p.fetchingObj.attr&types.Bit3>>3)
			case getRowHigh:
				p.objFetcherHigh = p.b.GetVRAM(p.getOBJRow(p.fetchingObj.id)|1, p.fetchingObj.attr&types.Bit3>>3)
				if p.fetchingObj.attr&types.Bit5 > 0 {
					p.objFetcherLow = bits.Reverse8(p.objFetcherLow)
					p.objFetcherHigh = bits.Reverse8(p.objFetcherHigh)
				}
			case pushPixels:
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
				p.fetcherObj = false
				p.visibleObjs--

				goto checkObj
				// TODO here go back to check for obj
			}

			p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, objStepCycles[p.objStep])
			p.objStep++

			return
		}

	checkObj:
		for p.visibleObjs > 0 && p.objBuffer[0].x < p.xForObj() && (p.objEnabled || p.cgbMode) {
			p.visibleObjs--
			p.objBuffer = p.objBuffer[1:]
		}

		// are there any pending OBJs on this X coordinate?
		if p.visibleObjs > 0 && (p.objEnabled || p.cgbMode) && p.objBuffer[0].x == p.xForObj() {
			// finish the current BG fetch (if needed)
			if p.bgFetcherStep < GetTileRowHighT2 || p.bgFIFO.Size == 0 {
				p.stepPixelFetcher()
				p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1) // sleep for 1 cycle
			} else {
				p.stepPixelFetcher()
				p.fetcherObj = true
				p.objStep = startObjStep
				p.fetchingObj = p.objBuffer[0]
				p.objBuffer = p.objBuffer[1:]
				p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)
			}

			return // OBJ fetching immediately halts pushing pixels to the LCD
		}

		p.pushPixel()
		p.stepPixelFetcher()
		if p.linePos != 160 {
			// end of line handle here
			p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 1)
			return
		}

		p.linePos = -16
		if !p.s.DoubleSpeed() {
			p.mode, p.modeToInt = ModeHBlank, ModeHBlank
			p.b.Unlock(io.OAM | io.VRAM)
		}
		p.lineState = finishVRAM
		p.winTriggerWx = false
	case startHBlank:
		p.modeToInt, p.modeToInt = ModeHBlank, ModeHBlank
		p.b.Unlock(io.OAM | io.VRAM)
		p.statUpdate()

		if p.cgbMode {
			p.b.HandleHDMA()
		}
		dotsPast := p.s.Cycle() - p.fifoCycle
		if p.s.DoubleSpeed() {
			dotsPast >>= 1
		}

		p.b.Debugf("starting hblank at %d %d\n", p.s.Cycle()-p.fifoCycle, p.fifoCycle)
		p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, 456-dotsPast)

		p.lineState = finishHBlank
		return
	case finishHBlank:
		if p.ly != 143 {
			p.modeToInt = ModeOAM
		}
		p.ly++
		// if we are on line 144, we are entering ModeVBlank
		if p.ly == 144 {
			// was the LCD just turned on? (the Game Boy never receives the first frame after turning on the LCD)
			if !p.cleared {
				p.renderBlank()
			}

			p.offscreenLineState = startVBlank
			p.handleOffscreenLine()
		} else {
			// go to OAM search
			p.b.Debugf("finishing HBlank at %d %d\n", p.s.Cycle(), p.fifoCycle)
			p.lineState = startLine
			p.handleVisualLine()
		}

		return
	}

	p.s.ScheduleEvent(scheduler.PPUHandleVisualLine, stateCycles[p.lineState])
	p.lineState++
}

type offscreenLineState int

const (
	startVBlank offscreenLineState = iota
	updateLYVBlank
	updateLYCVBlank
	handleVBlankInt
	line153Start
	line153LYUpdate
	line153LY0
	line153LYC
	line153LYC0
	endFrame
)

var offscreenLineCycles = []uint64{
	startVBlank:     2,   // 2
	updateLYVBlank:  2,   // 4
	updateLYCVBlank: 1,   // 5
	handleVBlankInt: 451, // 456
	line153Start:    2,   // 2
	line153LYUpdate: 4,   // 6
	line153LY0:      2,   // 8
	line153LYC:      4,   // 12
	line153LYC0:     444, // 456
}

// handleOffscreenLine handles lines 144 - 153
func (p *PPU) handleOffscreenLine() {
	switch p.offscreenLineState {
	case startVBlank:
		p.lyForComparison = 0xffff
		p.statUpdate()
	case updateLYVBlank:
		p.b.Set(types.LY, p.ly)
		if p.b.Model() >= types.CGB0 && p.ly == 144 && !p.statINT && p.status&0x20 > 0 {
			p.b.RaiseInterrupt(io.LCDINT)
		}
	case updateLYCVBlank:
		p.lyForComparison = uint16(p.ly)
		p.statUpdate()
	case handleVBlankInt:
		switch p.ly {
		case 144: // Entering VBlank
			p.mode, p.modeToInt = ModeVBlank, ModeVBlank
			p.b.RaiseInterrupt(io.VBlankINT)

			// entering vblank also triggers the OAM interrupt
			if p.b.Model() < types.CGB0 && !p.statINT && p.status&0x20 > 0 {
				p.b.RaiseInterrupt(io.LCDINT)
			}

			p.statUpdate()
		case 152: // Leaving VBlank
			p.offscreenLineState = line153Start
			p.ly++
			p.s.ScheduleEvent(scheduler.PPUHandleOffscreenLine, offscreenLineCycles[handleVBlankInt])
			return
		}

		p.ly++
		p.offscreenLineState = startVBlank
		p.s.ScheduleEvent(scheduler.PPUHandleOffscreenLine, offscreenLineCycles[handleVBlankInt]) // loop back around, accounting for inc
		return
	case line153Start:
		p.lyForComparison = 0xffff
		p.statUpdate()
	case line153LYUpdate:
		p.b.Set(types.LY, 153)
	case line153LY0:
		p.b.Set(types.LY, 0)
		p.lyForComparison = 153
		p.statUpdate()
	case line153LYC:
		p.lyForComparison = 0xffff
		p.statUpdate()
	case line153LYC0:
		p.lyForComparison = 0
		p.statUpdate()
	case endFrame:
		p.ly = 0
		p.winTriggerWy = false
		p.wly = -1
		p.lineState = startLine
		p.handleVisualLine()
		return
	}

	p.s.ScheduleEvent(scheduler.PPUHandleOffscreenLine, offscreenLineCycles[p.offscreenLineState])
	p.offscreenLineState++
}

func (p *PPU) xForObj() uint8 {
	ret := uint8(p.linePos) + 8
	if ret > 240 {
		return 0
	}

	return ret
}

func (p *PPU) stepPixelFetcher() {
	mode := BG
	switch p.bgFetcherStep {
	case GetTileIDT1:
		// window can be disabled mid-scanline?
		if !p.winEnabled {
			p.winTriggerWx = false
		}
		if p.winTriggerWx {
			mode = Window
		}

		p.fetcherTileNoAddress = p.getTileID(mode)
		p.bgFetcherStep++
	case GetTileIDT2:
		p.fetcherTileNo = p.b.GetVRAM(p.fetcherTileNoAddress, 0)
		if p.cgbMode { // CGB access both tile ID & attributes on the same dot
			p.fetcherTileAttr = p.b.GetVRAM(p.fetcherTileNoAddress, 1)
		}
		p.bgFetcherStep++
	case GetTileRowLowT1:
		if p.winTriggerWx {
			mode = Window
		}
		p.fetcherAddress = p.getTileRow(mode, p.fetcherTileNo)
		p.bgFetcherStep++
	case GetTileRowLowT2:
		p.fetcherData[0] = p.b.GetVRAM(p.fetcherAddress, p.fetcherTileAttr&types.Bit3>>3)
		p.bgFetcherStep++
	case GetTileRowHighT1:
		if p.winTriggerWx {
			mode = Window
		}
		p.fetcherAddress = p.getTileRow(mode, p.fetcherTileNo) | 1
		p.bgFetcherStep++
	case GetTileRowHighT2:
		p.fetcherData[1] = p.b.GetVRAM(p.fetcherAddress, p.fetcherTileAttr&types.Bit3>>3)
		p.bgFetcherStep++
		if p.fetcherTileAttr&types.Bit5 > 0 {
			p.fetcherData[0] = bits.Reverse8(p.fetcherData[0])
			p.fetcherData[1] = bits.Reverse8(p.fetcherData[1])
		}

		// was that a window fetch?
		if p.winTriggerWx {
			p.winTileX++
			p.winTileX &= 0x1f
		}

		fallthrough
	case PushPixels:
		p.bgFetcherStep = PushPixels
		// is the FIFO accepting pixels?
		if p.bgFIFO.Size > 0 {
			break // can't push to FIFO when it has pixels in it
		}

		if p.bgEnabled || p.cgbMode {
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
		}

		p.bgFIFO.Size = 8
		p.bgFetcherStep = GetTileIDT1
	}
}

// pushPixel attempts to push a pixel from one of the FIFOs to the LCD.
func (p *PPU) pushPixel() {
	if p.visibleObjs > 0 && p.objBuffer[0].x == 0 && (p.objEnabled || p.cgbMode) {
		return // pixels aren't pushed when an OBJ at X=0 is pending
	}

	if p.bgFIFO.Size == 0 {
		return // can only push when the BG FIFO isn't empty
	}

	var bgPX, objPX *FIFOEntry
	var bgPriority, drawOAM bool
	var bgEnabled = true

	bgPX = p.bgFIFO.Pop()
	bgPriority = bgPX.Attributes&types.Bit7 > 0

	// are there any pending OBJ pixels?
	if p.objFIFO.Size > 0 {
		objPX = p.objFIFO.Pop()

		// pixels with color=0 aren't drawn
		if objPX.Color > 0 && p.objEnabled {
			drawOAM = true
			if objPX.Attributes&types.Bit7 > 0 {
				bgPriority = true
			}
		}
	}

	if p.linePos+16 < 8 {
		if p.linePos&7 == int(p.scx)&7 {
			p.linePos = -8
		}
	}

	if p.linePos >= 160 || p.linePos < 0 {
		p.linePos++
		return
	}

	if !p.bgEnabled {
		if p.cgbMode {
			bgPriority = false
		} else {
			bgEnabled = false
		}
	}

	var pixel uint8
	if bgEnabled {
		pixel = bgPX.Color
	}
	if pixel > 0 && bgPriority {
		drawOAM = false
	}
	p.PreparedFrame[p.ly][p.lx] = p.ColourPalette[bgPX.Attributes&7][pixel]

	if drawOAM {
		p.PreparedFrame[p.ly][p.lx] = p.ColourSpritePalette[objPX.Palette][objPX.Color]
	}

	p.linePos++
	p.lx++
}

type FetcherMode int

const (
	BG FetcherMode = iota
	Window
)

// getTileID determines the address for which background/window tile to fetch pixels from.
func (p *PPU) getTileID(mode FetcherMode) uint16 {
	address := uint16(0x1800)

	switch mode {
	case BG:
		address |= uint16(p.bgTileMap) << 10
		address |= uint16(p.ly+p.scy) >> 3 << 5
		var x int
		if p.linePos+16 < 8 {
			x = int(p.scx) >> 3
		} else {
			x = ((int(p.scx) + p.linePos + 8) >> 3) & 0x1f
		}
		address |= uint16(x)
	case Window:
		address |= uint16(p.winTileMap) << 10
		address |= (uint16(p.wly) >> 3) << 5
		address |= uint16(p.winTileX)
		//address |= uint16(p.fetcherX)
	}

	return address
}

// getTileRow fetches one slice of the bitplane currently being read.
func (p *PPU) getTileRow(mode FetcherMode, id uint8) uint16 {
	address := uint16(0x0000)
	address |= uint16(id) << 4                      // Tile ID Offset
	address |= uint16(p.addressMode&^(id>>7)) << 12 // Negation of id.7 when LCDC.4 is set
	attr := p.fetcherTileAttr                       // Objects & CGB Only
	yPos := uint16(0)

	switch mode {
	case BG:
		yPos = uint16(p.ly+p.scy) & 7
	case Window:
		yPos = uint16(p.wly & 7)
	}

	if attr&types.Bit6 > 0 { // Y-Flip (CGB Only)
		yPos = ^yPos & 7
	}
	address |= yPos << 1 // Y-Pos Offset

	return address
}

func (p *PPU) getOBJRow(id uint8) uint16 {
	address := uint16(0x0000)
	address |= uint16(id) << 4 // Tile ID Offset
	attr := p.fetchingObj.attr // Objects & CGB Only
	yPos := (uint16(p.ly) - uint16(p.fetchingObj.y)) & 7

	if attr&types.Bit6 > 0 { // Y-Flip (Objects & CGB Only)
		yPos = ^yPos & 7
	}
	address |= yPos << 1 // Y-Pos Offset

	return address
}

func (p *PPU) statUpdate() {
	if !p.enabled {
		return
	}

	// get previous interrupt state
	prevInterruptLine := p.statINT

	// handle LY=LYC
	if p.lyForComparison != 0xffff || p.b.Model() <= types.CGBABC && !p.s.DoubleSpeed() {
		if uint8(p.lyForComparison) == p.lyCompare {
			p.lycINT = true
			p.status |= types.Bit2
		} else {
			if p.lyForComparison != 0xffff {
				p.lycINT = false
			}
			p.status &^= types.Bit2
		}
	}

	// handle stat int
	p.statINT = (p.modeToInt == ModeHBlank && p.status&types.Bit3 != 0) ||
		(p.modeToInt == ModeVBlank && p.status&types.Bit4 != 0) ||
		(p.modeToInt == ModeOAM && p.status&types.Bit5 != 0) ||
		(p.lycINT && p.status&types.Bit6 != 0)

	// trigger interrupt if needed
	if p.statINT && !prevInterruptLine {
		if p.modeToInt == ModeHBlank {
			p.b.Debugf("STAT INT %08b %d %d\n", p.status, p.modeToInt, p.s.Cycle())
		}
		p.b.RaiseInterrupt(io.LCDINT)
	}
}

// handleWY checks to see if the window has been enabled
func (p *PPU) handleWY() {
	if !p.enabled {
		return
	}

	comparison := p.ly
	if (!p.cgbMode || p.s.DoubleSpeed()) && p.lyForComparison != 0xffff {
		comparison = uint8(p.lyForComparison)
	}

	if p.winEnabled && p.wy == comparison {
		p.winTriggerWy = true
	}
}

func (p *PPU) fetcherY() uint8 {
	if p.winTriggerWx {
		return uint8(p.wly)
	} else {
		return p.ly + p.scy
	}
}

func (p *PPU) renderBlank() {
	for y := uint8(0); y < ScreenHeight; y++ {
		for x := uint8(0); x < ScreenWidth; x++ {
			p.PreparedFrame[y][x] = p.ColourPalette[0][0] // TODO handle GBC
		}
	}
	p.cleared = true
}

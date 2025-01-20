package ppu

import (
	_ "embed"
	"encoding/csv"
	"math/bits"
	"sort"
	"strconv"
	"strings"

	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
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
	x, y uint8
	attr uint8
	id   uint8
	TileEntry
}

// TileEntry is used to define a tile in the tile map or OAM.
type TileEntry struct {
	id, paletteNumber, vRAMBank uint8
	priority, xFlip, yFlip      bool
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
	lx                          int
	lyCompare, wly              uint8

	lyForComparison uint16
	lycINT, statINT bool

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

	// debug
	Debug struct {
		SpritesDisabled, BackgroundDisabled, WindowDisabled bool
	}

	bgFIFO, objFIFO               FIFO
	bgFetcherStep, objFetcherStep FIFOStep

	fetcherTileNo, fetcherTileAttr uint8
	fetcherX                       uint8
	fetcherLow, fetcherHigh        uint8
	objFetcherLow, objFetcherHigh  uint8
	fetcherObj                     bool
	fetchingObj                    Sprite
	fetcherWin                     bool
	winTriggerWy                   bool

	objBuffer []Sprite
}

// A FIFO can hold up to 8 pixels, the width of one tile.
type FIFO []FIFOEntry

// A FIFOEntry represents a single pixel in a FIFO and is composed of
// 2 bytes and four properties.
type FIFOEntry struct {
	Color      uint8
	Palette    uint8
	Attributes uint8
	ID         uint8
}

type FIFOStep int

const (
	GetTileID FIFOStep = iota
	_
	GetTileRowLow
	_
	GetTileRowHigh
	_
	_
	PushPixels
)

func New(b *io.Bus, s *scheduler.Scheduler) *PPU {
	p := &PPU{
		b: b,
		s: s,
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
			for i := scheduler.PPUHBlank; i <= scheduler.PPUOAMInterrupt; i++ {
				p.s.DescheduleEvent(i)
			}

			// when the LCD is off, LY reads 0, and STAT mode reads 0 (HBlank)
			p.b.Set(types.LY, 0)
			p.ly, p.mode = 0, 0

			p.b.Unlock(io.OAM)
			p.b.Unlock(io.VRAM)
		} else if !p.enabled && v&types.Bit7 != 0 {
			p.enabled = true
			// reset LYC to compare against and clear coincidence flag
			p.lyForComparison, p.ly = 0, 0
			p.b.Set(types.LY, 0)

			p.modeToInt = 255
			p.statUpdate()

			p.cleared = false
			p.s.ScheduleEvent(scheduler.PPUStartGlitchedLine0, 76)
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
		// writing to STAT briefly enables all STAT interrupts but only on DMG. Road Rash relies
		// on this bug, so maybe add a warning for users trying to play Road Rash in CGB mode
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
		return v
	})
	b.ReserveAddress(types.SCX, func(v byte) byte {
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
		return v
	})
	b.ReserveAddress(types.WX, func(v byte) byte {
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
		if p.b.IsGBCCart() {
			b.ReserveAddress(types.OPRI, func(v byte) byte {
				return v | 0xfe
			})
		}
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
	s.RegisterEvent(scheduler.PPUBeginFIFO, p.beginFIFO)
	s.RegisterEvent(scheduler.PPUFIFOTransfer, p.fifoTransfer)
	s.RegisterEvent(scheduler.PPUHBlankInterrupt, func() {
		p.modeToInt = ModeHBlank
		p.statUpdate()
		p.modeToInt = ModeVRAM

		p.s.ScheduleEvent(scheduler.PPUVRAMTransfer, 4)
	})

	return p
}

// startGlitchedFirstLine is called 76 cycles after the PPU is enabled,
// performing the first line of the screen in a glitched manner.
func (p *PPU) startGlitchedFirstLine() {
	p.statUpdate() // this occurs before the mode change, modeToInt should be 255 here
	p.modeToInt = ModeVRAM

	p.s.ScheduleEvent(scheduler.PPUContinueGlitchedLine0, 4)
}

// continueGlitchedFirstLine is called 4 cycles after startGlitchedFirstLine,
// continuing the first line of the screen in a glitched manner.
func (p *PPU) continueGlitchedFirstLine() {
	// OAM & VRAM are blocked until the end of VRAM transfer
	p.b.Lock(io.OAM)
	p.b.Lock(io.VRAM)
	p.mode = ModeVRAM

	p.s.ScheduleEvent(scheduler.PPUMiddleGlitchedLine0, 172)
}

func (p *PPU) middleGlitchedLine0() {
	p.mode, p.modeToInt = ModeHBlank, ModeHBlank
	p.statUpdate()

	p.b.Unlock(io.OAM)
	p.b.Unlock(io.VRAM)

	p.s.ScheduleEvent(scheduler.PPUEndGlitchedLine0, 196)
}

func (p *PPU) endGlitchedLine0() {
	p.modeToInt = 2
	p.s.ScheduleEvent(scheduler.PPUHBlank, 4)
}

func (p *PPU) endHBlank() {
	// increment current scanline
	p.ly++
	p.modeToInt = ModeOAM

	// if we are on line 144, we are entering ModeVBlank
	if p.ly == 144 {
		// was the LCD just turned on? (the Game Boy never receives the first frame after turning on the LCD)
		if !p.cleared {
			p.renderBlank()
		}

		p.startVBlank()
	} else {
		// go to OAM search
		p.startOAM()
	}

	if p.fetcherWin {
		p.wly++
	}
}

func (p *PPU) endVRAMTransfer() {
	p.mode, p.modeToInt = ModeHBlank, ModeHBlank

	p.b.Unlock(io.OAM)
	p.b.Unlock(io.VRAM)

	p.s.ScheduleEvent(scheduler.PPUStartHBlank, 4)
}

func (p *PPU) startHBlank() {
	p.statUpdate()

	if p.b.IsGBCCart() {
		p.b.HandleHDMA()
	}

	// schedule end of ModeHBlank
	p.s.ScheduleEvent(scheduler.PPUHBlank, uint64(196-scroll(p.b.Get(types.SCX)))-p.fifoCycle)
	p.fifoCycle = 0
}

// startOAM is performed on the first cycle of lines 0 to 143, and performs
// the OAM search for the current line. The OAM search lasts until cycle 88,
// when Mode 3 (VRAM) is entered.
//
// Lines 0 - 144:
//
//	OAM Search: 4 -> 84
func (p *PPU) startOAM() {
	p.b.Set(types.LY, p.ly) // update LY

	p.mode = ModeHBlank
	// OAM STAT int occurs 1-M cycle before STAT changes, except on line 0
	if p.ly != 0 {
		p.modeToInt = 2
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
	p.mode, p.modeToInt = ModeOAM, ModeOAM
	p.lyForComparison = uint16(p.ly)
	p.statUpdate()

	p.modeToInt = 255
	p.statUpdate()

	p.b.WLock(io.OAM)

	p.s.ScheduleEvent(scheduler.PPUPrepareEndOAMSearch, 76)

	if !p.winTriggerWy {
		p.winTriggerWy = p.b.Get(types.WY) == p.ly
	}
}

// endOAM is performed 80 cycles after startOAM, and performs the
// rest of the OAM search.
func (p *PPU) endOAM() {
	p.mode, p.modeToInt = ModeVRAM, ModeVRAM
	p.statUpdate()

	p.b.Lock(io.OAM | io.VRAM)

	p.fetcherWin = false

	// fill obj buffer
	p.objBuffer = []Sprite{}
	for i := uint16(0); i < 0xa0 && len(p.objBuffer) < 10; i += 4 {
		y, x, id, attr := p.b.Get(0xfe00+i), p.b.Get(0xfe00+i+1), p.b.Get(0xfe00+i+2), p.b.Get(0xfe00+i+3)

		if p.ly+16 >= y &&
			p.ly+16 < y+p.objSize {
			spr := Sprite{
				y:    y,
				x:    x,
				id:   id,
				attr: attr,
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

	// schedule beginning of fifo
	p.s.ScheduleEvent(scheduler.PPUBeginFIFO, 2)
}

func (p *PPU) beginFIFO() {
	// clear fifo stuff
	p.bgFIFO, p.objFIFO = FIFO{}, FIFO{}
	p.lx = -int(p.b.Get(types.SCX) % 8)
	p.fetcherX = 0
	p.bgFetcherStep = 0
	p.objFetcherStep = 0
	p.fetcherObj = false

	// output to the display is delayed by SCX % 8 dots while the pixels are discarded from the fetched tile
	p.fifoTransfer()
}

func (p *PPU) fifoTransfer() {
	// does an obj need to be fetched?
	if len(p.objBuffer) > 0 && p.objEnabled && !p.fetcherObj {
		if int(p.objBuffer[0].x) <= p.lx+8 {
			p.fetchingObj = p.objBuffer[0]
			p.objBuffer = p.objBuffer[1:]
			p.fetcherObj = true
			p.objFetcherStep = 0
		}
	}

	// handle obj fetcher
	if p.fetcherObj {
		p.fifoCycle++
		switch p.objFetcherStep {
		case GetTileRowLow:
			p.objFetcherLow = p.getTileRow(OBJ, p.fetchingObj.id, false)
		case GetTileRowHigh:
			p.objFetcherHigh = p.getTileRow(OBJ, p.fetchingObj.id, true)
		case PushPixels:
			if p.fetchingObj.attr&types.Bit5 > 0 { // X-Flip (CGB Only)
				p.objFetcherLow = bits.Reverse8(p.objFetcherLow)
				p.objFetcherHigh = bits.Reverse8(p.objFetcherHigh)
			}

			j := uint8(0)
			for i := uint8(0x80); i > 0; i >>= 1 {
				if p.fetchingObj.x+j < 8 {

					continue // offscreen
				}

				objFIFO := FIFOEntry{
					Attributes: p.fetchingObj.attr,
					Palette:    p.fetchingObj.attr & types.Bit4 >> 4,
				}
				if p.b.IsGBCCart() && p.b.IsGBC() {
					objFIFO.Palette = p.fetchingObj.attr & 7
				}
				if p.objFetcherLow&i > 0 {
					objFIFO.Color |= 1
				}
				if p.objFetcherHigh&i > 0 {
					objFIFO.Color |= 2
				}

				if len(p.objFIFO) > 0 && int(j) < len(p.objFIFO) {
					if p.objFIFO[j].Color == 0 || p.b.IsGBCCart() && p.b.IsGBC() && p.b.Get(types.OPRI)&types.Bit0 == 0 {
						p.objFIFO[j] = objFIFO
					}
				} else {
					p.objFIFO = append(p.objFIFO, objFIFO)
				}

				j++
			}

			p.fetcherObj = false
			p.objFetcherStep = 0
		}
		p.objFetcherStep++
		p.s.ScheduleEvent(scheduler.PPUFIFOTransfer, 1)
		return // obj fetch steals dots
	}

	// is the window requesting a piggyback?
	if p.winEnabled && !p.fetcherWin && p.winTriggerWy && p.lx >= int(p.b.Get(types.WX)-7) {
		p.fetcherWin = true
		p.fetcherX = 0
		p.bgFetcherStep = 0
		p.bgFIFO = FIFO{}
	}

	fetcherMode := BG
	if p.fetcherWin {
		fetcherMode = Window
	}

	switch p.bgFetcherStep {
	case GetTileID:
		p.fetcherTileNo = p.getTileID(fetcherMode)
	case GetTileRowLow:
		p.fetcherLow = p.getTileRow(fetcherMode, p.fetcherTileNo, false)
	case GetTileRowHigh:
		p.fetcherHigh = p.getTileRow(fetcherMode, p.fetcherTileNo, true)
		if p.fetcherTileAttr&types.Bit5 > 0 {
			p.fetcherLow = bits.Reverse8(p.fetcherLow)
			p.fetcherHigh = bits.Reverse8(p.fetcherHigh)
		}
	case PushPixels:
		// do we need to refill the FIFO?
		if len(p.bgFIFO) == 0 {

			for i := uint8(0x80); i > 0; i >>= 1 {
				entry := FIFOEntry{}
				if p.bgEnabled || p.b.IsGBCCart() && p.b.IsGBC() {
					if p.fetcherLow&i > 0 {
						entry.Color |= 1
					}
					if p.fetcherHigh&i > 0 {
						entry.Color |= 2
					}

					entry.Attributes = p.fetcherTileAttr
				}
				p.bgFIFO = append(p.bgFIFO, entry)
			}

			// reset fetcher state and advance x
			p.bgFetcherStep = -1
			p.fetcherX++
		} else {
			p.bgFetcherStep-- // wait until we can
		}
	}
	// advance fetcher step
	p.bgFetcherStep++

	// are there any pixels in the FIFO?
	if len(p.bgFIFO) > 0 {
		// handle offscreen pixels
		if p.lx < 0 {
			p.bgFIFO = p.bgFIFO[1:]
			p.lx++
		} else {
			bgPX := p.bgFIFO[0]
			var colors = p.ColourPalette[bgPX.Attributes&7][bgPX.Color]

			if len(p.objFIFO) > 0 {
				objPX := p.objFIFO[0]
				p.objFIFO = p.objFIFO[1:]

				if objPX.Color > 0 &&
					(bgPX.Color == 0 ||
						!(objPX.Attributes&types.Bit7 > 0 || (p.b.IsGBCCart() && p.b.IsGBC() && bgPX.Attributes&types.Bit7 > 0 && p.bgEnabled))) {
					colors = p.ColourSpritePalette[objPX.Palette][objPX.Color]
				}
			}

			// shift pixel from FIFO to screen
			p.PreparedFrame[p.ly][p.lx] = colors
			p.bgFIFO = p.bgFIFO[1:]

			p.lx++
			// have we filled the LCD yet?
			if p.lx == 160 {
				// schedule beginning of HBlank mode
				p.modeToInt = ModeHBlank
				p.statUpdate()
				p.modeToInt = ModeVRAM

				p.s.ScheduleEvent(scheduler.PPUVRAMTransfer, 4)
				return
			}
		}
	}

	p.s.ScheduleEvent(scheduler.PPUFIFOTransfer, 1)
}

// startVBlank is performed on the first cycle of each line 144 to 152, and
// performs the ModeVBlank period for the current line. The ModeVBlank period lasts
// until for 456 * 10 cycles, when the PPU enters Mode 2 (OAM search) on
// line 153 (PPU be like line 0, no line 153. you know, line 0, not the line 153 it's the next line :)).
func (p *PPU) startVBlank() {
	p.winTriggerWy = false
	// should we start line 153?
	if p.ly == 153 {
		p.startLine153()
		return
	}

	p.lyForComparison = 0xffff
	p.statUpdate()

	// set the LY register to current scanline
	p.b.Set(types.LY, p.ly)

	if p.ly == 144 {
		p.modeToInt = ModeOAM

		// trigger vblank interrupt
		if p.b.Model() != types.CGBABC && p.b.Model() != types.CGB0 {
			p.b.RaiseInterrupt(io.VBlankINT)
		}
	}
	p.statUpdate()

	p.s.ScheduleEvent(scheduler.PPUContinueVBlank, 4)
}

func (p *PPU) continueVBlank() {
	p.lyForComparison = uint16(p.ly)
	p.statUpdate()
	if p.ly == 144 {
		p.mode = ModeVBlank

		// trigger vblank interrupt
		if p.b.Model() == types.CGBABC || p.b.Model() == types.CGB0 {
			p.b.RaiseInterrupt(io.VBlankINT)
		}

		// entering vblank also triggers the OAM STAT interrupt if enabled
		if !p.statINT && p.status&0x20 != 0 {
			p.b.RaiseInterrupt(io.LCDINT)
		}
		p.modeToInt = ModeVBlank
		p.statUpdate()
	}
	p.ly++

	p.s.ScheduleEvent(scheduler.PPUStartVBlank, 452)
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
	p.ly, p.wly = 0, 0

	p.s.ScheduleEvent(scheduler.PPUStartOAMSearch, 444)
}

type FetcherMode int

const (
	BG FetcherMode = iota
	Window
	OBJ
)

// getTileID determines which background window/tile to fetch pixels from.
func (p *PPU) getTileID(mode FetcherMode) uint8 {
	address := uint16(0x1800)

	switch mode {
	case BG:
		address |= uint16(p.bgTileMap) << 10
		address |= uint16(p.ly+p.b.Get(types.SCY)) >> 3 << 5
		address |= (uint16(p.b.Get(types.SCX)>>3) + uint16(p.fetcherX)) & 0x1f
	case Window:
		address |= uint16(p.winTileMap) << 10
		address |= (uint16(p.wly) >> 3) << 5
		address |= uint16(p.fetcherX)
	}

	if p.b.IsGBCCart() && p.b.IsGBC() {
		p.fetcherTileAttr = p.b.GetVRAM(address, 1)
	}
	return p.b.GetVRAM(address, 0)
}

// getTileRow fetches one slice of the bitplane currently being read.
func (p *PPU) getTileRow(mode FetcherMode, id uint8, high bool) uint8 {
	address := uint16(0x0000)
	address |= uint16(id) << 4                      // Tile ID Offset
	address |= uint16(p.addressMode&^(id>>7)) << 12 // Negation of id.7 when LCDC.4 is set
	attr := p.fetcherTileAttr                       // Objects & CGB Only
	yPos := uint16(0)

	switch mode {
	case BG:
		yPos = uint16(p.ly+p.b.Get(types.SCY)) & 7
	case Window:
		yPos = uint16(p.wly & 7)
	case OBJ:
		yPos = (uint16(p.ly) - uint16(p.fetchingObj.y)) & 7
		attr = p.fetchingObj.attr
		address &^= 0x1000 // Clear Address Mode
	}

	if attr&types.Bit6 > 0 { // Y-Flip (Objects & CGB Only)
		yPos = ^yPos & 7
	}
	address |= yPos << 1 // Y-Pos Offset

	if high {
		address |= 1
	}

	return p.b.GetVRAM(address, attr&types.Bit3>>3)
}

// scroll is a helper function to determine the cycle offset of a scroll value
func scroll(value uint8) uint8 { return (value&7 + 3) &^ 3 }

func (p *PPU) statUpdate() {
	if !p.enabled {
		return
	}

	// get previous interrupt state
	prevInterruptLine := p.statINT

	// handle LY=LYC
	p.lycINT = p.lyForComparison != 0xffff && uint8(p.lyForComparison) == p.lyCompare
	p.status &^= types.Bit2
	if p.lycINT {
		p.status |= types.Bit2
	}

	// handle stat int
	p.statINT = (p.modeToInt == ModeHBlank && p.status&types.Bit3 != 0) ||
		(p.modeToInt == ModeVBlank && p.status&types.Bit4 != 0) ||
		(p.modeToInt == ModeOAM && p.status&types.Bit5 != 0) ||
		(p.lycINT && p.status&types.Bit6 != 0)

	// trigger interrupt if needed
	if p.statINT && !prevInterruptLine {
		p.b.RaiseInterrupt(io.LCDINT)
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

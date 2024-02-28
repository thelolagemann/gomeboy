package ppu

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
	"image"
	"image/color"
	"strconv"
	"strings"
	"unsafe"
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
	TileEntry
}

func (s Sprite) String() string {
	return fmt.Sprintf("X: %d Y: %d\n ID: %d VRAM: %d Palette: %d\n", s.x, s.y, s.id, s.vRAMBank, s.paletteNumber)
}

// A Tile has a size of 8x8 pixels, using a 2bpp format.
type Tile [16]uint8

// Draw draws the tile to the given image at the given position.
func (t Tile) Draw(img *image.RGBA, i int, i2 int, pal Palette) {
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			copy(img.Pix[((i2+y)*img.Stride)+((i+x)*4):], append(pal[(t[y]>>(7-x)&1)|(t[y+8]>>(7-x)&1)<<1][:], 0xff))
		}
	}
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
	mode, modeToInt, currentLine, status uint8
	lyCompare, windowInternal            uint8

	lyForComparison uint16
	lycINT, statINT bool
	scanlineInfo    [ScreenWidth]uint8 // bit 3 - priority, bit 2-0 colourNumber

	Sprites  [40]Sprite           // 40 sprite attributes from OAM
	TileData [2][384]Tile         // 384 tiles, 8x8 pixels each (double in CGB mode)
	TileMaps [2][32][32]TileEntry // 32x32 tiles, 8x8 pixels each

	cRAM                         [128]uint8 // 64 bytes BG+OBJ
	bcpsIncrement, ocpsIncrement bool
	bcpsIndex, ocpsIndex         uint8

	backgroundDirty                               bool
	backgroundLineRendered, backgroundLineChanged [ScreenHeight]bool
	dirtyScanlines, spriteScanlines               [ScreenHeight]bool              // used to determine when to re-render background
	spriteScanlinesColumn                         [ScreenHeight][ScreenWidth]bool // used for debug render views (todo remove and calculate on debug draw)

	dirtiedLog [65536]dirtyEvent
	lastDirty  uint16

	// external components
	b *io.Bus
	s *scheduler.Scheduler

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
}

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
			p.currentLine, p.mode = 0, 0

			p.b.Unlock(io.OAM)
			p.b.Unlock(io.VRAM)
		} else if !p.enabled && v&types.Bit7 != 0 {
			p.enabled = true
			// reset LYC to compare against and clear coincidence flag
			p.lyForComparison, p.currentLine = 0, 0
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
		if b.IsBooting() {
			p.currentLine = v
			return v
		}
		// any write to LY resets to 0
		p.currentLine = 0
		return 0
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
		if v == b.Get(types.BGP) || p.b.IsGBCCart() && p.b.IsGBC() {
			return v
		}
		p.dirtyBackground(bgp)
		p.ColourPalette[0] = p.BGColourisationPalette.remap(v)

		return v
	})
	b.ReserveAddress(types.OBP0, func(v byte) byte {
		if v == b.Get(types.OBP0) || p.b.IsGBCCart() && p.b.IsGBC() {
			return v
		}
		p.dirtyBackground(obp0)
		p.ColourSpritePalette[0] = p.OBJ0ColourisationPalette.remap(v)

		return v
	})
	b.ReserveAddress(types.OBP1, func(v byte) byte {
		if v == b.Get(types.OBP1) || p.b.IsGBCCart() && p.b.IsGBC() {
			return v
		}
		p.dirtyBackground(obp1)
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
		p.TileMaps = [2][32][32]TileEntry{}
	})

	// setup CGB only registers
	b.RegisterGBCHandler(func() {
		b.ReserveAddress(types.BCPS, func(v byte) byte {
			p.bcpsIndex = v & 0x3F
			p.bcpsIncrement = v&types.Bit7 > 0
			p.dirtyBackground(bcps)
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
				p.dirtyBackground(bcpd)

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
			p.dirtyBackground(ocps)
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
				p.dirtyBackground(ocpd)

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
	s.RegisterEvent(scheduler.PPUHBlankInterrupt, func() {
		p.modeToInt = ModeHBlank
		p.statUpdate()
		p.modeToInt = ModeVRAM

		p.s.ScheduleEvent(scheduler.PPUVRAMTransfer, 4)
	})

	// setup line changed
	for i := 0; i < len(p.backgroundLineChanged); i++ {
		p.backgroundLineChanged[i] = true
	}

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
	p.currentLine++
	p.modeToInt = ModeOAM

	// if we are on line 144, we are entering ModeVBlank
	if p.currentLine == 144 {
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
	p.mode, p.modeToInt = ModeHBlank, ModeHBlank

	p.b.Unlock(io.OAM)
	p.b.Unlock(io.VRAM)

	p.s.ScheduleEvent(scheduler.PPUStartHBlank, 4)
}

func (p *PPU) startHBlank() {
	p.renderScanline()
	p.statUpdate()

	if p.b.IsGBCCart() {
		p.b.HandleHDMA()
	}

	// schedule end of ModeHBlank
	p.s.ScheduleEvent(scheduler.PPUHBlank, uint64(196-scroll(p.b.Get(types.SCX))))
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

	p.mode = ModeHBlank
	// OAM STAT int occurs 1-M cycle before STAT changes, except on line 0
	if p.currentLine != 0 {
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
	p.lyForComparison = uint16(p.currentLine)
	p.statUpdate()

	p.modeToInt = 255
	p.statUpdate()

	p.b.WLock(io.OAM)

	p.s.ScheduleEvent(scheduler.PPUPrepareEndOAMSearch, 76)
}

// endOAM is performed 80 cycles after startOAM, and performs the
// rest of the OAM search.
func (p *PPU) endOAM() {
	p.mode, p.modeToInt = ModeVRAM, ModeVRAM
	p.statUpdate()

	p.b.Lock(io.OAM | io.VRAM)

	// schedule end of VRAM search
	p.s.ScheduleEvent(scheduler.PPUHBlankInterrupt, uint64(168+scroll(p.b.Get(types.SCX))))
}

func (p *PPU) WriteCorruptionOAM() {
	/*fmt.Println("corrupting ModeOAM")
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
	a := uint16(p.ModeOAM.data[row*4]) | uint16(p.ModeOAM.data[row*4+1])<<8
	b := uint16(p.ModeOAM.data[row*4-8]) | uint16(p.ModeOAM.data[row*4-7])<<8
	c := uint16(p.ModeOAM.data[row*4-6]) | uint16(p.ModeOAM.data[row*4-5])<<8

	// perform the bitwise glitch
	newValue := bitwiseGlitch(a, b, c)

	// replace the first word of the current row with the new value
	p.ModeOAM.data[row*4] = byte(newValue)
	p.ModeOAM.data[row*4+1] = byte(newValue >> 8)

	// replace the last 3 words of the row from the preceding row
	p.ModeOAM.data[row*4-6] = p.ModeOAM.data[row*4-2]
	p.ModeOAM.data[row*4-5] = p.ModeOAM.data[row*4-1]
	p.ModeOAM.data[row*4-4] = p.ModeOAM.data[row*4]
	p.ModeOAM.data[row*4-3] = p.ModeOAM.data[row*4+1]
	p.ModeOAM.data[row*4-2] = p.ModeOAM.data[row*4+2]
	p.ModeOAM.data[row*4-1] = p.ModeOAM.data[row*4+3]

	//panic(fmt.Sprintf("OAM corruption: row %d %d cycles until end of OAM search %s", row, cyclesUntilEndOAM, p.s.String()))*/
}

func bitwiseGlitch(a, b, c uint16) uint16 {
	return ((a ^ c) & (b ^ c)) ^ c
}

// startVBlank is performed on the first cycle of each line 144 to 152, and
// performs the ModeVBlank period for the current line. The ModeVBlank period lasts
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
	p.lyForComparison = uint16(p.currentLine)
	p.statUpdate()
	if p.currentLine == 144 {
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
	p.currentLine++

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
	p.currentLine, p.windowInternal = 0, 0

	p.s.ScheduleEvent(scheduler.PPUStartOAMSearch, 444)
}

func (p *PPU) DumpTileMaps(tileMap1, tileMap2 *image.RGBA, gap int) {
	// draw tilemap (0x9800 - 0x9BFF)
	for i := uint8(0); i < 32; i++ {
		for j := uint8(0); j < 32; j++ {
			tileEntry := p.TileMaps[0][j][i]
			// get tile data
			tile := p.TileData[tileEntry.vRAMBank][getID(tileEntry.id, p.addressMode)]
			tile.Draw(tileMap1, int(i)*(8+gap), int(j)*(8+gap), p.ColourPalette[0])

			tileEntry = p.TileMaps[1][j][i]

			// get tile data
			tile = p.TileData[tileEntry.vRAMBank][getID(tileEntry.id, p.addressMode)]
			tile.Draw(tileMap2, int(i)*(8+gap), int(j)*(8+gap), p.ColourPalette[0])
		}
	}
}

func (p *PPU) writeOAM(b [160]byte) {
	for i, value := range b {
		// get the current sprite and x, y pos
		s := &p.Sprites[i>>2]
		oldY, oldX := s.y, s.x

		switch i & 3 {
		case 0:
			s.y = value

			// was the sprite visible before?
			if oldY < ScreenHeight && oldX < ScreenWidth {
				// we need to remove the positions that the sprite was visible on before
				for i := oldY; i < oldY+8 && i < ScreenHeight; i++ {
					p.spriteScanlines[i] = false
					p.dirtyScanlines[i] = true
					for j := oldX; j < oldX+8 && j < ScreenWidth; j++ {
						p.spriteScanlinesColumn[i][j] = false
					}
				}
			}

			// is the sprite visible now?
			newYPos := s.y
			if newYPos > ScreenHeight || oldX > ScreenHeight {
				continue // sprite is not visible
			}

			// we need to add the positions that the sprite is now visible on
			for i := newYPos; i < newYPos+8 && i < ScreenHeight; i++ {
				p.spriteScanlines[i] = true
				for j := oldX; j < oldX+8 && j < ScreenWidth; j++ {
					p.spriteScanlinesColumn[i][j] = true
				}
			}
		case 1:
			s.x = value
			// was the sprite visible before?
			if oldY < ScreenHeight && oldX < ScreenWidth {
				// we need to remove the positions that the sprite was visible on
				for i := oldY; i < oldY+8 && i < ScreenHeight; i++ {
					p.spriteScanlines[i] = false
					p.dirtyScanlines[i] = true
					for j := oldX; j < oldX+8 && j < ScreenWidth; j++ {
						p.spriteScanlinesColumn[i][j] = false
					}
				}
			}

			// is the sprite visible now?
			newXPos := s.x
			if newXPos > ScreenWidth || oldY > ScreenHeight {
				continue // sprite is not visible
			}

			// we need to add the positions that the sprite is now visible on
			for i := oldY; i < oldY+8 && i < ScreenHeight; i++ {
				p.spriteScanlines[i] = true
				for j := newXPos; j < newXPos+8 && j < ScreenWidth; j++ {
					p.spriteScanlinesColumn[i][j] = true
				}
			}
		case 2:
			s.id = value
		case 3:
			s.priority = value&types.Bit7 == 0
			s.yFlip = value&types.Bit6 != 0
			s.xFlip = value&types.Bit5 != 0
			s.vRAMBank = (value >> 3) & 1

			if p.b.IsGBCCart() {
				s.paletteNumber = value & 0x7
			} else {
				s.paletteNumber = value & types.Bit4 >> 4
			}
		}
	}
}

func (p *PPU) writeVRAM(changes []io.VRAMChange) {
	for _, c := range changes {
		address, bank, value := c.Address, c.Bank, c.Value
		address &= 0x1FFF

		switch {
		case address <= 0x17FF:
			p.TileData[bank][address>>4][(address>>1)&0x7+((address&1)*8)] = value
			p.dirtyBackground(tile)
		case address <= 0x1FFF:
			y, x, id := (address>>5)&0x1f, address&0x1f, (address>>10)&1
			switch bank {
			case 0:
				p.TileMaps[id][y][x].id = value
				p.dirtyBackground(tileMap)
			case 1:
				// update the tilemap
				t := &p.TileMaps[id][y][x]
				t.priority = value&0x80 != 0
				t.yFlip = value&types.Bit6 != 0
				t.xFlip = value&types.Bit5 != 0
				t.paletteNumber = value & 0b111
				t.vRAMBank = value >> 3 & 0x1
				p.dirtyBackground(tileAttr)
			}
		}
	}

}

// colorNumber is a helper function to determine the colour number of the given index
func colorNumber(b1, b2, index uint8, xFlip bool) uint8 {
	shift := 7 - index
	if xFlip {
		shift = index
	}
	return (b1 >> shift & 0x1) | ((b2 >> shift & 0x1) << 1)
}

// getID is a helper function to determine the ID of a tile according to the addressing mode
func getID(id, mode uint8) uint16 { return uint16(id) + uint16(mode&^(id>>7))<<8 }

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

func (p *PPU) renderScanline() {
	currentScanline := p.b.Get(types.LY)
	if p.b.VRAMChanged() {
		p.b.VRAMCatchup(p.writeVRAM)
	}
	if (!p.backgroundLineRendered[currentScanline] || p.dirtyScanlines[currentScanline] || p.backgroundDirty) && (p.bgEnabled || p.b.IsGBCCart()) {
		p.renderBackgroundScanline()
	}

	if p.winEnabled && !p.Debug.WindowDisabled {
		p.renderWindowScanline()
	}

	if p.objEnabled && !p.Debug.SpritesDisabled {
		p.renderSpritesScanline(currentScanline)
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

func (p *PPU) renderWindowScanline() {
	// do nothing if window is out of bounds
	if p.b.Get(types.LY) < p.b.Get(types.WY) ||
		p.b.Get(types.WX) > ScreenWidth ||
		p.b.Get(types.WY) > ScreenHeight {
		return
	}

	// get the initial x pos and pixel pos
	xPos, xPixelPos := uint8(0), uint8(0)

	// get the first tile entry
	tileEntry, b1, b2, pal := p.getTile(xPos, p.windowInternal, p.winTileMap)

	scanline := &p.PreparedFrame[p.currentLine]

	for i := int(p.b.Get(types.WX)) - 7; i < ScreenWidth; i++ {
		if i < 0 {
			continue
		}
		colorNum := colorNumber(b1, b2, xPixelPos, tileEntry.xFlip)
		scanline[i] = pal[colorNum]

		p.scanlineInfo[i] = colorNum
		if tileEntry.priority {
			p.scanlineInfo[i] |= 0b100
		}

		// did we just finish a tile?
		if xPixelPos++; xPixelPos == 8 {
			xPos += 8
			// get the next tile entry
			tileEntry, b1, b2, pal = p.getTile(xPos, p.windowInternal, p.winTileMap)

			xPixelPos = 0
		}
	}

	p.dirtyScanlines[p.currentLine] = true
	p.windowInternal++
}

// getTile returns the TileEntry, and palette for the
// tile at the given xPos and yPos on the given tile map.
func (p *PPU) getTile(xPos, yPos, tileMap uint8) (tileEntry TileEntry, b1, b2 uint8, pal Palette) {
	tileEntry = p.TileMaps[tileMap][yPos>>3][xPos>>3]
	tileData := p.TileData[tileEntry.vRAMBank][getID(tileEntry.id, p.addressMode)]

	yPixelPos := yPos & 7
	if tileEntry.yFlip {
		yPixelPos = 7 - yPixelPos
	}
	b1, b2 = tileData[yPixelPos], tileData[yPixelPos+8]
	pal = p.ColourPalette[tileEntry.paletteNumber]

	return
}

func (p *PPU) renderBackgroundScanline() {
	// get the initial x and y pos
	yPos := p.b.Get(types.LY) + p.b.Get(types.SCY)
	xPos := p.b.Get(types.SCX)
	xPixelPos := xPos & 7

	// get the first tile entry
	tileEntry, b1, b2, pal := p.getTile(xPos, yPos, p.bgTileMap)

	var scanline [ScreenWidth][3]uint8

	for i := uint8(0); i < ScreenWidth; i++ {
		// get color number of pixel
		colorNum := colorNumber(b1, b2, xPixelPos, tileEntry.xFlip)
		// set scanline using unsafe to copy 4 bytes at a time
		*(*uint32)(unsafe.Pointer(&scanline[i])) = *(*uint32)(unsafe.Pointer(&pal[colorNum]))

		p.scanlineInfo[i] = colorNum
		if tileEntry.priority {
			p.scanlineInfo[i] |= 0b100
		}

		xPixelPos++
		xPos++

		// did we just finish a tile?
		if xPixelPos == 8 {
			tileEntry, b1, b2, pal = p.getTile(xPos, yPos, p.bgTileMap)
			xPixelPos = 0
		}
	}

	p.backgroundLineRendered[p.b.Get(types.LY)] = true
	p.dirtyScanlines[p.b.Get(types.LY)] = false

	// update scanline in frame
	p.PreparedFrame[p.b.Get(types.LY)] = scanline
}

func (p *PPU) renderSpritesScanline(scanline uint8) {
	if p.b.OAMChanged() {
		p.b.OAMCatchup(p.writeOAM)
	}

	spriteXPerScreen := [ScreenWidth]uint8{}
	spriteCount := 0 // number of sprites on the current scanline (max 10)

	for _, sprite := range p.Sprites {
		adjustedScanline := scanline + 16
		if sprite.y > adjustedScanline || sprite.y+p.objSize <= adjustedScanline {
			continue
		}

		yPixelPos := scanline - sprite.y
		if sprite.yFlip {
			yPixelPos = p.objSize - yPixelPos - 1
		}
		yPixelPos &= 7
		tileID := sprite.id
		if p.objSize == 16 {
			if (adjustedScanline-sprite.y < 8 && sprite.yFlip) || (adjustedScanline-sprite.y >= 8 && !sprite.yFlip) {
				tileID |= 0x01
			} else {
				tileID &= 0xFE
			}
		}

		// get the 2 bytes of data that make up the row of the tile
		b1 := p.TileData[sprite.vRAMBank][tileID][yPixelPos]
		b2 := p.TileData[sprite.vRAMBank][tileID][yPixelPos+8]

		pal := p.ColourSpritePalette[sprite.paletteNumber]

		for x := uint8(0); x < 8 && (sprite.x)+x < ScreenWidth+8; x++ {
			colourNumber := colorNumber(b1, b2, x, sprite.xFlip)
			pixelPos := (sprite.x - 8) + x
			if pixelPos >= ScreenWidth {
				continue
			}

			// skip if the color is transparent
			if colourNumber == 0 ||
				// handle bg/win tile -> sprite priority
				(!p.b.IsGBCCart() || p.bgEnabled) && !(sprite.priority && p.scanlineInfo[pixelPos]&4 == 0) && p.scanlineInfo[pixelPos]&3 != 0 ||
				// skip if sprite is already drawn on this pixel
				spriteXPerScreen[pixelPos] != 0 && (p.b.IsGBCCart() || spriteXPerScreen[pixelPos] <= sprite.x) {
				continue
			}

			// did rendering this sprite change the background
			if p.PreparedFrame[scanline][pixelPos] != pal[colourNumber] {
				p.backgroundLineRendered[scanline] = false
				p.PreparedFrame[scanline][pixelPos] = pal[colourNumber]
			}

			// mark the pixel as occupied
			spriteXPerScreen[pixelPos] = sprite.x
		}

		if spriteCount++; spriteCount == 10 {
			break
		}
	}
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
			} else if p.spriteScanlinesColumn[y][x] {
				img.Set(x, y, color.RGBA{0, 255, 0, 128})
			} else if p.spriteScanlines[y] {
				img.Set(x, y, color.RGBA{0, 0, 255, 128})
			}
		}
	}
}

func (p *PPU) DrawSprite(img *image.RGBA, spr Sprite) {
	// get the tile that sprite is using and call tile.Draw
	p.TileData[spr.vRAMBank][spr.id].Draw(img, 0, 0, p.ColourSpritePalette[spr.paletteNumber])

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

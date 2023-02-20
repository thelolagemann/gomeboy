// Package ppu provides a programmable pixel unit for the DMG and CGB.
package ppu

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/ppu/background"
	"github.com/thelolagemann/go-gameboy/internal/ppu/lcd"
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"github.com/thelolagemann/go-gameboy/internal/ram"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"image"
	"log"
	"os"
)

const (
	// ScreenWidth is the width of the screen in pixels.
	ScreenWidth = 160
	// ScreenHeight is the height of the screen in pixels.
	ScreenHeight = 144
)

type PPU struct {
	Debug struct {
		SpritesDisabled    bool
		BackgroundDisabled bool
		WindowDisabled     bool
	}

	*background.Background
	*lcd.Controller
	*lcd.Status
	CurrentScanline uint8
	LYCompare       uint8
	SpritePalettes  [2]palette.Palette

	WindowX         uint8
	WindowY         uint8
	WindowYInternal uint8

	// CGB registers
	vRAMBank uint8

	oam                        *OAM
	vRAM                       [2]*ram.RAM // Second bank only exists on CGB
	ColourPalette              *palette.CGBPalette
	ColourSpritePalette        *palette.CGBPalette
	compatibilityPalette       *palette.CGBPalette // TODO use more efficient data structure
	compatibilitySpritePalette *palette.CGBPalette

	tileData [2][384]*Tile // 384 tiles, 8x8 pixels each (double in CGB mode)
	tileMaps [2]TileMap    // 32x32 tiles, 8x8 pixels each

	irq *interrupts.Service

	PreparedFrame [ScreenHeight][ScreenWidth][3]uint8
	scanlineHit   [ScreenHeight]bool
	scanlineQueue [ScreenHeight]*RenderOutput

	scanlineData [ScanlineSize]uint8
	spriteData   [40]uint8 // 10 sprites, 4 bytes each. byte 1 and 2 are tile data, byte 3 is attributes, byte 4 is x position

	currentCycle       uint16
	bus                *mmu.MMU
	screenCleared      bool
	statInterruptDelay bool
	cleared            bool
	offClock           uint32
	refreshScreen      bool
	DMA                *DMA
	delayedTick        bool

	renderer    *Renderer
	rendererCGB *RendererCGB
}

func (p *PPU) init() {
	// setup components
	p.Controller = lcd.NewController(func(writeFn func()) {
		wasOn := p.Enabled
		writeFn()

		// if the screen was turned off, clear the screen
		if wasOn && !p.Enabled {
			// the screen should not be turned off unless in vblank
			if p.Mode != lcd.VBlank {
				panic("PPU: Screen was turned off while not in VBlank")
			}

			// clear the screen
			p.renderBlank()

			// enter hblank
			p.Mode = lcd.HBlank

			// reset the scanline
			p.CurrentScanline = 0
		} else if !wasOn && p.Enabled {
			p.checkLYC()
			p.checkStatInterrupts(false)
			// if the screen was turned on, reset the clock
			p.currentCycle = 4
			p.delayedTick = true
		}
	})
	p.Status = lcd.NewStatus(func(writeFn func()) {
		writeFn()
		if p.Enabled {
			p.checkStatInterrupts(false)
		}
	})

	// setup registers
	types.RegisterHardware(
		types.LY,
		func(v uint8) {
			// any write to LY resets the value to 0
			p.CurrentScanline = 0
		},
		func() uint8 {
			return p.CurrentScanline
		},
	)
	types.RegisterHardware(
		types.LYC,
		func(v uint8) {
			p.LYCompare = v

			if p.Enabled {
				p.checkLYC()
				p.checkStatInterrupts(false)
			}
		},
		func() uint8 {
			return p.LYCompare
		},
	)
	types.RegisterHardware(
		types.OBP0,
		func(v uint8) {
			p.SpritePalettes[0] = palette.ByteToPalette(v)
			if p.bus.IsGBCCompat() {
				for i := 0; i < 4; i++ {
					p.compatibilitySpritePalette.Palettes[0][i] = p.compatibilitySpritePalette.GetColour(7, p.SpritePalettes[0][i])
				}
			}
		},
		func() uint8 {
			return p.SpritePalettes[0].ToByte()
		},
	)
	types.RegisterHardware(
		types.OBP1,
		func(v uint8) {
			p.SpritePalettes[1] = palette.ByteToPalette(v)
			if p.bus.IsGBCCompat() {
				// reorganize the colour palette based on the sprite palette
				for i := 0; i < 4; i++ {
					p.compatibilitySpritePalette.Palettes[1][i] = p.compatibilitySpritePalette.GetColour(6, p.SpritePalettes[1][i])
				}
			}
		},
		func() uint8 {
			return p.SpritePalettes[1].ToByte()
		},
	)
	types.RegisterHardware(
		types.WX,
		func(v uint8) {
			p.WindowX = v
		},
		func() uint8 {
			return p.WindowX
		},
	)
	types.RegisterHardware(
		types.WY,
		func(v uint8) {
			p.WindowY = v
		},
		func() uint8 {
			return p.WindowY
		},
	)

	// CGB registers

	types.RegisterHardware(
		types.VBK,
		func(v uint8) {
			if p.bus.IsGBCCompat() {
				p.vRAMBank = v & types.Bit0
			}
		},
		func() uint8 {
			if p.bus.IsGBCCompat() {
				return p.vRAMBank
			}
			return 0xFF
		},
	)
	types.RegisterHardware(
		types.BCPS,
		func(v uint8) {
			if p.bus.IsGBCCompat() {
				p.ColourPalette.SetIndex(v)
			}
		},
		func() uint8 {
			if p.bus.IsGBCCompat() {
				return p.ColourPalette.GetIndex()
			}
			return 0xFF
		},
	)
	types.RegisterHardware(
		types.BCPD,
		func(v uint8) {
			if p.bus.IsGBCCompat() && p.colorPaletteUnlocked() {
				p.ColourPalette.Write(v)
			}
		},
		func() uint8 {
			if p.bus.IsGBCCompat() && p.colorPaletteUnlocked() {
				return p.ColourPalette.Read()
			}
			return 0xFF
		},
	)
	types.RegisterHardware(
		types.OCPS,
		func(v uint8) {
			if p.bus.IsGBCCompat() {
				p.ColourSpritePalette.SetIndex(v)
			}
		},
		func() uint8 {
			if p.bus.IsGBCCompat() {
				return p.ColourSpritePalette.GetIndex()
			}
			return 0xFF
		},
	)
	types.RegisterHardware(
		types.OCPD,
		func(v uint8) {
			if p.bus.IsGBCCompat() && p.colorPaletteUnlocked() {
				p.ColourSpritePalette.Write(v)
			}
		},
		func() uint8 {
			if p.bus.IsGBCCompat() && p.colorPaletteUnlocked() {
				return p.ColourSpritePalette.Read()
			}
			return 0xFF
		},
	)

	// initialize tile data
	for i := 0; i < 2; i++ {
		for j := 0; j < len(p.tileData[0]); j++ {
			p.tileData[i][j] = &Tile{}
		}
	}

	// initialize tile map
	for i := 0; i < 2; i++ {
		for j := 0; j < len(p.tileMaps); j++ {
			p.tileMaps[i] = NewTileMap()
		}
	}

	p.ColourPalette = palette.NewCGBPallette()
	p.ColourSpritePalette = palette.NewCGBPallette()
	p.compatibilityPalette = palette.NewCGBPallette()
	p.compatibilitySpritePalette = palette.NewCGBPallette()
}

// TODO pass channel to send frame to
func (p *PPU) StartRendering() {
	output := make(chan *RenderOutput, ScreenHeight)
	// setup renderer
	if p.bus.IsGBCCompat() {
		renderJobs := make(chan RenderJobCGB, 20)
		p.rendererCGB = NewRendererCGB(renderJobs, output, p.bus.IsGBC())
		p.bus.HDMA.AttachVRAM(p.WriteVRAM)

		p.vRAM[1] = ram.NewRAM(0x2000)
	} else {
		renderJobs := make(chan RenderJob, 20)
		p.renderer = NewRenderer(renderJobs, output)
	}

	// create goroutine to handle rendering
	go func() {
		var renderOutput *RenderOutput
		for i := 0; i < ScreenHeight; i++ { // the last 8 scanlines are rendered as they are needed to avoid flickering
			renderOutput = <-output
			if p.scanlineHit[renderOutput.Line] {
				p.scanlineQueue[renderOutput.Line] = renderOutput
			} else {
				p.PreparedFrame[renderOutput.Line] = renderOutput.Scanline
				p.scanlineHit[renderOutput.Line] = true
			}

			if i == ScreenHeight-1 {
				// the last scanline has been rendered, so the frame is ready
				i = 0

				// reset scanline hit
				p.scanlineHit = [ScreenHeight]bool{}

				// process any queued scanlines TODO
				p.scanlineQueue = [ScreenHeight]*RenderOutput{}
			}
		}
	}()
}

func New(mmu *mmu.MMU, irq *interrupts.Service) *PPU {
	oam := NewOAM()
	p := &PPU{
		Background: background.NewBackground(),
		tileData:   [2][384]*Tile{},

		bus: mmu,
		irq: irq,
		oam: oam,
		vRAM: [2]*ram.RAM{
			ram.NewRAM(8192),
		},
		DMA: NewDMA(mmu, oam),
	}

	p.init()
	return p
}

// TODO save compatibility palette
// - load game with boot ROM enabled
// - save colour palette to file (bgp = index 0 of colour palette, obp1 = index 0 of sprite palette, obp2 = index 1 of sprite palette)
// - encoded filename as hash of palette

func (p *PPU) LoadCompatibilityPalette() {
	if p.bus.BootROM != nil {
		return // don't load compatibility palette if boot ROM is enabled (as the boot ROM will setup the palette)
	}
	hash := p.bus.Cart.Header().TitleChecksum()
	entryWord := uint16(hash) << 8
	if p.bus.Cart.Header().Title != "" {
		entryWord |= uint16(p.bus.Cart.Header().Title[3])
	}
	paletteEntry, ok := palette.GetCompatibilityPaletteEntry(entryWord)

	if !ok {
		p.bus.Log.Infof("No compatibility palette found for hash %02X", hash)
		paletteEntry = palette.CompatibilityPalettes[0x1C][0x03]
	}
	p.bus.Log.Infof("Loaded compatibility palette %02X", hash)

	// copy the loaded compatibility palette into the palette (so the palette can be re-organised)
	p.compatibilityPalette.Palettes[0][0] = paletteEntry.BG[0]
	p.compatibilityPalette.Palettes[0][1] = paletteEntry.BG[1]
	p.compatibilityPalette.Palettes[0][2] = paletteEntry.BG[2]
	p.compatibilityPalette.Palettes[0][3] = paletteEntry.BG[3]

	p.compatibilitySpritePalette.Palettes[7][0] = paletteEntry.OBJ0[0]
	p.compatibilitySpritePalette.Palettes[7][1] = paletteEntry.OBJ0[1]
	p.compatibilitySpritePalette.Palettes[7][2] = paletteEntry.OBJ0[2]
	p.compatibilitySpritePalette.Palettes[7][3] = paletteEntry.OBJ0[3]

	p.compatibilitySpritePalette.Palettes[6][0] = paletteEntry.OBJ1[0]
	p.compatibilitySpritePalette.Palettes[6][1] = paletteEntry.OBJ1[1]
	p.compatibilitySpritePalette.Palettes[6][2] = paletteEntry.OBJ1[2]
	p.compatibilitySpritePalette.Palettes[6][3] = paletteEntry.OBJ1[3]
}

func (p *PPU) SaveCompatibilityPalette() {
	compatPal := palette.CompatibilityPalette{
		BGP:  p.ColourPalette.Palettes[0],
		OBP0: p.ColourSpritePalette.Palettes[0],
		OBP1: p.ColourSpritePalette.Palettes[1],
	}

	// create hash of palette
	hash := sha256.New()
	for _, c := range compatPal.BGP {
		hash.Write([]byte{c[0], c[1], c[2]})
	}
	for _, c := range compatPal.OBP0 {
		hash.Write([]byte{c[0], c[1], c[2]})
	}
	for _, c := range compatPal.OBP1 {
		hash.Write([]byte{c[0], c[1], c[2]})
	}

	// create file
	file, err := os.Create(fmt.Sprintf("compatibility_palette_%x", hash.Sum(nil)))
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// encode palette
	enc := json.NewEncoder(file)

	// write palette to file
	err = enc.Encode(compatPal)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Compatibility palette saved to file: compatibility_palette_%x", hash.Sum(nil))
}

func (p *PPU) Read(address uint16) uint8 {
	// read from VRAM
	if address >= 0x8000 && address <= 0x9FFF {
		if p.vramUnlocked() {
			return p.vRAM[p.vRAMBank].Read(address - 0x8000)
		} else {
			return 0xFF
		}
	}

	// read from OAM
	if address >= 0xFE00 && address <= 0xFE9F {
		if p.oamUnlocked() && !p.DMA.IsTransferring() {
			return p.oam.Read(address - 0xFE00)
		}
		return 0xff
	}

	// illegal read
	panic(fmt.Sprintf("PPU: Read from invalid address: %X", address))
}

func (p *PPU) vramUnlocked() bool {
	return p.Mode != lcd.VRAM
}

func (p *PPU) oamUnlocked() bool {
	return p.Mode != lcd.OAM && p.Mode != lcd.VRAM
}

func (p *PPU) DumpTileMap() image.Image {
	var img *image.RGBA
	img = image.NewRGBA(image.Rect(0, 0, 256, 512))

	// draw tilemap (0x9800 - 0x9BFF)
	for i := uint8(0); i < 32; i++ {
		for j := uint8(0); j < 32; j++ {
			tileEntry := p.calculateTileID(j, i, 0)
			// get tile data
			tile := p.tileData[tileEntry.Attributes.VRAMBank][tileEntry.GetID(p.UsingSignedTileData())]
			tile.Draw(img, int(i*8), int(j*8))
		}
	}

	// draw tilemap (0x9C00 - 0x9FFF)
	for i := uint8(0); i < 32; i++ {
		for j := uint8(0); j < 32; j++ {
			tileEntry := p.calculateTileID(j, i, 1)

			// get tile data
			tile := p.tileData[tileEntry.Attributes.VRAMBank][tileEntry.GetID(p.UsingSignedTileData())]
			tile.Draw(img, int(i)*8, int(j)*8+256)
		}
	}

	return img
}

func (p *PPU) DumpTiledata() image.Image {
	// 3 tilesets of 128 tiles each = 384 tiles total (CGB doubles everything)
	// 1 tile = 8x8 pixels
	// 384 * 64 = 24576 pixels total
	// 256 * 96 = 24576 pixels total
	// CGB = 768 * 64 = 49152 pixels total
	// CGB = 512 * 96 = 49152 pixels total
	var img *image.RGBA
	if p.bus.IsGBCCompat() {
		img = image.NewRGBA(image.Rect(0, 0, 256, 192))
	} else {
		img = image.NewRGBA(image.Rect(0, 0, 256, 96))
	}

	for i, tile := range p.tileData[0] {
		// calculate the x and y position of the tile
		x := (i % 32) * 8
		y := (i / 32) * 8

		tile.Draw(img, x, y)
	}

	if p.bus.IsGBCCompat() {
		for i, tile := range p.tileData[1] {
			// calculate the x and y position of the tile
			x := (i % 32) * 8
			y := (i/32)*8 + 96

			tile.Draw(img, x, y)
		}
	}

	return img
}

func (p *PPU) colorPaletteUnlocked() bool {
	return p.Mode != lcd.VRAM
}

func (p *PPU) WriteVRAM(address uint16, value uint8) {
	// is the VRAM currently locked?
	// FIXME: Boot ROM logo appears garbled when this is enabled (why? otherwise it's fine)
	if !p.vramUnlocked() {
		return
	}

	// write to the current VRAM bank
	p.vRAM[p.vRAMBank].Write(address, value)

	// are we writing to the tile data?
	if address <= 0x17FF {
		p.UpdateTile(address, value)
		// update the tile data
	} else if address <= 0x1FFF {
		if p.vRAMBank == 0 {
			// which offset are we writing to?
			if address >= 0x1800 && address <= 0x1BFF {
				// tilemap 0
				p.UpdateTileMap(address, 0)
			}
			if address >= 0x1C00 && address <= 0x1FFF {
				// tilemap 1
				p.UpdateTileMap(address, 1)
			}
		}
		if p.vRAMBank == 1 {
			// update the tile attributes
			if address >= 0x1800 && address <= 0x1BFF {
				// tilemap 0
				p.UpdateTileAttributes(address, 0, value)
			}
			if address >= 0x1C00 && address <= 0x1FFF {
				// tilemap 1
				p.UpdateTileAttributes(address, 1, value)
			}
		}
	}
}

func (p *PPU) Write(address uint16, value uint8) {
	// VRAM (0x8000 - 0x9FFF)
	if address >= 0x8000 && address <= 0x9FFF {
		p.WriteVRAM(address-0x8000, value)
		return
	}
	// OAM (0xFE00 - 0xFE9F)
	if address >= 0xFE00 && address <= 0xFE9F {
		if p.oamUnlocked() && !p.DMA.IsTransferring() {
			p.oam.Write(address-0xFE00, value)
		}
		return
	}

	// illegal writes
	panic(fmt.Sprintf("ppu: illegal write to address %04X", address))
}

// UpdateTile updates the tile at the given address
func (p *PPU) UpdateTile(address uint16, value uint8) {
	// get the tile address
	index := address & 0x1FFE // only the lower 13 bits are used

	// get the tileID
	tileID := index >> 4 // divide by 16

	// get the tile row
	row := (address >> 1) & 0x7

	// set the tile data
	p.tileData[p.vRAMBank][tileID][row][address%2] = value
}

func (p *PPU) UpdateTileMap(address uint16, tilemapIndex uint8) {
	// determine the y and x position
	y := (address / 32) & 0x1F
	x := address & 0x1F

	// update the tilemap
	p.tileMaps[tilemapIndex][y][x].TileID = p.vRAM[0].Read(address)
}

func (p *PPU) UpdateTileAttributes(index uint16, tilemapIndex uint8, value uint8) {
	// panic(fmt.Sprintf("updating tile %x with %b", index, value))
	// determine the y and x position
	y := (index / 32) & 0x1F
	x := index & 0x1F

	// update the tilemap
	p.tileMaps[tilemapIndex][y][x].Attributes.Write(value)
}

// checkLYC checks if the LYC interrupt should be triggered.
func (p *PPU) checkLYC() {
	if p.CurrentScanline == p.LYCompare {
		p.Status.Coincidence = true
	} else {
		p.Status.Coincidence = false
	}
}

// checkStatInterrupts checks if the STAT interrupt should be triggered.
func (p *PPU) checkStatInterrupts(vblankTrigger bool) {
	lyInt := p.Coincidence && p.CoincidenceInterrupt
	mode0Int := p.Mode == lcd.HBlank && p.HBlankInterrupt
	mode1Int := p.Mode == lcd.VBlank && p.VBlankInterrupt
	mode2Int := p.Mode == lcd.OAM && p.OAMInterrupt
	vBlankInt := vblankTrigger && p.Mode == lcd.OAM // vblank interrupt is triggered at the end of OAM

	// fmt.Println("checking stat interrupts for mode", p.Mode)
	//fmt.Println(p.CoincidenceInterrupt, "lyInt", lyInt, "mode0Int", mode0Int, "mode1Int", mode1Int, "mode2Int", mode2Int, "vBlankInt", vBlankInt)
	if lyInt || mode0Int || mode1Int || mode2Int || vBlankInt {
		// if not stat interrupt requested, request it
		if !p.statInterruptDelay {
			p.irq.Request(interrupts.LCDFlag)
			p.statInterruptDelay = true
		}
	} else {
		p.statInterruptDelay = false
	}
}

func (p *PPU) HasFrame() bool {
	return p.refreshScreen
}

// Tick the PPU by one cycle. This will update the PPU's state and
// render the current scanline if necessary.
func (p *PPU) Tick() {
	if !p.Enabled {
		// p.Mode = lcd.HBlank
		return
	}

	// update the current cycle
	p.currentCycle++

	// 80, 172, 196-204, 456 are the only cycles that we care about
	if !(p.currentCycle == 80 || p.currentCycle == 172 || (p.currentCycle >= 196 && p.currentCycle <= 204) || p.currentCycle == 456) {
		return // avoid switch statement for performance (albeit a small one)
	}

	// step logic (ordered by number of ticks required to optimize calls)
	switch p.Status.Mode {
	case lcd.HBlank:
		// are we handling the line 0 M-cycle delay?
		// https://github.com/Gekkio/mooneye-test-suite/blob/main/acceptance/ppu/lcdon_timing-GS.s#L24
		if p.delayedTick {
			if p.currentCycle == 80 {
				p.delayedTick = false
				p.currentCycle = 0

				p.checkLYC()
				p.checkStatInterrupts(false)

				// go to mode 3
				p.Mode = lcd.VRAM
				return
			}
		}

		// have we reached the cycle threshold for the next scanline?
		if p.currentCycle == hblankCycles[p.ScrollX&0x07] {
			// reset cycle and increment scanline
			p.currentCycle = 0
			p.CurrentScanline++

			// check LYC
			p.checkLYC()

			// check if we've reached the end of the visible screen
			// and need to enter VBlank
			if p.CurrentScanline == 144 {
				// enter VBBlank mode and trigger VBlank interrupt
				p.Mode = lcd.VBlank
				p.checkStatInterrupts(true)

				p.irq.Request(interrupts.VBlankFlag)

				// flag that the screen needs to be refreshed
				p.refreshScreen = true

				// was the LCD just turned on? (the Game Boy never receives the first frame after turning on the LCD)
				if !p.Cleared() {
					p.renderBlank()
				}

				// update palette
				//palette.UpdatePalette()
			} else {
				// enter OAM mode
				p.Mode = lcd.OAM
				p.checkStatInterrupts(false)
			}
		}

	case lcd.VRAM:
		if p.currentCycle == 172 {
			p.currentCycle = 0
			p.Mode = lcd.HBlank

			// notify HDMA that we're in HBlank
			if p.bus.IsGBC() {
				p.bus.HDMA.SetHBlank()
			}
			p.checkStatInterrupts(false)
			p.renderScanline()
		}
	case lcd.OAM:
		if p.currentCycle == 80 {
			p.currentCycle = 0
			p.Mode = lcd.VRAM
		}
	case lcd.VBlank:
		if p.currentCycle == 456 {
			p.currentCycle = 0
			p.CurrentScanline++

			// check LYC
			p.checkLYC()
			p.checkStatInterrupts(false)

			if p.CurrentScanline >= 153 {
				// reset scanline and enter OAM mode
				p.Mode = lcd.OAM
				p.CurrentScanline = 0
				p.WindowYInternal = 0

				// check LYC
				p.checkLYC()
				p.checkStatInterrupts(false)
			}
		}
	}
}

var hblankCycles = []uint16{204, 200, 200, 200, 200, 196, 196, 196}

func (p *PPU) renderScanline() {
	if (p.BackgroundEnabled || p.bus.IsGBC()) && !p.Debug.BackgroundDisabled {
		p.renderBackground()
	} else {
		p.renderBlankLine()
	}
	if p.WindowEnabled && !p.Debug.WindowDisabled {
		p.renderWindow()
	}

	if p.SpriteEnabled && !p.Debug.SpritesDisabled {
		p.renderSprites()
	}

	// send job to the renderer
	if p.bus.IsGBCCompat() {
		job := RenderJobCGB{
			XStart:            p.ScrollX,
			Scanline:          p.scanlineData,
			Sprites:           p.spriteData,
			BackgroundEnabled: p.BackgroundEnabled,
			palettes:          p.compatibilityPalette,
			objPalette:        p.compatibilitySpritePalette,
			Line:              p.CurrentScanline,
		}
		if p.bus.IsGBC() || p.bus.BootROM != nil {
			job.palettes = p.ColourPalette
			job.objPalette = p.ColourSpritePalette
		}
		p.rendererCGB.QueueJob(job)
	} else {
		p.renderer.AddJob(RenderJob{
			XStart:     p.ScrollX,
			Scanline:   p.scanlineData,
			Sprites:    p.spriteData,
			palettes:   p.Palette,
			objPalette: p.SpritePalettes,
			Line:       p.CurrentScanline,
		})
	}

	// clear scanline data
	p.scanlineData = [ScanlineSize]uint8{}
	p.spriteData = [40]uint8{}
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
	for x := uint8(0); x < ScanlineSize; x++ {
		p.scanlineData[x] = 0
	}
}

func (p *PPU) renderWindow() {
	// do nothing if window is out of bounds
	if p.CurrentScanline < p.WindowY {
		return
	} else if p.WindowX > ScreenWidth {
		return
	} else if p.WindowY > ScreenHeight {
		return
	}

	yPos := p.WindowYInternal
	tileYIndex := yPos / 8

	mapOffset := uint8(p.WindowTileMapAddress / 0x9C00)

	// iterate over the 21 (20 if exactly aligned) tiles that make up the scanline
	for i := uint8(0); i < 21; i++ {
		if (i * 8) < p.WindowX-7 {
			continue
		}
		xPos := (i * 8) - (p.WindowX - 7)
		row := yPos % 8

		// get the tile
		tileEntry := p.calculateTileID(tileYIndex, xPos/8, mapOffset)
		tileID := tileEntry.GetID(p.UsingSignedTileData())

		// should we flip the tile vertically?
		if tileEntry.Attributes.YFlip {
			row = 7 - row
		}
		if (p.ScrollX+xPos)%8 != 0 {
			// if we're not aligned to a tile boundary, we need to write to 2 tiles
			// so here we need to work out the offset of the second tile and write to it

			// where are we starting in the first tile?
			offset := (p.ScrollX + xPos) % 8

			// get the existing data to rewrite
			existingData := p.scanlineData[i*TileSizeInBytes]
			existingData2 := p.scanlineData[i*TileSizeInBytes+1]

			// TODO
			// - fix window rendering to be aligned correctly
			// - handle offsets
			// - find way to merge existing attributes with new ones
			// - maybe switch to FIFO rendering?

			// rewrite the first tile
			p.scanlineData[i*TileSizeInBytes] = (p.tileData[tileEntry.Attributes.VRAMBank][tileID][row][0] >> offset) | (existingData << (8 - offset))

			// rewrite the second tile
			p.scanlineData[i*TileSizeInBytes+1] = (p.tileData[tileEntry.Attributes.VRAMBank][tileID][row][1] >> offset) | (existingData2 << (8 - offset))

			// rewrite the attributes

		} else {
			// copy the tile data to the scanline
			p.scanlineData[i*TileSizeInBytes] = p.tileData[tileEntry.Attributes.VRAMBank][tileID][row][0]
			p.scanlineData[i*TileSizeInBytes+1] = p.tileData[tileEntry.Attributes.VRAMBank][tileID][row][1]
			p.scanlineData[i*TileSizeInBytes+2] = tileEntry.Attributes.value
		}

	}

	p.WindowYInternal++
}

func (p *PPU) renderBackground() {
	// determine the y pos and index of the tile
	yPos := p.CurrentScanline + p.ScrollY
	tileYIndex := yPos / 8

	// determine map offset
	mapOffset := uint8(p.BackgroundTileMapAddress / 0x9C00) // 0x9800 = 0, 0x9c00 = 1

	// iterate over the 21 (20 if exactly aligned) tiles that make up a scanline
	var tileEntry TileMapEntry
	for i := uint8(0); i < 21; i++ {
		// which byte is this tile in the scanline?
		scanlineByte := i * TileSizeInBytes

		// get the x pos and row of the tile
		xPos := i*8 + p.ScrollX
		row := yPos % 8

		// get the tile
		tileEntry = p.calculateTileID(tileYIndex, xPos/8, mapOffset)
		tileID := tileEntry.GetID(p.UsingSignedTileData())

		// should we flip the tile?
		if tileEntry.Attributes.YFlip {
			row = 7 - row
		}

		// copy the 3 bytes of tile data into the scanline
		p.scanlineData[scanlineByte] = p.tileData[tileEntry.Attributes.VRAMBank][tileID][row][0]
		p.scanlineData[scanlineByte+1] = p.tileData[tileEntry.Attributes.VRAMBank][tileID][row][1]
		p.scanlineData[scanlineByte+2] = tileEntry.Attributes.value
	}
}

// calculateTileID calculates the tile ID for the current scanline
func (p *PPU) calculateTileID(tilemapOffset, lineOffset uint8, mapOffset uint8) TileMapEntry {
	// get the tile entry from the tilemap
	tileEntry := p.tileMaps[mapOffset][tilemapOffset][lineOffset]

	return tileEntry
}

// renderSprites renders the sprites on the current scanline.
func (p *PPU) renderSprites() {
	spriteCount := 0 // number of sprites on the current scanline (max 10)

	for _, sprite := range p.oam.Sprites {
		byteIndex := spriteCount * SpriteSizeInBytes
		spriteY := sprite.Y - 16
		spriteX := sprite.X - 8

		if spriteY > p.CurrentScanline || spriteY+p.SpriteSize <= p.CurrentScanline {
			continue
		}
		if spriteCount >= 10 {
			break
		}

		// get the row of the sprite to render
		tileRow := p.CurrentScanline - spriteY

		// should we flip the sprite vertically?
		if sprite.FlipY {
			tileRow = p.SpriteSize - tileRow - 1
		}
		tileRow %= 8

		// determine the tile ID
		tileID := uint16(sprite.TileID)
		if p.SpriteSize == 16 {
			if p.CurrentScanline-spriteY < 8 {
				if sprite.FlipY {
					tileID |= 1
				} else {
					tileID &= 0xFE
				}
			} else {
				if sprite.FlipY {
					tileID &= 0xFE
				} else {
					tileID |= 1
				}
			}
		}

		// copy the sprite data to the sprite data
		p.spriteData[byteIndex] = p.tileData[sprite.VRAMBank][tileID][tileRow][0]
		p.spriteData[byteIndex+1] = p.tileData[sprite.VRAMBank][tileID][tileRow][1]
		p.spriteData[byteIndex+2] = sprite.SpriteAttributes.value

		// set the sprite x position for the current scanline
		p.spriteData[byteIndex+3] = spriteX

		// increment the sprite count
		spriteCount++
	}
}

func (p *PPU) ClearRefresh() {
	p.refreshScreen = false
}

// SaveCGBPalettes saves all of the currently active CGB palettes.
func (p *PPU) SaveCGBPalettes() {
	p.ColourPalette.SaveExample("bg.png")
	p.ColourSpritePalette.SaveExample("sprite.png")
}

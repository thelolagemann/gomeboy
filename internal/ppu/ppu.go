// Package ppu provides a programmable pixel unit for the DMG and CGB.
package ppu

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/ppu/background"
	"github.com/thelolagemann/go-gameboy/internal/ppu/lcd"
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"github.com/thelolagemann/go-gameboy/internal/ram"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"image"
	"math/bits"
)

const (
	// ScreenWidth is the width of the screen in pixels.
	ScreenWidth = 160
	// ScreenHeight is the height of the screen in pixels.
	ScreenHeight = 144
)

type PPU struct {
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

	oam                 *OAM
	vRAM                [2]*ram.RAM // Second bank only exists on CGB
	colourPalette       *palette.CGBPalette
	colourSpritePalette *palette.CGBPalette

	tileData [2][384]*Tile // 384 tiles, 8x8 pixels each (double in CGB mode)
	tileMaps [2]TileMap    // 32x32 tiles, 8x8 pixels each

	irq *interrupts.Service

	PreparedFrame [ScreenHeight][ScreenWidth][3]uint8

	scanlineData [63]uint8

	currentCycle       uint16
	bus                *mmu.MMU
	screenCleared      bool
	statInterruptDelay bool
	cleared            bool
	offClock           uint32
	refreshScreen      bool
	DMA                *DMA
	delayedTick        bool

	tileBgPriority [ScreenWidth][ScreenHeight]bool
	renderer       *Renderer
	rendererCGB    *RendererCGB
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
			p.SetMode(lcd.HBlank)

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
		},
		func() uint8 {
			return p.SpritePalettes[0].ToByte()
		},
	)
	types.RegisterHardware(
		types.OBP1,
		func(v uint8) {
			p.SpritePalettes[1] = palette.ByteToPalette(v)
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
	if p.bus.IsGBC() {
		types.RegisterHardware(
			types.VBK,
			func(v uint8) {
				p.vRAMBank = v & types.Bit0 // only the first bit is used
			},
			func() uint8 {
				return p.vRAMBank
			},
		)
		types.RegisterHardware(
			types.BCPS,
			func(v uint8) {
				p.colourPalette.SetIndex(v)
			},
			func() uint8 {
				return p.colourPalette.GetIndex()
			},
		)
		types.RegisterHardware(
			types.BCPD,
			func(v uint8) {
				p.colourPalette.Write(v)
			},
			func() uint8 {
				// TODO handle locked state
				return p.colourPalette.Read()
			},
		)
		types.RegisterHardware(
			types.OCPS,
			func(v uint8) {
				p.colourSpritePalette.SetIndex(v)
			},
			func() uint8 {
				return p.colourSpritePalette.GetIndex()
			},
		)
		types.RegisterHardware(
			types.OCPD,
			func(v uint8) {
				p.colourSpritePalette.Write(v)
			},
			func() uint8 {
				// TODO handle locked state
				return p.colourSpritePalette.Read()
			},
		)
	}

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

	output := make(chan *RenderOutput, ScreenHeight)
	// setup renderer
	if p.bus.IsGBC() {
		renderJobs := make(chan RenderJobCGB, ScreenHeight)
		p.rendererCGB = NewRendererCGB(renderJobs, output)
	} else {
		renderJobs := make(chan RenderJob, 20)
		p.renderer = NewRenderer(renderJobs, output)
	}

	// create goroutine to handle rendering
	go func() {
		var renderOutput *RenderOutput
		for i := 0; i < ScreenHeight; i++ { // the last 8 scanlines are rendered as they are needed to avoid flickering
			renderOutput = <-output
			p.PreparedFrame[renderOutput.Line] = renderOutput.Scanline

			if i == ScreenHeight-1 {
				// the last scanline has been rendered, so the frame is ready
				p.refreshScreen = true
				i = 0
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

	// initialize CGB features
	if mmu.IsGBC() {
		p.colourPalette = palette.NewCGBPallette()
		p.colourSpritePalette = palette.NewCGBPallette()
		p.vRAM[1] = ram.NewRAM(0x2000)
		mmu.HDMA.AttachVRAM(p.WriteVRAM)
	}

	p.init()
	return p
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
			tileEntry := p.calculateTileID(j, i, 0x1800)
			// get tile data
			tile := p.tileData[tileEntry.Attributes.VRAMBank][tileEntry.GetID(p.UsingSignedTileData())]
			tile.Draw(img, int(i*8), int(j*8))
		}
	}

	// draw tilemap (0x9C00 - 0x9FFF)
	for i := uint8(0); i < 32; i++ {
		for j := uint8(0); j < 32; j++ {
			tileEntry := p.calculateTileID(j, i, 0x1C00)

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
	if p.bus.IsGBC() {
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

	if p.bus.IsGBC() {
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
	if p.currentCycle < 80 {
		return // nothing happens during the first 80 cycles
	}

	// step logic (ordered by number of ticks required to optimize calls)
	switch p.Status.Mode {
	case lcd.HBlank:
		// are we handling the line 0 M-cycle delay?
		// https://github.com/Gekkio/mooneye-test-suite/blob/main/acceptance/ppu/lcdon_timing-GS.s#L24
		if p.delayedTick && p.currentCycle == 80 {
			p.delayedTick = false
			p.currentCycle = 0

			p.checkLYC()
			p.checkStatInterrupts(false)

			// go to mode 3
			p.SetMode(lcd.VRAM)
			return
		}

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
				p.SetMode(lcd.VBlank)
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
				p.SetMode(lcd.OAM)
				p.checkStatInterrupts(false)
			}
		}

	case lcd.VRAM:
		if p.currentCycle == 172 {
			//fmt.Println("PPU: Tick", p.currentCycle, p.CurrentScanline, p.Mode)
			p.currentCycle = 0
			p.SetMode(lcd.HBlank)

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
			p.SetMode(lcd.VRAM)
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
				p.SetMode(lcd.OAM)
				p.CurrentScanline = 0
				p.WindowYInternal = 0

				// check LYC
				p.checkLYC()
				p.checkStatInterrupts(false)
			}
		}
	}
}

var hblankCycles = [8]uint16{204, 200, 200, 200, 200, 196, 196, 196}

func (p *PPU) renderScanline() {
	if p.BackgroundEnabled || p.bus.IsGBC() {
		p.renderBackground()
	} else {
		p.renderBlankLine()
	}
	if p.WindowEnabled {
		p.renderWindow()
	}

	if p.SpriteEnabled {
		p.renderSprites()
	}

	// send job to the renderer
	if p.bus.IsGBC() {
		p.rendererCGB.QueueJob(RenderJobCGB{
			XStart:     p.ScrollX,
			Scanline:   p.scanlineData,
			palettes:   p.colourPalette,
			objPalette: p.colourSpritePalette,
			Line:       p.CurrentScanline,
		})
	} else {
		p.renderer.AddJob(RenderJob{
			Scanline:   p.scanlineData,
			palettes:   p.Palette,
			objPalette: p.SpritePalettes,
			Line:       p.CurrentScanline,
		})
	}

	// clear scanline data
	p.scanlineData = [ScanlineSize]uint8{}
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
	for x := 0; x < ScreenWidth; x++ {
		p.scanlineData[x] = 0
	}
}

func (p *PPU) renderWindow() {
	var xPos, yPos uint8

	// do nothing if window is out of bounds
	if p.CurrentScanline < p.WindowY {
		return
	} else if p.WindowX > ScreenWidth {
		return
	} else if p.WindowY > ScreenHeight {
		return
	}

	yPos = p.WindowYInternal
	tileYIndex := (yPos) / 8

	var mapOffset uint16

	if p.WindowTileMapAddress == 0x9800 {
		mapOffset = 0x1800
	} else {
		mapOffset = 0x1C00
	}

	// iterate over the 21 (20 if exactly aligned) tiles that make up the scanline
	for i := uint8(0); i < 21; i++ {
		if (i * 8) < p.WindowX-7 {
			continue
		}

		xPos = (i * 8) - (p.WindowX - 7)

		// get the tile
		tileEntry := p.calculateTileID(tileYIndex, xPos/8, mapOffset)
		tileID := tileEntry.GetID(p.UsingSignedTileData())

		// get the tile data
		var b1, b2 uint8
		b1 = p.tileData[tileEntry.Attributes.VRAMBank][tileID][yPos%8][0]
		b2 = p.tileData[tileEntry.Attributes.VRAMBank][tileID][yPos%8][1]

		// should we flip the tile vertically?
		if tileEntry.Attributes.YFlip {
			b1 = p.tileData[tileEntry.Attributes.VRAMBank][tileID][7-(yPos%8)][0]
			b2 = p.tileData[tileEntry.Attributes.VRAMBank][tileID][7-(yPos%8)][1]
		}

		// should we flip the tile horizontally?
		if tileEntry.Attributes.XFlip {
			b1 = bits.Reverse8(b1)
			b2 = bits.Reverse8(b2)
		}

		// copy the tile data to the scanline
		p.scanlineData[i*3] = b1
		p.scanlineData[i*3+1] = b2

		// copy the tile info to the scanline
		p.scanlineData[i*3+2] = tileEntry.Attributes.PaletteNumber << 4
	}

	/*for i := uint8(0); i < ScreenWidth; i++ {
		// set BG priority
		p.tileBgPriority[i][p.CurrentScanline] = tileEntry.Attributes.UseBGPriority

	}*/
	p.WindowYInternal++
}

func (p *PPU) renderBackground() {
	// setup variables
	var xPos, yPos uint8

	yPos = p.CurrentScanline + p.ScrollY
	tileYIndex := yPos / 8

	var mapOffset uint16
	if p.BackgroundTileMapAddress == 0x9800 {
		mapOffset = 0x1800
	} else {
		mapOffset = 0x1C00
	}

	// iterate over the 21 (20 if exactly aligned) tiles that make up a scanline
	for i := uint8(0); i < 21; i++ {
		xPos = i*8 + p.ScrollX
		// get the tile
		tileEntry := p.calculateTileID(tileYIndex, xPos/8, mapOffset)
		tileID := tileEntry.GetID(p.UsingSignedTileData())

		var b1, b2 uint8
		b1 = p.tileData[tileEntry.Attributes.VRAMBank][tileID][yPos%8][0]
		b2 = p.tileData[tileEntry.Attributes.VRAMBank][tileID][yPos%8][1]

		// should we flip the tile?
		if tileEntry.Attributes.YFlip {
			b1 = p.tileData[tileEntry.Attributes.VRAMBank][tileID][7-(yPos%8)][0]
			b2 = p.tileData[tileEntry.Attributes.VRAMBank][tileID][7-(yPos%8)][1]
		}
		if tileEntry.Attributes.XFlip {
			b1 = bits.Reverse8(b1)
			b2 = bits.Reverse8(b2)
		}

		// copy the 2 bytes of tile data into the scanline
		p.scanlineData[i*3] = b1
		p.scanlineData[i*3+1] = b2

		// copy the byte of tile info into the scanline
		p.scanlineData[i*3+2] = tileEntry.Attributes.PaletteNumber << 4
	}

	/*
		for i := uint8(0); i < ScreenWidth; i++ {
			// set BG priority
			p.tileBgPriority[i][p.CurrentScanline] = tileEntry.Attributes.UseBGPriority
		}*/
}

// calculateTileID calculates the tile ID for the current scanline
func (p *PPU) calculateTileID(tilemapOffset, lineOffset uint8, mapOffset uint16) TileMapEntry {
	// determine which tilemap to use
	var tilemapNumber uint8
	if mapOffset == 0x1800 {
		tilemapNumber = 0
	} else {
		tilemapNumber = 1
	}
	// get the tile entry from the tilemap
	tileEntry := p.tileMaps[tilemapNumber][tilemapOffset][lineOffset]

	return tileEntry
}

// renderSprites renders the sprites on the current scanline.
func (p *PPU) renderSprites() {
	// spriteXPerScreen := [ScreenWidth]uint8{}
	spriteCount := 0 // number of sprites on the current scanline (max 10)

	for _, sprite := range p.oam.Sprites {

		if sprite.GetY() > p.CurrentScanline || sprite.GetY()+p.SpriteSize <= p.CurrentScanline {
			continue
		}
		if spriteCount >= 10 {
			break
		}
		spriteCount++
		// determine which byte of the scanline to start writing to
		startByte := (sprite.GetX() / 8) * 3
		if startByte > 60 {
			continue
		}

		// get the row of the sprite to render
		tileRow := p.CurrentScanline - sprite.GetY()

		// should we flip the sprite vertically?
		if sprite.FlipY {
			tileRow = p.SpriteSize - tileRow - 1
		}

		// determine the tile ID
		tileID := sprite.TileID
		if p.SpriteSize == 16 {
			if tileRow < 8 {
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

		// get the tile data
		var b1, b2 uint8
		b1 = p.tileData[sprite.VRAMBank][tileID][tileRow%8][0]
		b2 = p.tileData[sprite.VRAMBank][tileID][tileRow%8][1]

		// should we flip the sprite horizontally?
		if sprite.FlipX {
			b1 = bits.Reverse8(b1)
			b2 = bits.Reverse8(b2)
		}

		// fmt.Println("startTile", startTile)

		// copy the tile data to the scanline
		p.scanlineData[startByte] = b1
		p.scanlineData[startByte+1] = b2

		// copy the tile info to the scanline
		p.scanlineData[startByte+2] = sprite.CGBPalette<<4 | sprite.UseSecondPalette<<3 | types.Bit2

		/*
			tilerowIndex := p.CurrentScanline - sprite.GetY()
			if sprite.FlipY {
				tilerowIndex = p.SpriteSize - tilerowIndex - 1
			}
			tilerowIndex %= 8
			tileID := uint16(sprite.TileID)
			if p.SpriteSize == 16 {
				if p.CurrentScanline-sprite.GetY() < 8 {
					if sprite.FlipY {
						tileID |= 0x01
					} else {
						tileID &= 0xFE
					}
				} else {
					if sprite.FlipY {
						tileID &= 0xFE
					} else {
						tileID |= 0x01
					}
				}
			}
			tilerow := p.tileData[sprite.VRAMBank][tileID][tilerowIndex]

			for x := uint8(0); x < 8; x++ {
				// skip if the sprite is out of bounds
				pixelPos := sprite.GetX() + x
				if pixelPos < 0 || pixelPos >= ScreenWidth {
					continue
				}

				// get the color of the pixel using the sprite palette
				color := tilerow[x]
				if sprite.FlipX {
					color = tilerow[7-x]
				}

				// skip if the color is transparent
				if color == 0 {
					continue
				}

				// skip if the sprite doesn't have priority and the background is not transparent
				if !p.bus.IsGBC() || p.BackgroundEnabled {
					if !(sprite.Priority && !p.tileBgPriority[pixelPos][p.CurrentScanline]) &&
						(p.scanlineData[pixelPos] != 0) {
						continue
					}
				}

				if p.bus.IsGBC() {
					// skip if the sprite doesn't have priority and the background is not transparent
					if spriteXPerScreen[pixelPos] != 0 {
						continue
					}
				} else {
					// skip if pixel is occupied by sprite with lower x coordinate
					if spriteXPerScreen[pixelPos] != 0 && spriteXPerScreen[pixelPos] <= sprite.GetX()+10 {
						continue
					}
				}

				p.scanlineData[pixelPos] = sprite.CGBPalette<<4 | sprite.UseSecondPalette<<3 | types.Bit2 | color

				// mark the pixel as occupied
				spriteXPerScreen[pixelPos] = sprite.GetX() + 10
			}*/
	}
}

func (p *PPU) ClearRefresh() {
	p.refreshScreen = false
}

func (p *PPU) getObjectColourFromPalette(paletteNumber uint8, colour uint8) [3]uint8 {
	return p.SpritePalettes[paletteNumber].GetColour(colour)
}

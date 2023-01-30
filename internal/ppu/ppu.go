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
	"github.com/thelolagemann/go-gameboy/internal/types/registers"
	"image"
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
	vRAM                [2]ram.RAM // Second bank only exists on CGB
	colourPalette       *palette.CGBPalette
	colourSpritePalette *palette.CGBPalette

	tileData [2][384]*Tile // 384 tiles, 8x8 pixels each (double in CGB mode)
	tileMaps [2]TileMap    // 32x32 tiles, 8x8 pixels each (double in CGB mode)

	irq *interrupts.Service

	PreparedFrame [ScreenWidth][ScreenHeight][3]uint8

	currentCycle       int16
	bus                *mmu.MMU
	ScreenData         [ScreenWidth][ScreenHeight][3]uint8
	tileScanline       [ScreenWidth]uint8
	screenCleared      bool
	statInterruptDelay bool
	cleared            bool
	offClock           uint32
	refreshScreen      bool
	DMA                *DMA
	delayedTick        bool

	currentFramePalette palette.Colour

	tileBgPriority [ScreenWidth][ScreenHeight]bool

	// new cgb stuff

	tileAttributes [2048]*TileAttributes // 32x32 tile attributes
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
			fmt.Println(p.Mode)
		}
	})
	p.Status = lcd.NewStatus(func(writeFn func()) {
		writeFn()
		if p.Enabled {
			p.checkStatInterrupts(false)
		}
	})

	// setup registers
	registers.RegisterHardware(
		registers.LY,
		func(v uint8) {
			// any write to LY resets the value to 0
			p.CurrentScanline = 0
		},
		func() uint8 {
			return p.CurrentScanline
		},
	)
	registers.RegisterHardware(
		registers.LYC,
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
	registers.RegisterHardware(
		registers.OBP0,
		func(v uint8) {
			p.SpritePalettes[0] = palette.ByteToPalette(v)
		},
		func() uint8 {
			return p.SpritePalettes[0].ToByte()
		},
	)
	registers.RegisterHardware(
		registers.OBP1,
		func(v uint8) {
			p.SpritePalettes[1] = palette.ByteToPalette(v)
		},
		func() uint8 {
			return p.SpritePalettes[1].ToByte()
		},
	)
	registers.RegisterHardware(
		registers.WX,
		func(v uint8) {
			p.WindowX = v
		},
		func() uint8 {
			return p.WindowX
		},
	)
	registers.RegisterHardware(
		registers.WY,
		func(v uint8) {
			p.WindowY = v
		},
		func() uint8 {
			return p.WindowY
		},
	)

	// CGB registers
	if p.bus.IsGBC() {
		registers.RegisterHardware(
			registers.VBK,
			func(v uint8) {
				p.vRAMBank = v & types.Bit0 // only the first bit is used
			},
			func() uint8 {
				return p.vRAMBank
			},
		)
		registers.RegisterHardware(
			registers.BCPS,
			func(v uint8) {
				p.colourPalette.SetIndex(v)
			},
			func() uint8 {
				return p.colourPalette.GetIndex()
			},
		)
		registers.RegisterHardware(
			registers.BCPD,
			func(v uint8) {
				p.colourPalette.Write(v)
			},
			func() uint8 {
				// TODO handle locked state
				return p.colourPalette.Read()
			},
		)
		registers.RegisterHardware(
			registers.OCPS,
			func(v uint8) {
				p.colourSpritePalette.SetIndex(v)
			},
			func() uint8 {
				return p.colourSpritePalette.GetIndex()
			},
		)
		registers.RegisterHardware(
			registers.OCPD,
			func(v uint8) {
				p.colourSpritePalette.Write(v)
			},
			func() uint8 {
				// TODO handle locked state
				return p.colourSpritePalette.Read()
			},
		)
	}

	// initialize tile attributes
	for i := 0; i < len(p.tileAttributes); i++ {
		p.tileAttributes[i] = &TileAttributes{}
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
			p.tileMaps[i] = TileMap{}
		}
	}
}

func New(mmu *mmu.MMU, irq *interrupts.Service) *PPU {
	oam := NewOAM()
	p := &PPU{
		Background:   background.NewBackground(),
		currentCycle: 0,

		tileData: [2][384]*Tile{},

		bus: mmu,
		irq: irq,
		oam: oam,
		vRAM: [2]ram.RAM{
			ram.NewRAM(8192),
			ram.NewRAM(8192), // TODO only create if CGB
		},
		DMA:                 NewDMA(mmu, oam),
		colourPalette:       palette.NewCGBPallette(),
		colourSpritePalette: palette.NewCGBPallette(),
		tileAttributes:      [2048]*TileAttributes{},
	}
	mmu.HDMA.AttachVRAM(p.WriteVRAM)

	p.init()
	return p
}

func (p *PPU) PrepareFrame() {
	p.PreparedFrame = p.ScreenData
	p.ScreenData = [ScreenWidth][ScreenHeight][3]uint8{}
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
	// 2 tilemaps per bank (0x1800 bytes) (32x32 tiles)
	// 1 byte per tile (0-255)
	fmt.Println("Dumping tilemaps")

	var img *image.RGBA
	if p.bus.IsGBC() {
		img = image.NewRGBA(image.Rect(0, 0, 512, 512))
	} else {
		img = image.NewRGBA(image.Rect(0, 0, 256, 256))
	}

	// draw tilemap (0x9800 - 0x9BFF)
	for x := 0; x < 32; x++ {
		for y := 0; y < 32; y++ {
			// get tile number
			tileNumber := int(p.vRAM[0].Read(uint16(0x1800 + (y*32 + x))))

			// is it a signed tile number?
			if p.Controller.UsingSignedTileData() {
				if tileNumber < 128 {
					tileNumber += 256
				}
			}

			// get tile
			tile := p.tileData[0][tileNumber]
			tile.Draw(img, x*8, y*8)

			// get attributes
			attributes := p.tileAttributes[y*32+x]
			attributes.Draw(img, x*8, y*8)
		}
	}

	// draw tilemap (0x9C00 - 0x9FFF)
	for x := 0; x < 32; x++ {
		for y := 0; y < 32; y++ {
			// get tile number
			tileNumber := int(p.vRAM[0].Read(uint16(0x1C00 + (y*32 + x))))

			// is it a signed tile number?
			if p.Controller.UsingSignedTileData() {
				if tileNumber < 128 {
					tileNumber += 256
				}
			}

			// get tile
			tile := p.tileData[0][tileNumber]
			tile.Draw(img, x*8, y*8+256)

			// get attributes
			attributes := p.tileAttributes[y*32+x+1024]
			attributes.Draw(img, x*8, y*8+256)
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
	if p.bus.IsGBC() {
		// write to the current VRAM bank
		p.vRAM[p.vRAMBank].Write(address, value)

		// are we writing to the tile data?
		if address <= 0x17FF {
			p.UpdateTile(address)
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
				p.UpdateTileAttributes(address-0x1800, value)
			}
		}
	} else {
		p.vRAM[0].Write(address, value)
		if address <= 0x17FF {
			p.UpdateTile(address)
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
func (p *PPU) UpdateTile(index uint16) {
	// get the tile index
	index &= 0x1FFE // only the lower 13 bits are used

	// get the tileID
	tileID := index >> 4 // divide by 16

	// get the tile row
	row := (index >> 1) & 0x7

	// iterate over the 8 pixels in the row
	for i := 0; i < 8; i++ {
		bitIndex := uint8(1 << (7 - i))

		// get the low and high bits
		low := 0
		high := 0

		if p.vRAM[p.vRAMBank].Read(index)&bitIndex != 0 {
			low = 1
		}

		if p.vRAM[p.vRAMBank].Read(index+1)&bitIndex != 0 {
			high = 2
		}

		// set the pixel
		p.tileData[p.vRAMBank][tileID][row][i] = (low) + (high)
	}
}

func (p *PPU) UpdateTileMap(address uint16, tilemapIndex uint8) {
	// determine the y and x position
	y := (address / 32) & 0x1F
	x := address & 0x1F

	// update the tilemap
	p.tileMaps[tilemapIndex][y][x] = p.vRAM[0].Read(address)
}

func (p *PPU) UpdateTileAttributes(index uint16, value uint8) {
	// panic(fmt.Sprintf("updating tile %x with %b", index, value))
	// get the ID of the tile being updated (0-383)
	p.tileAttributes[index].Write(index, value)
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

// HasDoubleSpeed returns false as the PPU does not respond to double speed.
func (p *PPU) HasDoubleSpeed() bool {
	return false
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

	// step logic
	switch p.Status.Mode {
	case lcd.HBlank:
		if p.bus.IsGBC() {
			p.bus.HDMA.SetHBlank()
		}

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

		if p.currentCycle == p.hblankCycles() {
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
				palette.UpdatePalette()
			} else {
				// enter OAM mode
				p.SetMode(lcd.OAM)
				p.checkStatInterrupts(false)
			}
		}
	case lcd.VBlank:
		if p.currentCycle == 456 {

			//fmt.Println("PPU: Tick", p.currentCycle, p.CurrentScanline, p.Mode)
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
	case lcd.OAM:
		if p.currentCycle == 80 {
			//fmt.Println("PPU: Tick", p.currentCycle, p.CurrentScanline, p.Mode)
			p.currentCycle = 0
			p.SetMode(lcd.VRAM)
		}
	case lcd.VRAM:
		if p.currentCycle == 172 {
			//fmt.Println("PPU: Tick", p.currentCycle, p.CurrentScanline, p.Mode)
			p.currentCycle = 0
			p.SetMode(lcd.HBlank)
			p.checkStatInterrupts(false)

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
		}
	}
}

func (p *PPU) hblankCycles() int16 {
	switch p.ScrollX & 0x07 {
	case 0x00:
		return 204
	case 0x01, 0x02, 0x03, 0x04:
		return 200
	case 0x05, 0x06, 0x07:
		return 196
	}

	return 0
}

func (p *PPU) renderBlank() {
	for x := 0; x < ScreenWidth; x++ {
		for y := 0; y < ScreenHeight; y++ {
			p.ScreenData[x][y] = p.Palette.GetColour(0)
		}
	}
	p.Clear()
}

func (p *PPU) renderBlankLine() {
	for x := 0; x < ScreenWidth; x++ {
		p.ScreenData[x][p.CurrentScanline] = p.Palette.GetColour(0)
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

	for i := uint8(0); i < ScreenWidth; i++ {
		if i < p.WindowX-7 {
			continue
		}

		xPos = i - (p.WindowX - 7)
		tileXIndex := (xPos / 8)

		tileID := uint16(p.calculateTileID(tileYIndex, tileXIndex, mapOffset))
		// get the tile attributes for the tile
		tileAttributes := p.tileAttributes[mapOffset-0x1800+uint16((tileYIndex)*32)+uint16(tileXIndex)]

		// get pixel position within tile
		xPixelPos := xPos % 8
		yPixelPos := yPos % 8

		// are we flipping?
		if p.bus.IsGBC() {
			if tileAttributes.YFlip {
				yPixelPos = 7 - yPixelPos
			}
			if tileAttributes.XFlip {
				xPixelPos = 7 - xPixelPos
			}
		}

		// get the colour (shade) of the pixel using the background palette
		pixelShade := p.tileData[tileAttributes.VRAMBank][tileID][yPixelPos][xPixelPos]

		// convert the shade to a colour
		pixelColour := p.Palette.GetColour(uint8(pixelShade))

		if p.bus.IsGBC() {
			pixelColour = p.colourPalette.GetColour(tileAttributes.PaletteNumber, uint8(pixelShade))
			p.tileBgPriority[i][p.CurrentScanline] = tileAttributes.UseBGPriority
		}

		// set the pixel on the screen
		p.ScreenData[i][p.CurrentScanline] = pixelColour
	}
	p.WindowYInternal++
}

func (p *PPU) renderBackground() {
	var xPos, yPos uint8

	yPos = p.CurrentScanline + p.ScrollY
	tileYIndex := yPos / 8

	var mapOffset uint16
	if p.BackgroundTileMapAddress == 0x9800 {
		mapOffset = 0x1800
	} else {
		mapOffset = 0x1C00
	}

	for i := uint8(0); i < ScreenWidth; i++ {
		// determine the x position of the pixel
		xPos = i + p.ScrollX
		tileXIndex := xPos / 8

		// determine the tile ID to draw from the tile map
		tileID := uint16(p.calculateTileID(tileYIndex, tileXIndex, mapOffset))

		// get the tile attributes for the tile
		tileAttributes := p.tileAttributes[mapOffset-0x1800+uint16(tileYIndex)*32+uint16(tileXIndex)]

		// get pixel position within tile
		xPixelPos := xPos % 8
		yPixelPos := yPos % 8

		// are we flipping?
		if p.bus.IsGBC() {
			if tileAttributes.YFlip {
				yPixelPos = 7 - yPixelPos
			}
			if tileAttributes.XFlip {
				xPixelPos = 7 - xPixelPos
			}
		}

		// get the colour (shade) of the pixel using the background palette
		pixelShade := p.tileData[tileAttributes.VRAMBank][tileID][yPixelPos][xPixelPos]

		// convert the shade to a colour
		var pixelColour [3]uint8

		if p.bus.IsGBC() {
			pixelColour = p.colourPalette.GetColour(tileAttributes.PaletteNumber, byte(pixelShade))
			p.tileBgPriority[i][p.CurrentScanline] = tileAttributes.UseBGPriority
		} else {
			pixelColour = p.Palette.GetColour(uint8(pixelShade))
		}
		p.ScreenData[i][p.CurrentScanline] = pixelColour
	}

}

// calculateTileID calculates the tile ID for the current scanline
func (p *PPU) calculateTileID(tilemapOffset, lineOffset uint8, mapOffset uint16) int {
	// determine which tilemap to use
	var tilemapNumber uint8
	if mapOffset == 0x1800 {
		tilemapNumber = 0
	} else {
		tilemapNumber = 1
	}
	// determine the tile ID to draw from the tile map

	tileID := int(p.tileMaps[tilemapNumber][tilemapOffset][lineOffset])

	// if the tile ID is signed, we need to convert it to an unsigned value
	if p.UsingSignedTileData() {
		if tileID < 128 {
			tileID += 256
		}
	}

	return tileID
}

// renderSprites renders the sprites on the current scanline.
func (p *PPU) renderSprites() {
	spriteXPerScreen := [ScreenWidth]uint8{}
	spriteCount := 0 // number of sprites on the current scanline (max 10)

	for _, sprite := range p.oam.Sprites {

		if sprite.GetY() > p.CurrentScanline || sprite.GetY()+p.SpriteSize <= p.CurrentScanline {
			continue
		}
		if spriteCount >= 10 {
			break
		}
		spriteCount++

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
					p.ScreenData[pixelPos][p.CurrentScanline] != p.colourPalette.GetColour(0, 0) {
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

			rgb := p.getObjectColourFromPalette(sprite.UseSecondPalette, uint8(color))

			if p.bus.IsGBC() {
				rgb = p.colourSpritePalette.GetColour(sprite.CGBPalette, uint8(color))
			}

			// draw the pixel
			p.ScreenData[pixelPos][p.CurrentScanline] = rgb

			// mark the pixel as occupied
			spriteXPerScreen[pixelPos] = sprite.GetX() + 10
		}
	}
}

func (p *PPU) ClearRefresh() {
	p.refreshScreen = false
}

func (p *PPU) getObjectColourFromPalette(paletteNumber uint8, colour uint8) [3]uint8 {
	return p.SpritePalettes[paletteNumber].GetColour(colour)
}

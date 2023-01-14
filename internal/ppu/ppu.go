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
)

const (
	// ScreenWidth is the width of the screen in pixels.
	ScreenWidth = 160
	// ScreenHeight is the height of the screen in pixels.
	ScreenHeight = 144
)

const (
	// CurrentScanlineRegister is the address of the Current Scanline Register.
	// This register contains the current scanline being drawn and is read-only. It
	// can hold values from 0 to 153. With values from 144 to 153, the PPU is in
	// V-Blank.
	CurrentScanlineRegister = 0xFF44
	// LYCompareRegister is the address of the LY Compare Register. This register
	// contains the value that the Current Scanline Register is compared to. When
	// the two registers are equal, the LYC=LY Coincidence flag in the LCD Status
	// Register is set, and a STAT interrupt is requested.
	LYCompareRegister = 0xFF45
	// DMARegister is the address of the DMA Register. This register contains the
	// address of the ROM or RAM to copy to the PPU's OAM. The address is divided by 100
	// and the first 160 bytes are copied to the PPU's OAM.
	DMARegister = 0xFF46
	// ObjectPalette0Register is the address of the Object Palette 0 Register.
	// This register contains the color palette for sprite 0, and is only used in DMG
	// mode. They work exactly the same as the Background Palette Register, except
	// that the lower two bits are ignored (color index 0 is transparent).
	//
	//  Bit 7-6 - Color for Shade 3
	//  Bit 5-4 - Color for Shade 2
	//  Bit 3-2 - Color for Shade 1
	//  Bit 1-0 - Not used
	ObjectPalette0Register = 0xFF48
	// ObjectPalette1Register is the address of the Object Palette 1 Register.
	// This register contains the color palette for sprite 1, and is only used in DMG
	// mode. They work exactly the same as the Background Palette Register, except
	// that the lower two bits are ignored (color index 0 is transparent).
	//
	//  Bit 7-6 - Color for Shade 3
	//  Bit 5-4 - Color for Shade 2
	//  Bit 3-2 - Color for Shade 1
	//  Bit 1-0 - Not used
	ObjectPalette1Register = 0xFF49
	// WindowYRegister is the address of the Window Y Register. This register
	// contains the Y position of the window. The window is displayed when the
	// Window Display Enable bit in the LCD Control Register is set to 1, and the
	// window is positioned above the background and sprites.
	WindowYRegister = 0xFF4A
	// WindowXRegister is the address of the Window X Register. This register
	// contains the X position of the window minus 7. The window is displayed when
	// the Window Display Enable bit in the LCD Control Register is set to 1, and
	// the window is positioned above the background and sprites.
	WindowXRegister = 0xFF4B
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

	oam  ram.RAM
	vRAM ram.RAM

	tileData               [2][512]Tile
	rawTileData            [2][512][16]uint8
	sprites                [40]Sprite
	currentTileLineDotData *[8]int

	irq *interrupts.Service

	PreparedFrame [ScreenWidth][ScreenHeight][3]uint8

	currentCycle       int16
	bus                mmu.IOBus
	screenData         [ScreenWidth][ScreenHeight][3]uint8
	tileScanline       [ScreenWidth]uint8
	screenCleared      bool
	statInterruptDelay bool
	lcdIntThrow        bool
	vBlankIntThrow     bool
	cleared            bool
	dmaRegister        uint8

	currentFramePalette palette.Colour

	HasDMA bool
}

func New(mmu mmu.IOBus, irq *interrupts.Service) *PPU {
	p := &PPU{
		Background:      background.NewBackground(),
		Controller:      lcd.NewController(),
		Status:          lcd.NewStatus(),
		CurrentScanline: 0,
		LYCompare:       0,
		SpritePalettes:  [2]palette.Palette{},
		WindowX:         0,
		WindowY:         0,
		currentCycle:    0,

		sprites:                [40]Sprite{},
		currentTileLineDotData: new([8]int),

		bus:  mmu,
		irq:  irq,
		oam:  ram.NewRAM(160),
		vRAM: ram.NewRAM(8192),
	}

	// initialize sprites
	for i := 0; i < 40; i++ {
		p.sprites[i] = NewSprite()
	}
	return p
}

func (p *PPU) Read(address uint16) uint8 {
	// read from VRAM
	if address >= 0x8000 && address <= 0x9FFF {
		return p.vRAM.Read(address - 0x8000)
	}
	// read from OAM
	if address >= 0xFE00 && address <= 0xFE9F {
		return p.oam.Read(address - 0xFE00)
	}
	switch address {
	case lcd.ControlRegister:
		return p.Controller.Read(address)
	case lcd.StatusRegister:
		return 0xFF
	case background.ScrollYRegister, background.ScrollXRegister, background.PaletteRegister:
		return p.Background.Read(address)
	case CurrentScanlineRegister:
		return p.CurrentScanline
	case LYCompareRegister:
		return p.LYCompare
	case ObjectPalette0Register:
		return p.SpritePalettes[0].ToByte()
	case ObjectPalette1Register:
		return p.SpritePalettes[1].ToByte()
	case WindowXRegister:
		return p.WindowX
	case WindowYRegister:
		return p.WindowY
	case DMARegister:
		return p.dmaRegister
	default:
		panic(fmt.Sprintf("ppu: illegal read from address %04X", address))
	}
}

// doHDMATransfer performs a DMA transfer from the given address to the PPU's OAM.
func (p *PPU) doHDMATransfer(value uint8) {
	srcAddress := uint16(value) << 8 // src address is value * 100 (shift left 8 bits)
	for i := 0; i < 0xA0; i++ {
		toAddress := 0xFE00 + uint16(i) // OAM starts at 0xFE00 then subtract 0x8000 to get the offset
		p.Write(toAddress, p.bus.Read(srcAddress+uint16(i)))
	}
	p.HasDMA = true
}

func (p *PPU) Write(address uint16, value uint8) {
	// write to VRAM
	if address >= 0x8000 && address <= 0x9FFF {
		p.vRAM.Write(address-0x8000, value)
		// update tile data
		p.UpdateTile(address-0x8000, value)
		return
	}
	// write to OAM
	if address >= 0xFE00 && address <= 0xFE9F {
		p.oam.Write(address-0xFE00, value)
		// update sprite data
		p.UpdateSprite(address-0xFE00, value)
		return
	}

	switch address {
	case lcd.ControlRegister:
		p.Controller.Write(address, value)
	case lcd.StatusRegister:
		p.Status.Write(address, value)
	case background.ScrollXRegister, background.ScrollYRegister, background.PaletteRegister:
		p.Background.Write(address, value)
	case CurrentScanlineRegister:
		// writing to this register resets it to 0
		p.CurrentScanline = 0
	case LYCompareRegister:
		p.LYCompare = value
	case ObjectPalette0Register:
		p.SpritePalettes[0] = palette.ByteToPalette(value)
	case ObjectPalette1Register:
		p.SpritePalettes[1] = palette.ByteToPalette(value)
	case WindowXRegister:
		p.WindowX = value
	case WindowYRegister:
		p.WindowY = value
	case DMARegister:
		p.doHDMATransfer(value)
		p.dmaRegister = value
	default:
		panic(fmt.Sprintf("ppu: illegal write to address %04X", address))
	}
}

func (p *PPU) UpdateSprite(address uint16, value uint8) {
	spriteId := address & 0x00FF / 4
	p.sprites[spriteId].UpdateSprite(address, value)
}

// UpdateTile updates the tile at the given index with the given data.
func (p *PPU) UpdateTile(index uint16, value uint8) {
	// get the ID of the tile being updated (0-383)
	tileID := uint16((index&0x1FFF)>>4) & 511

	// update the tile data
	p.rawTileData[0][tileID][index%16] = value

	// update the tile
	p.tileData[0][tileID] = NewTile(p.rawTileData[0][tileID])
}

// Step advances the PPU by the given number of cycles. This is
// to keep the PPU in sync with the CPU.
func (p *PPU) Step(cycles uint16) {
	if !p.Enabled {
		// reset values if the LCD is disabled
		p.CurrentScanline = 0
		p.currentCycle = 0
		p.WindowYInternal = 0

		// LCD goes into HBlank mode when disabled
		p.SetMode(lcd.HBlank)
	}

	// update the current cycle
	p.currentCycle += int16(cycles)

	// step logic
	switch p.Status.Mode {
	case lcd.HBlank:
		// HBlank (85 to 208 dots) TODO : adjust timing
		if p.currentCycle >= 204 {
			p.currentCycle = 0
			p.stepScanline()

			// check if we've reached the end of the screen (144 lines)
			if p.CurrentScanline == 144 {
				// set VBlank and request an interrupt
				p.SetMode(lcd.VBlank)
				p.irq.Request(interrupts.VBlankFlag)

				// render the frame
				p.PreparedFrame = p.screenData

				// reset values
				p.screenData = [ScreenWidth][ScreenHeight][3]uint8{}
				p.WindowYInternal = 0

				// the LCD never receives the first frame after being enabled,
				// so we need to render a blank frame
				// https://github.com/pinobatch/little-things-gb/tree/master/firstwhite
				if !p.Cleared() {
					p.renderBlank()
				}

				// update the palette (to avoid mid-frame palette changes)
				palette.UpdatePalette()
			} else {
				p.SetMode(lcd.OAM)
			}
		}
	case lcd.VBlank:
		// VBlank (4560 dots, 10 lines) (144 to 153 lines)
		if p.currentCycle >= 456 {
			p.currentCycle = 0
			p.stepScanline()

			// check if we've reached the end of the VBlank period (10 lines)
			if p.CurrentScanline > 153 {
				// reset scanline to 0, and set to HBlank
				p.CurrentScanline = 0
				p.SetMode(lcd.OAM)
			}
		}
	case lcd.OAM:
		// OAM (80 dots)
		if p.currentCycle >= 80 {
			p.currentCycle = 0
			p.SetMode(lcd.VRAM)
		}
	case lcd.VRAM:
		// 172 cycles VRAM
		if p.currentCycle >= 172 {
			p.currentCycle = 0
			p.SetMode(lcd.HBlank)

			// if background enabled, render the background
			if p.BackgroundEnabled {
				p.renderBackground()
			} else {
				// otherwise, render a blank line
				p.renderBlankLine()
			}

			// if background and window enabled, render the window (as window piggybacks
			// on the background rendering, so if the background is disabled, the window
			// will be disabled too)
			if p.BackgroundEnabled && p.WindowEnabled {
				p.renderWindow()
			}

			// if sprites enabled, render the sprites
			if p.SpriteEnabled {
				p.renderSprites()
			}
		}
	}

	// should we request an interrupt?
	lyInt := p.LYCompare == p.CurrentScanline && p.CoincidenceInterrupt
	mode0Int := p.HBlankInterrupt && p.Status.Mode == lcd.HBlank
	mode1Int := p.VBlankInterrupt && p.Status.Mode == lcd.VBlank
	mode2Int := p.OAMInterrupt && p.Status.Mode == lcd.OAM

	if p.detectRisingEdge(lyInt || mode0Int || mode1Int || mode2Int) {
		p.irq.Request(interrupts.LCDFlag)
	}
}

func (p *PPU) stepScanline() {
	p.CurrentScanline++
}

func (p *PPU) renderBlank() {
	for x := 0; x < ScreenWidth; x++ {
		for y := 0; y < ScreenHeight; y++ {
			p.screenData[x][y] = p.Palette.GetColour(0)
		}
	}
	p.PreparedFrame = p.screenData
	p.Clear()
}

func (p *PPU) renderBlankLine() {
	for x := 0; x < ScreenWidth; x++ {
		p.screenData[x][p.CurrentScanline] = p.Palette.GetColour(0)
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
	tileYIndex := p.WindowTileMapAddress + uint16(yPos)/8*32

	for i := uint8(0); i < ScreenWidth; i++ {
		if i < p.WindowX-7 {
			continue
		}

		xPos = i - (p.WindowX - 7)
		tileXIndex := uint16(xPos / 8)

		tileID := p.calculateTileID(tileYIndex, tileXIndex)

		// get pixel position within tile
		xPixelPos := xPos % 8
		yPixelPos := yPos % 8

		// get the color of the pixel using the background palette
		color := p.tileData[0][tileID][yPixelPos][xPixelPos]
		p.screenData[i][p.CurrentScanline] = p.Palette.GetColour(uint8(color))
	}
	p.WindowYInternal++
}

func (p *PPU) renderBackground() {
	var xPos, yPos uint8

	yPos = p.CurrentScanline + p.ScrollY
	tileYIndex := p.BackgroundTileMapAddress + uint16(yPos/8)*32
	for i := uint8(0); i < ScreenWidth; i++ {
		xPos = i + p.ScrollX
		tileXIndex := uint16(xPos / 8)
		tileID := p.calculateTileID(tileYIndex, tileXIndex)

		// get pixel position within tile
		xPixelPos := xPos % 8
		yPixelPos := yPos % 8

		// get the color of the pixel using the background palette
		color := p.tileData[0][tileID][yPixelPos][xPixelPos]
		p.screenData[i][p.CurrentScanline] = p.Palette.GetColour(uint8(color))
	}
}

func (p *PPU) detectRisingEdge(signal bool) bool {
	result := signal && !p.statInterruptDelay
	p.statInterruptDelay = signal
	return result
}

// calculateTileID calculates the tile ID for the current scanline
func (p *PPU) calculateTileID(tilemapOffset, lineOffset uint16) int {
	// determine the tile ID
	tileID := int(p.vRAM.Read(tilemapOffset + lineOffset - 0x8000))

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

	for _, sprite := range p.sprites {

		if sprite.Y > p.CurrentScanline || sprite.Y+p.SpriteSize <= p.CurrentScanline {
			continue
		}
		if spriteCount >= 10 {
			break
		}
		spriteCount++

		tilerowIndex := p.CurrentScanline - sprite.Y
		if sprite.FlipY {
			tilerowIndex = p.SpriteSize - tilerowIndex - 1
		}
		tilerowIndex %= 8
		tileID := uint16(sprite.tileID)
		if p.SpriteSize == 16 {
			if p.CurrentScanline-sprite.Y < 8 {
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

		tilerow := p.tileData[0][tileID][tilerowIndex]

		for x := uint8(0); x < 8; x++ {
			// skip if the sprite is out of bounds
			pixelPos := sprite.X + x
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
			if p.BackgroundEnabled {
				if !sprite.Priority && p.screenData[pixelPos][p.CurrentScanline] != p.SpritePalettes[sprite.UseSecondPalette].GetColour(0) {
					continue
				}
			}

			// skip if pixel is occupied by sprite with lower x coordinate
			if spriteXPerScreen[pixelPos] != 0 && spriteXPerScreen[pixelPos] <= sprite.X {
				continue
			}

			// draw the pixel
			p.screenData[pixelPos][p.CurrentScanline] = p.SpritePalettes[sprite.UseSecondPalette].GetColour(uint8(color))

			// mark the pixel as occupied
			spriteXPerScreen[pixelPos] = sprite.X
		}
	}
}

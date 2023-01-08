// Package ppu provides a programmable pixel unit for the DMG and CGB.
package ppu

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/ppu/background"
	"github.com/thelolagemann/go-gameboy/internal/ppu/lcd"
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"github.com/thelolagemann/go-gameboy/pkg/utils"
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
	Sprite0Palette  uint8
	Sprite1Palette  uint8
	WindowX         uint8
	WindowY         uint8

	irq *interrupts.Service

	PreparedFrame [ScreenWidth][ScreenHeight][3]uint8

	currentCycle       int16
	bus                mmu.IOBus
	bgPriority         [ScreenWidth][ScreenHeight]bool
	screenData         [ScreenWidth][ScreenHeight][3]uint8
	tileScanline       [ScreenWidth]uint8
	screenCleared      bool
	statInterruptDelay bool
}

func New(mmu mmu.IOBus, irq *interrupts.Service) *PPU {
	return &PPU{
		Background:      background.NewBackground(),
		Controller:      lcd.NewController(),
		Status:          lcd.NewStatus(),
		CurrentScanline: 0,
		LYCompare:       0,
		Sprite0Palette:  0,
		Sprite1Palette:  0,
		WindowX:         0,
		WindowY:         0,
		currentCycle:    0,

		bus: mmu,
		irq: irq,
	}
}

func (p *PPU) Read(address uint16) uint8 {
	switch address {
	case lcd.ControlRegister:
		return p.Controller.Read(address)
	case lcd.StatusRegister:
		return p.Status.Read(address)
	case background.ScrollYRegister, background.ScrollXRegister, background.PaletteRegister:
		return p.Background.Read(address)
	case CurrentScanlineRegister:
		return p.CurrentScanline
	case LYCompareRegister:
		return p.LYCompare
	case ObjectPalette0Register:
		return p.Sprite0Palette
	case ObjectPalette1Register:
		return p.Sprite1Palette
	case WindowXRegister:
		return p.WindowX
	case WindowYRegister:
		return p.WindowY
	case DMARegister:
		return 0xFF
	default:
		panic(fmt.Sprintf("ppu: illegal read from address %04X", address))
	}
}

func (p *PPU) Write(address uint16, value uint8) {
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
		p.Sprite0Palette = value
	case ObjectPalette1Register:
		p.Sprite1Palette = value
	case WindowXRegister:
		p.WindowX = value
	case WindowYRegister:
		p.WindowY = value
	default:
		panic(fmt.Sprintf("ppu: illegal write to address %04X", address))
	}
}

// Step advances the PPU by the given number of cycles. This is
// to keep the PPU in sync with the CPU.
func (p *PPU) Step(cycles uint16) {
	p.setLCDStatus()

	if !p.Controller.Enabled {
		return
	}

	p.currentCycle -= int16(cycles)
	if p.currentCycle <= 0 {
		// we've reached the end of the current scanline and need to move on to the next one
		p.CurrentScanline++
		p.currentCycle = 456

		// if we've reached the start of the VBlank period, we need to set the VBlank interrupt flag
		if p.CurrentScanline == 144 {
			p.irq.Request(interrupts.VBlankFlag)
		} else if p.CurrentScanline > 153 {
			p.CurrentScanline = 0
			p.renderFrame()
		} else if p.CurrentScanline < 144 {
			p.renderScanline()
		}
	}
}

// setLCDStatus determines the current LCD status and sets the appropriate bits
// in the LCD Status Register.
func (p *PPU) setLCDStatus() {
	if !p.Controller.Enabled {
		// if the LCD is disabled, set mode to 0 and reset
		p.Status.SetMode(lcd.HBlank)

		// reset the current scanline to 0
		p.CurrentScanline = 0
		p.currentCycle = 456
		return
	}

	currentMode := p.Status.Mode
	reqInt := false
	if p.CurrentScanline >= 144 {
		p.Status.SetMode(lcd.VBlank)
		reqInt = p.Status.VBlankInterrupt
	} else {
		if p.currentCycle >= 376 {
			p.Status.SetMode(lcd.OAM)
			reqInt = p.Status.OAMInterrupt
		} else if p.currentCycle >= 204 {
			p.Status.SetMode(lcd.VRAM)
		} else {
			p.Status.SetMode(lcd.HBlank)
			reqInt = p.Status.HBlankInterrupt
		}
	}

	// if the current mode is different from the previous mode, we need to request an interrupt
	if reqInt && currentMode != p.Status.Mode {
		p.irq.Request(interrupts.LCDFlag)
	}

	// if LY == LYC, we need to set the coincidence flag and request an interrupt if necessary
	if p.Status.CoincidenceInterrupt && p.CurrentScanline == p.LYCompare {
		p.Coincidence = true
		p.irq.Request(interrupts.LCDFlag)
	} else {
		p.Coincidence = false
	}
}

// renderScanline renders the current scanline to the screen.
func (p *PPU) renderScanline() {
	if p.Controller.BackgroundEnabled {
		p.renderBackground()
	}

	if p.Controller.SpriteEnabled {
		p.renderSprites()
	}
}

// renderTiles renders the background and window tiles to the screen.
func (p *PPU) renderTiles() {
	var usingWindow = false

	// determine if we're using the window or the background
	if p.Controller.WindowEnabled && p.WindowY <= p.CurrentScanline {
		usingWindow = true
	}

	var tileMap uint16

	if usingWindow {
		tileMap = p.Controller.WindowTileMapAddress
	} else {
		tileMap = p.Controller.BackgroundTileMapAddress
	}

	// yPos is used to determine which of the 32 vertical tiles we're rendering
	yPos := byte(0)
	if usingWindow {
		yPos = p.CurrentScanline - p.WindowY
	} else {
		yPos = p.CurrentScanline + p.Background.ScrollY
	}

	// determine tile row
	tileRow := uint16(yPos/8) * 32

	// start rendering the tiles
	for pixel := uint8(0); pixel < ScreenWidth; pixel++ {
		// xPos is used to determine which of the 32 horizontal tiles we're rendering
		xPos := pixel + p.WindowX

		// if we're using the window, we need to offset xPos by the window's X position
		if usingWindow && pixel >= p.WindowX {
			xPos = pixel - p.WindowX
		}

		// determine tile column
		tileCol := uint16(xPos / 8)

		// determine tile number (which can be signed or unsigned)
		var tileNum uint16

		tileAddress := tileMap + tileRow + tileCol
		if !p.UsingSignedTileData() {
			tileNum = uint16(p.bus.Read(tileAddress))
		} else {
			tileNum = uint16(int8(p.bus.Read(tileAddress)))
		}

		// determine the tile's address
		var tileLocation = p.TileDataAddress
		if !p.UsingSignedTileData() {
			tileLocation += tileNum * 16
		} else {
			tileLocation += (tileNum + 128) * 16
		}

		// determine the line of the tile to render
		line := byte(yPos % 8)
		line *= 2
		data1 := p.bus.Read(tileLocation + uint16(line))
		data2 := p.bus.Read(tileLocation + uint16(line) + 1)

		// determine pixel color
		colourBit := int((xPos%8)-7) * -1
		colourNum := utils.Val(data2, uint8(colourBit))<<1 | utils.Val(data1, uint8(colourBit))

		// determine the color palette to use
		colour := palette.GetColour(colourNum)

		// set the pixels
		p.screenData[pixel][p.CurrentScanline] = colour
	}
}

func (p *PPU) renderBackground() {
	// yPos is used to determine which of the 32 vertical tiles we're rendering
	yPos := byte(0)
	yPos = p.CurrentScanline + p.Background.ScrollY

	// determine tile row
	tileRow := uint16(yPos/8) * 32

	// start rendering the tiles
	for pixel := uint8(0); pixel < ScreenWidth; pixel++ {
		// xPos is used to determine which of the 32 horizontal tiles we're rendering
		xPos := pixel + p.Background.ScrollX

		// determine tile column
		tileCol := uint16(xPos / 8)

		// determine the tile's address
		var tileLocation = p.TileDataAddress
		tileAddress := p.BackgroundTileMapAddress + tileRow + tileCol
		if !p.UsingSignedTileData() {
			tileNum := int16(p.bus.Read(tileAddress))
			tileLocation = tileLocation + uint16(tileNum*16)
		} else {
			tileNum := int16(int8(p.bus.Read(tileAddress)))
			tileLocation = uint16(int32(tileLocation) + int32((tileNum+128)*16))
		}

		// determine the line of the tile to render
		line := yPos % 8
		line *= 2
		data1 := p.bus.Read(tileLocation + uint16(line))
		data2 := p.bus.Read(tileLocation + uint16(line) + 1)

		// determine pixel color
		colourBit := int8((xPos%8)-7) * -1
		colourNum := utils.Val(data2, uint8(colourBit))<<1 | utils.Val(data1, uint8(colourBit))

		// determine the color palette to use
		colour := palette.GetColour(colourNum)

		// set the pixels
		p.screenData[pixel][p.CurrentScanline] = colour
	}
}

// renderSprites renders the sprites on the current scanline.
func (p *PPU) renderSprites() {
	for i := uint8(0); i < 40; i++ {
		// sprite occupies 4 bytes in OAM
		index := i * 4
		yPos := p.bus.Read(0xFE00+uint16(index)) - 16
		xPos := p.bus.Read(0xFE00+uint16(index)+1) - 8

		tileLocation := p.bus.Read(0xFE00 + uint16(index) + 2)
		attributes := p.bus.Read(0xFE00 + uint16(index) + 3)

		yFlip := utils.Test(attributes, 6)
		xFlip := utils.Test(attributes, 5)

		// does sprite overlap with current scanline?
		if p.CurrentScanline >= yPos && p.CurrentScanline < yPos+p.Controller.SpriteSize {
			// determine which line of the tile to render
			line := int(p.CurrentScanline - yPos)

			// if the sprite is flipped vertically, we need to render the opposite line
			if yFlip {
				line -= int(p.Controller.SpriteSize)
				line *= -1
			}

			line *= 2
			tileAddress := (0x8000 + uint16(tileLocation)*16) + uint16(line)
			data1 := p.bus.Read(tileAddress)
			data2 := p.bus.Read(tileAddress + 1)

			for tilePixel := int8(7); tilePixel >= 0; tilePixel-- {
				colourBit := tilePixel
				if xFlip {
					colourBit -= 7
					colourBit *= -1
				}

				colourNum := utils.Val(data2, uint8(colourBit))
				colourNum <<= 1
				colourNum |= utils.Val(data1, uint8(colourBit))

				colour := palette.GetColour(colourNum)

				xPix := 0 - int(tilePixel)
				xPix += 7

				pixel := xPos + uint8(xPix)

				// check within bounds
				if pixel < 0 || pixel >= ScreenWidth || p.CurrentScanline < 0 || p.CurrentScanline >= ScreenHeight {
					continue
				}

				p.screenData[pixel][p.CurrentScanline] = colour
			}
		}
	}
}

// renderFrame renders the current frame to the screen.
func (p *PPU) renderFrame() {
	for x := 0; x < ScreenWidth; x++ {
		for y := 0; y < ScreenHeight; y++ {
			p.PreparedFrame[x][y] = p.screenData[x][y]
		}
	}
}

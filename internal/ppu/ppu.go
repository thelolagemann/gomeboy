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

	oam  ram.RAM
	vRAM ram.RAM

	tileData               [2][512]Tile
	rawTileData            [2][512][16]uint8
	sprites                [40]Sprite
	spritesLarge           [40]Sprite
	currentTileLineDotData *[8]int

	irq *interrupts.Service

	PreparedFrame [ScreenWidth][ScreenHeight][3]uint8

	currentCycle       int16
	bus                mmu.IOBus
	screenData         [ScreenWidth][ScreenHeight][3]uint8
	tileScanline       [ScreenWidth]uint8
	screenCleared      bool
	statInterruptDelay bool
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
		spritesLarge:           [40]Sprite{},
		currentTileLineDotData: new([8]int),

		bus:  mmu,
		irq:  irq,
		oam:  ram.NewRAM(160),
		vRAM: ram.NewRAM(8192),
	}

	// initialize sprites
	for i := 0; i < 40; i++ {
		p.sprites[i] = NewDefaultSprite()
		p.spritesLarge[i] = NewLargeSprite()
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
		// TODO: check if OAM is locked
		return p.oam.Read(address - 0xFE00)
	}
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
		return p.SpritePalettes[0].ToByte()
	case ObjectPalette1Register:
		return p.SpritePalettes[1].ToByte()
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
	default:
		panic(fmt.Sprintf("ppu: illegal write to address %04X", address))
	}
}

func (p *PPU) UpdateSprite(address uint16, value uint8) {
	spriteId := address & 0x00FF / 4
	if p.SpriteSize == 8 {
		p.sprites[spriteId].UpdateSprite(address, value)
	} else {
		p.spritesLarge[spriteId].UpdateSprite(address, value)
	}
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
			for _, s := range p.sprites {
				s.ResetScanlines()
			}
			for _, s := range p.spritesLarge {
				s.ResetScanlines()
			}
			p.irq.Request(interrupts.VBlankFlag)
			p.PreparedFrame = p.screenData
		} else if p.CurrentScanline > 153 {
			p.CurrentScanline = 0
		} else if p.CurrentScanline < 144 {
			if p.Controller.BackgroundEnabled {
				p.renderBackground()
			}

			if p.Controller.WindowEnabled {
				p.renderWindow()
			}

			if p.Controller.SpriteEnabled {
				p.renderSprites()
			}
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

func (p *PPU) renderWindow() {
	// determine yPos
	yPos := int(p.CurrentScanline - p.WindowY)

	if (p.WindowX <= 166) && (p.WindowY <= 143) && yPos >= 0 {
		tileMapOffset := p.WindowTileMapAddress + uint16(yPos)/8*32
		lineOffset := uint16(0)

		xPos := int((p.WindowX - 7) % 255)

		// determine where in the tile we are
		tileX := xPos % 8
		tileY := yPos % 8

		// draw scanline
		p.drawScanline(tileMapOffset, lineOffset, xPos, tileX, tileY)
	}
}

func (p *PPU) renderBackground() {
	// determine yPos
	yPos := int(p.CurrentScanline) + int(p.Background.ScrollY)
	tilemapOffset := p.BackgroundTileMapAddress + uint16(yPos)%256/8*32
	lineOffset := uint16(p.ScrollX) / 8 % 32

	// determine where in the tile we are
	tileX := int(p.ScrollX) % 8
	tileY := yPos % 8

	// draw scanline
	p.drawScanline(tilemapOffset, lineOffset, 0, tileX, tileY)
}

// drawScanline draws the current scanline
func (p *PPU) drawScanline(tilemapOffset, lineOffset uint16, screenX, tileX, tileY int) {
	// determine the tile ID
	tileID := p.calculateTileID(tilemapOffset, lineOffset)

	// draw loop
	for ; screenX < ScreenWidth; screenX++ {

		// draw the pixel to the framebuffer
		p.screenData[screenX][p.CurrentScanline] = p.Background.Palette.GetColour(uint8(p.tileData[0][tileID][tileY][tileX]))

		// increment the tile X position until we reach the end of the tile
		tileX++
		if tileX == 8 {
			tileX = 0
			lineOffset = (lineOffset + 1) % 32

			// get the next tile ID
			tileID = p.calculateTileID(tilemapOffset, lineOffset)
		}
	}
}

// calculateTileID calculates the tile ID for the current scanline
func (p *PPU) calculateTileID(tilemapOffset, lineOffset uint16) int {
	// determine the tile ID
	tileID := int(p.bus.Read(tilemapOffset + lineOffset))

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
	if p.Controller.SpriteSize == 8 {
		for _, sprite := range p.sprites {
			if sprite.Attributes().X != 0x00 && sprite.Attributes().Y != 0x00 {
				if sprite.Attributes().Y-16 <= 0 {
					sprite.PushScanlines(int(p.CurrentScanline), 8)
				} else if sprite.Attributes().Y-16 == int(p.CurrentScanline) {
					sprite.PushScanlines(int(p.CurrentScanline), 8)
				}

				if !sprite.IsScanlineEmpty() {
					if scanline, tileLine := sprite.PopScanline(); scanline == int(p.CurrentScanline) {
						p.drawSpriteLine(sprite, sprite.TileID(0), 0, tileLine)
					}
				}
			}
		}
	} else {
		for _, sprite := range p.spritesLarge {
			if sprite.Attributes().X != 0x00 && sprite.Attributes().Y != 0x00 {
				if sprite.Attributes().Y-16 <= 0 {
					sprite.PushScanlines(int(p.CurrentScanline), 16)
				} else if sprite.Attributes().Y-16 == int(p.CurrentScanline) {
					sprite.PushScanlines(int(p.CurrentScanline), 16)
				}

				if !sprite.IsScanlineEmpty() {
					if scanline, tileLine := sprite.PopScanline(); scanline == int(p.CurrentScanline) {
						if sprite.Attributes().FlipY {
							if tileLine < 8 {
								p.drawSpriteLine(sprite, sprite.TileID(1), 0, tileLine)
							} else {
								p.drawSpriteLine(sprite, sprite.TileID(0), 8, tileLine-8)
							}
						} else {
							if tileLine < 8 {
								p.drawSpriteLine(sprite, sprite.TileID(0), 0, tileLine)
							} else {
								p.drawSpriteLine(sprite, sprite.TileID(1), 8, tileLine-8)
							}
						}
					}
				}
			}
		}
	}
}

// drawSpriteLine draws a single line of a sprite to the framebuffer.
func (p *PPU) drawSpriteLine(sprite Sprite, tileId, yOffset, tileY int) {
	if sprite.Attributes().X >= 0 && sprite.Attributes().Y >= 0 {
		var t = &p.tileData[0][tileId]
		formatTileLine(t, tileY, sprite.Attributes().FlipX, sprite.Attributes().FlipY, p.currentTileLineDotData)

		sx, sy := sprite.Attributes().X-8, sprite.Attributes().Y-16
		for tileX := 0; tileX < 8; tileX++ {
			if p.currentTileLineDotData[tileX] != 0 {
				adjX, adjY := sx+tileX, sy+tileY+yOffset
				if (adjY < ScreenHeight && adjY >= 0) && (adjX < ScreenWidth && adjX >= 0) {
					// if sprite doesn't have priority and background color isn't shade 0 then don't draw
					if sprite.Attributes().Priority && p.screenData[adjX][adjY] != palette.GetColour(0) {
						continue
					}
					p.screenData[adjX][adjY] = p.SpritePalettes[sprite.Attributes().UseSecondPalette].GetColour(uint8(p.currentTileLineDotData[tileX]))
				}
			}
		}
	}
}

func formatTileLine(t *Tile, tileY int, flipX, flipY bool, tileLine *[8]int) {
	// flip both
	if flipX && flipY {
		for x := 0; x < 8; x++ {
			tileLine[x] = t[7-tileY][7-x]
		}
		return
	}
	if flipX {
		for x := 0; x < 8; x++ {
			tileLine[x] = t[tileY][7-x]
		}
		return
	}
	if flipY {
		for y := 0; y < len(t[7-tileY]); y++ {
			tileLine[y] = t[7-tileY][y]
		}
		return
	}
	for y := 0; y < len(t[tileY]); y++ {
		tileLine[y] = t[tileY][y]
	}
}

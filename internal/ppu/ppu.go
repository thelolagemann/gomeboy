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
		if p.Status.Mode == lcd.VRAM || p.Status.Mode == lcd.OAM {
			return 0xFF
		}
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
		// if mode 2 or 3, the cpu can't access OAM
		if p.Status.Mode == lcd.OAM || p.Status.Mode == lcd.VRAM {
			return
		}
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
	// update the status register
	if !p.Enabled {
		p.CurrentScanline = 0
		p.currentCycle = 456
		p.SetMode(lcd.HBlank)
		p.WindowYInternal = 0
	} else {
		if p.CurrentScanline >= 144 {
			p.SetMode(lcd.VBlank)
			p.lcdIntThrow = false
		} else if p.currentCycle >= 456-80 {
			p.SetMode(lcd.OAM)
			p.lcdIntThrow = false
		} else if p.currentCycle >= 456-80-172 {
			p.SetMode(lcd.VRAM)
			p.lcdIntThrow = false
		} else {
			p.SetMode(lcd.HBlank)
			if p.HBlankInterrupt && !p.lcdIntThrow {
				p.irq.Request(interrupts.LCDFlag)
				p.lcdIntThrow = true
			}
		}
	}

	p.currentCycle -= int16(cycles)
	if p.currentCycle <= 0 {
		p.currentCycle += 456
		p.CurrentScanline++
		// if the scanline is 144, we are in VBlank and we need to throw an interrupt
		if p.CurrentScanline == 144 {
			if !p.vBlankIntThrow {
				p.irq.Request(interrupts.VBlankFlag)

				// throw LCD interrupt if enabled
				if p.VBlankInterrupt {
					p.irq.Request(interrupts.LCDFlag)
				}
				p.vBlankIntThrow = true
			}

			// draw the screen
			p.PreparedFrame = p.screenData
			p.WindowYInternal = 0
		} else if p.CurrentScanline > 153 {
			p.CurrentScanline = 0
			p.vBlankIntThrow = false
		}

		// throw LCD interrupt if enabled
		if p.CurrentScanline == p.LYCompare && p.CoincidenceInterrupt {
			p.irq.Request(interrupts.LCDFlag)
			p.Status.Write(lcd.StatusRegister, p.Status.Read(lcd.StatusRegister)|0x04)
		}

		// render the scanline
		if p.CurrentScanline < 144 {
			if p.Enabled {
				if p.BackgroundEnabled {
					p.renderBackground()
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
		tileXIndex := xPos / 8

		tileID := p.calculateTileID(tileYIndex, uint16(tileXIndex))
		if p.UsingSignedTileData() && tileID < 128 {
			tileID += 256
		}

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
	tileYIndex := p.BackgroundTileMapAddress + uint16(yPos)%256/8*32

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

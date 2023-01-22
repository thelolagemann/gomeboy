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
	"image"
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

	oam                 ram.RAM
	vRAM                [2]ram.RAM // Second bank only exists on CGB
	vRAMBank            uint8
	colourPalette       *palette.CGBPalette
	colourSpritePalette *palette.CGBPalette

	tileData    [2][384]*Tile // 384 tiles, 8x8 pixels each (double in CGB mode)
	rawTileData [2][384][16]uint8
	sprites     [40]Sprite

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

	// new cgb stuff

	tileAttributes [0x800]*TileAttributes // 32x32 tile attributes
}

func New(mmu *mmu.MMU, irq *interrupts.Service) *PPU {
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

		sprites:  [40]Sprite{},
		tileData: [2][384]*Tile{},

		bus: mmu,
		irq: irq,
		oam: ram.NewRAM(160),
		vRAM: [2]ram.RAM{
			ram.NewRAM(8192),
			ram.NewRAM(8192), // TODO only create if CGB
		},
		vRAMBank:            0,
		DMA:                 NewDMA(mmu),
		colourPalette:       palette.NewCGBPallette(),
		colourSpritePalette: palette.NewCGBPallette(),
		tileAttributes:      [0x800]*TileAttributes{},
	}

	// initialize sprites
	for i := 0; i < 40; i++ {
		p.sprites[i] = NewSprite()
	}

	// initialize tile attributes
	for i := 0; i < len(p.tileAttributes); i++ {
		p.tileAttributes[i] = &TileAttributes{}
	}

	// initialize tile data
	for i := 0; i < 768; i++ {
		p.tileData[i/384][i%384] = &Tile{}
	}
	return p
}

func (p *PPU) PrepareFrame() {
	p.PreparedFrame = p.ScreenData
	p.ScreenData = [ScreenWidth][ScreenHeight][3]uint8{}
}

func (p *PPU) Read(address uint16) uint8 {
	// read from VRAM
	if address >= 0x8000 && address <= 0x9FFF {
		// are we reading from the tile data?
		if address >= 0x8000 && address <= 0x97FF {
			return p.vRAM[p.vRAMBank].Read(address - 0x8000)
		}
		// are we reading from the tile attributes?
		if address >= 0x9800 && address <= 0x9FFF {
			return p.tileAttributes[address-0x9800].Read(address)
		}
		return 0xff
	}

	// read from OAM
	if address >= 0xFE00 && address <= 0xFE9F {
		if p.oamUnlocked() && !p.DMA.IsTransferring() {
			return p.oam.Read(address - 0xFE00)
		}
		return 0xff
	}

	// read from registers
	switch address {
	case lcd.ControlRegister:
		return p.Controller.Read(lcd.ControlRegister)
	case lcd.StatusRegister:
		return p.Status.Read(lcd.StatusRegister) // bit 7 is always set
	case background.ScrollYRegister:
		return p.Background.ScrollY
	case background.ScrollXRegister:
		return p.Background.ScrollX
	case CurrentScanlineRegister:
		return p.CurrentScanline
	case LYCompareRegister:
		return p.LYCompare
	case DMARegister:
		return p.DMA.Read(DMARegister)
	case background.PaletteRegister:
		return p.Background.Palette.ToByte()
	case ObjectPalette0Register:
		return p.SpritePalettes[0].ToByte()
	case ObjectPalette1Register:
		return p.SpritePalettes[1].ToByte()
	case WindowYRegister:
		return p.WindowY
	case WindowXRegister:
		return p.WindowX
	case 0xFF4F:
		if p.bus.IsGBC() {
			return p.vRAMBank | 0xFE
		} else {
			return 0xFF
		}
	case 0xFF68:
		if p.bus.IsGBC() {
			return p.colourPalette.GetIndex()
		} else {
			return 0xFF
		}
	case 0xFF69:
		if p.bus.IsGBC() && p.colorPaletteUnlocked() {
			panic("be")
			return p.colourPalette.Read()
		} else {
			return 0xFF
		}
	case 0xFF6A:
		if p.bus.IsGBC() {
			return p.colourSpritePalette.GetIndex()
		} else {
			return 0xFF
		}
	case 0xFF6B:
		if p.bus.IsGBC() && p.colorPaletteUnlocked() {
			panic("be")
			return p.colourSpritePalette.Read()
		} else {
			return 0xFF
		}
	}

	panic(fmt.Sprintf("PPU: Read from invalid address: %X", address))
}

func (p *PPU) vramUnlocked() bool {
	return p.Mode != lcd.VRAM
}

func (p *PPU) oamUnlocked() bool {
	return p.Mode != lcd.OAM && p.Mode != lcd.VRAM
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
		img = image.NewRGBA(image.Rect(0, 0, 512, 96))
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
			x := (i%32)*8 + 256
			y := (i / 32) * 8

			tile.Draw(img, x, y)
		}
	}

	return img
}

func (p *PPU) colorPaletteUnlocked() bool {
	return p.Mode != lcd.VRAM
}

func (p *PPU) WriteVRAM(address uint16, value uint8) {
	if p.vramUnlocked() {
		if address <= 0x17FF { // tile data
			// p.tileData[p.vRAMBank][address] = value

		}
	}
}

func (p *PPU) Write(address uint16, value uint8) {
	// write to VRAM
	if address >= 0x8000 && address <= 0x9FFF {
		// write the raw data to the VRAM bank currently selected (0+ is CGB only)
		p.vRAM[p.vRAMBank].Write(address-0x8000, value)

		// are we writing to the tile data?
		if address >= 0x8000 && address <= 0x97FF {
			p.UpdateTile(address-0x8000, value)
			return
		}

		// are we writing to the tile attributes? (vram bank 1)
		if address >= 0x9800 && address <= 0x9FFF && p.vRAMBank == 1 {
			p.UpdateTileAttributes(address-0x9800, value)
			return
		}

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
		wasOn := p.Enabled
		p.Controller.Write(address, value)

		// if the screen was turned off, clear the screen
		if wasOn && !p.Enabled {
			// the screen should not be turned off unless in vblank
			if p.Mode != lcd.VBlank {
				panic("PPU: Screen was turned off while not in VBlank")
			}

			// enter hblank
			p.SetMode(lcd.HBlank)

			// reset the scanline
			p.CurrentScanline = 0
		} else if !wasOn && p.Enabled {
			p.checkLYC()
			p.checkStatInterrupts(false)
			// if the screen was turned on, reset the clock
			p.SetMode(lcd.HBlank)
			p.currentCycle = 4
			p.delayedTick = true
		}
	case lcd.StatusRegister:
		p.Status.Write(address, value)
		if p.Enabled {
			p.checkStatInterrupts(false)
		}
	case background.ScrollXRegister, background.ScrollYRegister, background.PaletteRegister:
		p.Background.Write(address, value)
	case CurrentScanlineRegister:
		// writing to this register resets it to 0
		p.CurrentScanline = 0
	case LYCompareRegister:
		p.LYCompare = value

		// check if the LYC interrupt should be triggered
		if p.Enabled {
			p.checkLYC()
			p.checkStatInterrupts(false)
		}
	case ObjectPalette0Register:
		p.SpritePalettes[0] = palette.ByteToPalette(value)
	case ObjectPalette1Register:
		p.SpritePalettes[1] = palette.ByteToPalette(value)
	case WindowXRegister:
		p.WindowX = value
	case WindowYRegister:
		p.WindowY = value
	case DMARegister:
		p.DMA.Write(address, value)
	case 0xFF4F:
		if p.bus.IsGBC() {
			p.vRAMBank = value & 0x01 // only bit 0 is used
		}
	case 0xFF68:
		if p.bus.IsGBC() {
			p.colourPalette.SetIndex(value)
		}
	case 0xFF69:
		if p.bus.IsGBC() && p.colorPaletteUnlocked() {
			p.colourPalette.Write(value)
		}
	case 0xFF6A:
		if p.bus.IsGBC() {
			p.colourSpritePalette.SetIndex(value)
		}
	case 0xFF6B:
		if p.bus.IsGBC() && p.colorPaletteUnlocked() {
			p.colourSpritePalette.Write(value)
		}
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
	// get the tileID
	index &= 0x1FFE // clear the last bit

	// get the tileID
	tileID := index >> 4

	// get the tile row
	y := (index >> 1) & 0x07 // clear the last 3 bits

	// iterate over the 8 pixels in the row
	for x := uint16(0); x < 8; x++ {
		bitIndex := uint8(1 << (7 - x))
		low := 0
		high := 0

		if p.vRAM[p.vRAMBank].Read(index)&bitIndex != 0 {
			low = 1
		}
		if p.vRAM[p.vRAMBank].Read(index+1)&bitIndex != 0 {
			high = 2
		}

		// set the pixel in the tile
		p.tileData[p.vRAMBank][tileID][y][x] = (low + high)
	}
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
		p.Mode = lcd.HBlank
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
				// p.renderSprites()
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
	//fmt.Println("rendering window")
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

		// get the colour (shade) of the pixel using the background palette
		pixelShade := p.tileData[p.vRAMBank][tileID][yPixelPos][xPixelPos]

		// convert the shade to a colour
		pixelColour := p.Palette.GetColour(uint8(pixelShade))

		if p.bus.IsGBC() {
			pixelColour = p.colourPalette.GetColour(p.tileAttributes[tileID].PaletteNumber, uint8(pixelShade))
		}

		// set the pixel on the screen
		p.ScreenData[i][p.CurrentScanline] = pixelColour
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
		tileID := uint16(p.calculateTileID(tileYIndex, tileXIndex))

		tileAttributes := p.tileAttributes[tileID]

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
			pixelColour = p.colourPalette.GetColour(tileAttributes.PaletteNumber, byte(pixelShade))
		}
		p.ScreenData[i][p.CurrentScanline] = pixelColour
	}

}

// calculateTileID calculates the tile ID for the current scanline
func (p *PPU) calculateTileID(tilemapOffset, lineOffset uint16) int {
	// determine the tile ID
	tileID := int(p.vRAM[0].Read(tilemapOffset + lineOffset - 0x8000))

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
		tilerow := p.tileData[p.vRAMBank][tileID][tilerowIndex]

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
				if !sprite.Priority && p.ScreenData[pixelPos][p.CurrentScanline] != p.SpritePalettes[sprite.UseSecondPalette].GetColour(0) {
					continue
				}
			}

			// skip if pixel is occupied by sprite with lower x coordinate
			if spriteXPerScreen[pixelPos] != 0 && spriteXPerScreen[pixelPos] <= sprite.X {
				continue
			}

			// draw the pixel
			p.ScreenData[pixelPos][p.CurrentScanline] = p.SpritePalettes[sprite.UseSecondPalette].GetColour(uint8(color))

			// mark the pixel as occupied
			spriteXPerScreen[pixelPos] = sprite.X
		}
	}
}

func (p *PPU) ClearRefresh() {
	p.refreshScreen = false
}

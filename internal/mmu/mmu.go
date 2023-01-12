// Package mmu provides a memory management unit for the Game Boy. The
// MMU is unaware of the other components, and handles all the memory
// reads and writes via the IOBus interface.
package mmu

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/thelolagemann/go-gameboy/internal/cartridge"
	"github.com/thelolagemann/go-gameboy/internal/ram"
	"github.com/thelolagemann/go-gameboy/pkg/log"
	"github.com/thelolagemann/go-gameboy/pkg/utils"
)

// IOBus is the interface that the MMU uses to communicate with the other
// components.
type IOBus interface {
	Read(address uint16) uint8
	Write(address uint16, value uint8)
}

// MMU represents the memory management unit of the Game Boy.
// It contains the whole 64kB address space of the Game, separated into
// 12 different memory banks.
type MMU struct {
	biosFinished bool
	bios         []byte
	isMocking    bool

	// 64kB address space
	// (0x0000-0x3FFF) - ROM bank 0
	Cart *cartridge.Cartridge
	// (0x4000-0x7FFF) - ROM bank 1 TODO implement ROM bank switching

	// (0x8000-0x9FFF) - VRAM
	// TODO redirect to video component

	// (0xA000-0xBFFF) - external RAM TODO implement RAM bank switching

	// (0xC000-0xDFFF) - internal RAM
	iRAM ram.RAM

	// (0xE000-0xFDFF) - echo of 8kB internal RAM
	eRAM ram.RAM

	// (0xFE00-0xFE9F) - sprite attribute table (OAM)
	// TODO redirect to video component

	// (0xFEA0-0xFEFF) - unusable memory

	// (0xFF00-0xFF4B) - I/O
	Joypad     IOBus // 0xFF00
	Serial     IOBus // 0xFF01 - 0xFF02
	Timer      IOBus // 0xFF04 - 0xFF07
	Interrupts IOBus // 0xFF0F - 0xFFFF
	Sound      IOBus // 0xFF10 - 0xFF3F
	Video      IOBus // 0xFF40 - 0xFF4B

	// (0xFF4C-0xFF7F) - unusable memory

	// (0xFF80-0xFFFE) - internal RAM
	zRAM ram.RAM

	// (0xFFFF) - interrupt enable register

	mockBank ram.RAM

	Log log.Logger
}

// NewMMU returns a new MMU.
func NewMMU(cart *cartridge.Cartridge, joypad, serial, timer, interrupts, sound IOBus) *MMU {
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	m := &MMU{
		biosFinished: false,
		bios:         []uint8{},
		Cart:         cart,
		iRAM:         ram.NewRAM(0x2000),
		eRAM:         ram.NewRAM(0x1E00),

		zRAM: ram.NewRAM(0x7F),

		Joypad:     joypad,
		Serial:     serial,
		Timer:      timer,
		Interrupts: interrupts,
		Sound:      sound,
		Log:        l,
	}

	// load bios depending on cartridge type
	if cart.Header().Hardware() == "CGB" {
		// TODO load cgb bios
	} else {
	}
	m.bios = gbBios[:]

	return m
}

// AttachVideo attaches the video component to the MMU.
func (m *MMU) AttachVideo(video IOBus) {
	m.Video = video
}

// EnableMock enables the mock bank.
func (m *MMU) EnableMock() {
	m.isMocking = true
	m.mockBank = ram.NewRAM(0xFFFF)
}

// Read returns the value at the given address. It handles all the memory
// banks, mirroring, I/O, etc.
func (m *MMU) Read(address uint16) uint8 {
	if m.isMocking {
		return m.mockBank.Read(address)
	}
	switch {
	// BIOS (0x0000-0x00FF)
	case address <= 0x00FF:
		if !m.biosFinished {
			return m.bios[address]
		}
		return m.Cart.Read(address)
	// ROM (0x0000-0x7FFF)
	case address <= 0x7FFF:
		return m.Cart.Read(address)
	// VRAM (0x8000-0x9FFF)
	case address <= 0x9FFF:
		return m.Video.Read(address)
	// External RAM (0xA000-0xBFFF)
	case address <= 0xBFFF:
		return m.Cart.Read(address)
	// Working RAM (0xC000-0xDFFF)
	case address <= 0xDFFF:
		return m.iRAM.Read(address - 0xC000)
	// Working RAM shadow (0xE000-0xFDFF)
	case address <= 0xFDFF:
		return m.iRAM.Read(address - 0xE000)
	// OAM (0xFE00-0xFE9F)
	case address <= 0xFE9F:
		return m.Video.Read(address)
	// Unusable memory (0xFEA0-0xFEFF)
	case address <= 0xFEFF:
		return 1
	// IO (0xFF00-0xFF3F)
	case address <= 0xFF3F:
		// panic(fmt.Sprintf("unimplemented IO read at 0x%04X", address))
		switch {
		// Joypad (0xFF00)
		case address == 0xFF00:
			return m.Joypad.Read(address)
		// Serial (0xFF01-0xFF02)
		case address == 0xFF01 || address == 0xFF02:
			return m.Serial.Read(address)
		// Timer (0xFF04-0xFF07)
		case address >= 0xFF04 && address <= 0xFF07:
			return m.Timer.Read(address)
		// Sound (0xFF10-0xFF3F)
		case address >= 0xFF10 && address <= 0xFF26:
			return m.Sound.Read(address)
		case address == 0xFF0F:
			return m.Interrupts.Read(address)
		case address == 0xFF03:
			return 0xFF
		default:
			return 0xFF
		}
	// GPU (0xFF40-0xFF4B)
	case address >= 0xFF40 && address <= 0xFF4B:
		return m.Video.Read(address)
	// Unusable memory (0xFF4C-0xFF7F)
	case address <= 0xFF7F:
		return 1
	// Zero page RAM (0xFF80-0xFFFE)
	case address <= 0xFFFE:
		return m.zRAM.Read(address - 0xFF80)
		// InterruptAddress enable register (0xFFFF)
	case address == 0xFFFF:
		return m.Interrupts.Read(address)
	}
	panic(fmt.Sprintf("Invalid address 0x%04X", address))
}

// Read16 returns the 16bit value at the given address.
func (m *MMU) Read16(address uint16) uint16 {
	return uint16(m.Read(address)) | uint16(m.Read(address+1))<<8
}

// Write writes the given value to the given address. It handles all the memory
// banks, mirroring, I/O, etc.
func (m *MMU) Write(address uint16, value uint8) {
	if m.isMocking {
		m.mockBank.Write(address, value)
		return
	}
	// m.Bus.Log().Debugf("mmu\t writing 0x%02X to 0x%04X", value, address)
	switch {
	// ROM (0x0000-0x7FFF)
	case address <= 0x7FFF:
		m.Cart.Write(address, value)
	// VRAM (0x8000-0x9FFF)
	case address <= 0x9FFF:
		m.Video.Write(address, value)
	// External RAM (0xA000-0xBFFF)
	case address <= 0xBFFF:
		m.Cart.Write(address, value)
	// Working RAM (0xC000-0xDFFF)
	case address <= 0xDFFF:
		m.iRAM.Write(address-0xC000, value)
	// Working RAM shadow (0xE000-0xFDFF)
	case address <= 0xFDFF:
		m.iRAM.Write(address-0xE000, value)
	// OAM (0xFE00-0xFE9F)
	case address <= 0xFE9F:
		m.Video.Write(address, value)
	// I/O (0xFF00-0xFF7F)
	case address <= 0xFF7F:
		switch address {
		case 0xFF00:
			m.Joypad.Write(address, value)
		case 0xFF01:
			m.Serial.Write(address, value)
		case 0xFF04, 0xFF05, 0xFF06, 0xFF07:
			m.Timer.Write(address, value)
		case 0xFF0F, 0xFFFF:
			m.Interrupts.Write(address, value)

		case 0xFF10, 0xFF11, 0xFF12, 0xFF13, 0xFF14, 0xFF16, 0xFF17, 0xFF18, 0xFF19, 0xFF1A, 0xFF1B, 0xFF1C, 0xFF1D, 0xFF1E, 0xFF20, 0xFF21, 0xFF22, 0xFF23, 0xFF24, 0xFF25, 0xFF26:
			m.Sound.Write(address, value)
			// waveform RAM
		case 0xFF30, 0xFF31, 0xFF32, 0xFF33, 0xFF34, 0xFF35, 0xFF36, 0xFF37, 0xFF38, 0xFF39, 0xFF3A, 0xFF3B, 0xFF3C, 0xFF3D, 0xFF3E, 0xFF3F:
			m.Sound.Write(address, value)
		case 0xFF40, 0xFF41, 0xFF42, 0xFF43, 0xFF44, 0xFF45, 0xFF47, 0xFF48, 0xFF49, 0xFF4A, 0xFF4B:
			m.Video.Write(address, value)
		case 0xFF46:
			m.doHDMATransfer(value)
		case 0xFF50:
			m.biosFinished = true
		}
	// Zero page RAM (0xFF80-0xFFFE)
	case address <= 0xFFFE:
		m.zRAM.Write(address-0xFF80, value)
	// InterruptAddress enable register (0xFFFF)
	case address == 0xFFFF:
		m.Interrupts.Write(address, value)
	default:
		panic(fmt.Sprintf("mmu\t illegal write to 0x%04X", address))
	}
}

// Write16 writes the given 16bit value to the given address.
func (m *MMU) Write16(address uint16, value uint16) {
	upper, lower := utils.Uint16ToBytes(value)
	m.Write(address, lower)
	m.Write(address+1, upper)
}

// doHDMATransfer performs a DMA transfer from the given address to the PPU's OAM.
func (m *MMU) doHDMATransfer(value uint8) {
	srcAddress := uint16(value) << 8 // src address is value * 100 (shift left 8 bits)
	for i := 0; i < 0xA0; i++ {
		m.Write(0xFE00+uint16(i), m.Read(srcAddress+uint16(i)))
	}
}

// LoadCartridge loads a cartridge from a byte slice into the MMU.
func (m *MMU) LoadCartridge(rom []byte) {
	if len(rom) == 0 {
		m.Cart = cartridge.NewEmptyCartridge()
	} else {
		m.Cart = cartridge.NewCartridge(rom)
	}
}

var gbBios = [0x100]uint8{
	0x31, 0xFE, 0xFF, 0xAF, 0x21, 0xFF, 0x9F, 0x32, 0xCB, 0x7C, 0x20, 0xFB, 0x21, 0x26, 0xFF, 0x0E,
	0x11, 0x3E, 0x80, 0x32, 0xE2, 0x0C, 0x3E, 0xF3, 0xE2, 0x32, 0x3E, 0x77, 0x77, 0x3E, 0xFC, 0xE0,
	0x47, 0x11, 0x04, 0x01, 0x21, 0x10, 0x80, 0x1A, 0xCD, 0x95, 0x00, 0xCD, 0x96, 0x00, 0x13, 0x7B,
	0xFE, 0x34, 0x20, 0xF3, 0x11, 0xD8, 0x00, 0x06, 0x08, 0x1A, 0x13, 0x22, 0x23, 0x05, 0x20, 0xF9,
	0x3E, 0x19, 0xEA, 0x10, 0x99, 0x21, 0x2F, 0x99, 0x0E, 0x0C, 0x3D, 0x28, 0x08, 0x32, 0x0D, 0x20,
	0xF9, 0x2E, 0x0F, 0x18, 0xF3, 0x67, 0x3E, 0x64, 0x57, 0xE0, 0x42, 0x3E, 0x91, 0xE0, 0x40, 0x04,
	0x1E, 0x02, 0x0E, 0x0C, 0xF0, 0x44, 0xFE, 0x90, 0x20, 0xFA, 0x0D, 0x20, 0xF7, 0x1D, 0x20, 0xF2,
	0x0E, 0x13, 0x24, 0x7C, 0x1E, 0x83, 0xFE, 0x62, 0x28, 0x06, 0x1E, 0xC1, 0xFE, 0x64, 0x20, 0x06,
	0x7B, 0xE2, 0x0C, 0x3E, 0x87, 0xE2, 0xF0, 0x42, 0x90, 0xE0, 0x42, 0x15, 0x20, 0xD2, 0x05, 0x20,
	0x4F, 0x16, 0x20, 0x18, 0xCB, 0x4F, 0x06, 0x04, 0xC5, 0xCB, 0x11, 0x17, 0xC1, 0xCB, 0x11, 0x17,
	0x05, 0x20, 0xF5, 0x22, 0x23, 0x22, 0x23, 0xC9, 0xCE, 0xED, 0x66, 0x66, 0xCC, 0x0D, 0x00, 0x0B,
	0x03, 0x73, 0x00, 0x83, 0x00, 0x0C, 0x00, 0x0D, 0x00, 0x08, 0x11, 0x1F, 0x88, 0x89, 0x00, 0x0E,
	0xDC, 0xCC, 0x6E, 0xE6, 0xDD, 0xDD, 0xD9, 0x99, 0xBB, 0xBB, 0x67, 0x63, 0x6E, 0x0E, 0xEC, 0xCC,
	0xDD, 0xDC, 0x99, 0x9F, 0xBB, 0xB9, 0x33, 0x3E, 0x3C, 0x42, 0xB9, 0xA5, 0xB9, 0xA5, 0x42, 0x3C,
	0x21, 0x04, 0x01, 0x11, 0xA8, 0x00, 0x1A, 0x13, 0xBE, 0x20, 0xFE, 0x23, 0x7D, 0xFE, 0x34, 0x20,
	0xF5, 0x06, 0x19, 0x78, 0x86, 0x23, 0x05, 0x20, 0xFB, 0x86, 0x20, 0xFE, 0x3E, 0x01, 0xE0, 0x50,
}

var cgbBios = [0x900]uint8{}

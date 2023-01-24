// Package mmu provides a memory management unit for the Game Boy. The
// MMU is unaware of the other components, and handles all the memory
// reads and writes via the IOBus interface.
package mmu

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/thelolagemann/go-gameboy/internal/boot"
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
	bios         IOBus
	isMocking    bool

	// 64kB address space
	// (0x0000-0x3FFF) - ROM bank 0
	Cart *cartridge.Cartridge
	// (0x4000-0x7FFF) - ROM bank 1 TODO implement ROM bank switching

	// (0x8000-0x9FFF) - VRAM
	// TODO redirect to video component

	// (0xA000-0xBFFF) - external RAM TODO implement RAM bank switching

	// (0xC000-0xCFFF) - internal RAM bank 0 fixed
	iRAM ram.RAM

	// (0xD000-0xDFFF) - internal switchable RAM bank 1 - 7
	wRAM     [7]ram.RAM
	wRAMBank uint8

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

	HDMA *HDMA

	key uint8
}

// NewMMU returns a new MMU.
func NewMMU(cart *cartridge.Cartridge, joypad, serial, timer, interrupts, sound IOBus) *MMU {

	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	l.Formatter = &logrus.TextFormatter{
		DisableColors:    true,
		DisableTimestamp: true,
		DisableSorting:   true,
		DisableQuote:     true,
	}
	m := &MMU{
		biosFinished: false,
		Cart:         cart,
		iRAM:         ram.NewRAM(0x2000),
		eRAM:         ram.NewRAM(0x1E00),
		wRAM: [7]ram.RAM{
			ram.NewRAM(0x1000),
			ram.NewRAM(0x1000),
			ram.NewRAM(0x1000),
			ram.NewRAM(0x1000),
			ram.NewRAM(0x1000),
			ram.NewRAM(0x1000),
			ram.NewRAM(0x1000),
		},

		zRAM: ram.NewRAM(0x7F),

		Joypad:     joypad,
		Serial:     serial,
		Timer:      timer,
		Interrupts: interrupts,
		Sound:      sound,
		Log:        l,
	}

	m.HDMA = NewHDMA(m)

	// load boot depending on cartridge type
	if cart.Header().Hardware() == "CGB" {
		// TODO load cgb boot
		m.bios = boot.NewBootROM(boot.CGBBootROM[:], boot.CGBBootROMChecksum)
	} else {
		// load dmg boot
		m.bios = boot.NewBootROM(boot.DMGBootROM[:], boot.DMBBootROMChecksum)
	}

	return m
}

func (m *MMU) Key() uint8 {
	return m.key
}

func (m *MMU) SetKey(key uint8) {
	m.key = key
}

// AttachVideo attaches the video component to the MMU.
func (m *MMU) AttachVideo(video IOBus) {
	m.Video = video
}

func (m *MMU) IsGBC() bool {
	return m.Cart.Header().Hardware() == "CGB"
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
	// BIOS (0x0000-0x0900) // TODO handle CGB bios (0x900 bytes)
	case address <= 0x0900:
		if !m.biosFinished && address < 0x0100 || !m.biosFinished && m.Cart.Header().Hardware() == "CGB" && address >= 0x200 && address < 0x900 {
			return m.bios.Read(address)
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
	// Working RAM (0xC000-0xCFFF)
	case address <= 0xDFFF:
		if m.IsGBC() && address >= 0xD000 && address <= 0xDFFF {
			return m.wRAM[m.wRAMBank].Read(address - 0xD000)
		}
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
	case address == 0xFF4C, address == 0xFF4E, address == 0xFF50:
		return 0xFF
	case address == 0xFF4D:
		if m.IsGBC() {
			return m.key | 0x7e
		} else {
			return 0xFF
		}
		// HDMA (0xFF51-0xFF55)
	case address >= 0xFF51 && address <= 0xFF55:
		return m.HDMA.Read(address)
	// GPU CGB (0xFF4F-0xFF70)
	case address == 0xFF4F || address >= 0xFF68 && address <= 0xFF6B:
		return m.Video.Read(address)

	// Unusable memory (0xFF4C-0xFF7F)
	case address <= 0xFF7F:
		return 0xFF
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
	case address <= 0xCFFF:
		m.iRAM.Write(address-0xC000, value)
	// Working RAM (0xD000-0xDFFF) (switchable bank 1-7)
	case address <= 0xDFFF:
		if m.IsGBC() {
			m.wRAM[m.wRAMBank].Write(address-0xD000, value)
		} else {
			m.iRAM.Write(address-0xD000, value)
		}
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
		case 0xFF40, 0xFF41, 0xFF42, 0xFF43, 0xFF44, 0xFF45, 0xFF46, 0xFF47, 0xFF48, 0xFF49, 0xFF4A, 0xFF4B:
			m.Video.Write(address, value)
		case 0xFF4F, 0xFF68, 0xFF69, 0xFF6A, 0xFF6B:
			m.Video.Write(address, value)
		case 0xFF50:
			m.biosFinished = true
		case 0xFF4D:
			if m.IsGBC() {
				m.key = value
			}
		case 0xFF51, 0xFF52, 0xFF53, 0xFF54, 0xFF55:
			m.HDMA.Write(address, value)
		case 0xFF70:
			if m.IsGBC() {
				m.wRAMBank = value & 0b111 // first 3 bits
				if m.wRAMBank != 0 {
					m.wRAMBank-- // 0 is bank 1
				}
			}

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

// LoadCartridge loads a cartridge from a byte slice into the MMU.
func (m *MMU) LoadCartridge(rom []byte) {
	if len(rom) == 0 {
		m.Cart = cartridge.NewEmptyCartridge()
	} else {
		m.Cart = cartridge.NewCartridge(rom)
	}
}

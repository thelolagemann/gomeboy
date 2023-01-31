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
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/thelolagemann/go-gameboy/internal/types/registers"
	"github.com/thelolagemann/go-gameboy/pkg/log"
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
	bootROM      *boot.ROM
	isMocking    bool

	// 64kB address space
	// (0x0000-0x3FFF) - ROM bank 0
	Cart *cartridge.Cartridge
	// (0x4000-0x7FFF) - ROM bank 1 TODO implement ROM bank switching

	// (0x8000-0x9FFF) - VRAM
	// TODO redirect to video component

	// (0xA000-0xBFFF) - external RAM TODO implement RAM bank switching

	// (0xC000-0xCFFF) - internal RAM bank 0 fixed

	// (0xD000-0xDFFF) - internal switchable RAM bank 1 - 7
	wRAM     [8]*ram.Ram
	wRAMBank uint8

	// (0xFE00-0xFE9F) - sprite attribute table (OAM)
	// TODO redirect to video component

	// (0xFEA0-0xFEFF) - unusable memory

	// (0xFF00-0xFF4B) - I/O
	Serial IOBus // 0xFF01 - 0xFF02
	Sound  IOBus // 0xFF10 - 0xFF3F
	Video  IOBus // 0xFF40 - 0xFF4B

	// (0xFF4C-0xFF7F) - unusable memory

	// (0xFF80-0xFFFE) - internal RAM
	zRAM *ram.Ram

	// (0xFFFF) - interrupt enable register

	mockBank ram.RAM

	Log log.Logger

	HDMA *HDMA

	key0  uint8
	key1  uint8
	isGBC bool
}

func (m *MMU) init() {
	// setup registers
	// CGB registers
	if m.IsGBC() {
		registers.RegisterHardware(
			registers.KEY0,
			func(v uint8) {
				m.key0 = v & 0xf // only lower nibble is writable
			}, func() uint8 {
				return m.key0
			})
		registers.RegisterHardware(
			registers.SVBK,
			func(v uint8) {
				v &= 0x07 // only 3 bits are used
				if v == 0 {
					v = 1
				}
				m.wRAMBank = v
			},
			func() uint8 {
				return m.wRAMBank
			},
		)
	}
}

// NewMMU returns a new MMU.
func NewMMU(cart *cartridge.Cartridge, serial, sound IOBus) *MMU {
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
		wRAM: [8]*ram.Ram{
			ram.NewRAM(0x1000),
			ram.NewRAM(0x1000),
			ram.NewRAM(0x1000),
			ram.NewRAM(0x1000),
			ram.NewRAM(0x1000),
			ram.NewRAM(0x1000),
			ram.NewRAM(0x1000),
			ram.NewRAM(0x1000),
		},

		zRAM: ram.NewRAM(0x80), // 128 bytes

		Serial: serial,
		Sound:  sound,
		Log:    l,
		isGBC:  cart.Header().Hardware() == "CGB",
	}

	// load boot depending on cartridge type
	if cart.Header().Hardware() == "CGB" {
		m.bootROM = boot.NewBootROM(boot.CGBBootROM[:], boot.CGBBootROMChecksum)
		m.HDMA = NewHDMA(m)
	} else {
		// load dmg boot
		m.bootROM = boot.NewBootROM(boot.DMGBootROM[:], boot.DMBBootROMChecksum)
	}

	m.init()

	return m
}

func (m *MMU) Key() uint8 {
	return m.key1
}

func (m *MMU) SetKey(key uint8) {
	m.key1 = key
}

// AttachVideo attaches the video component to the MMU.
func (m *MMU) AttachVideo(video IOBus) {
	m.Video = video
}

func (m *MMU) IsGBC() bool {
	return m.isGBC
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
	if address >= 0xFF00 && registers.Has(address) {
		return registers.HardwareRegisters[address].Read()
	}
	switch {
	// BOOT ROM / ROM (0x0000-0x7FFF)
	case address <= 0x7FFF:
		if !m.biosFinished {
			if address < 0x100 {
				return m.bootROM.Read(address)
			}
			// CGB boot ROM is 0x900 bytes long, with a gap of 0x100 bytes
			// at 0x100 - 0x1FF, to read the cartridge header.
			if m.Cart.Header().Hardware() == "CGB" && address >= 0x200 && address < 0x900 {
				return m.bootROM.Read(address)
			}
		}
		return m.Cart.Read(address)
	// VRAM (0x8000-0x9FFF)
	case address >= 0x8000 && address <= 0x9FFF:
		return m.Video.Read(address)
	// External RAM (0xA000-0xBFFF)
	case address >= 0xA000 && address <= 0xBFFF:
		return m.Cart.Read(address)
	// WRAM (Bank 0) (0xC000-0xCFFF)
	case address >= 0xC000 && address <= 0xCFFF:
		return m.wRAM[0].Read(address - 0xC000)
	// WRAM (Bank 1 / 1-7 (CGB)) (0xD000-0xDFFF)
	case address >= 0xD000 && address <= 0xDFFF:
		if m.IsGBC() {
			return m.wRAM[m.wRAMBank].Read(address - 0xD000)
		}
		return m.wRAM[1].Read(address - 0xD000)
	// WRAM (Bank 0 / Echo) (0xE000-0xEFFF)
	case address >= 0xE000 && address <= 0xFDFF:
		// which bank is being read from?
		if address >= 0xE000 && address <= 0xEFFF {
			return m.wRAM[0].Read(address & 0x0FFF)
		} else if address >= 0xF000 && address <= 0xFDFF {
			return m.wRAM[1].Read(address & 0x0FFF)
		}
	// OAM (0xFE00-0xFE9F)
	case address >= 0xFE00 && address <= 0xFE9F:
		return m.Video.Read(address)
	// Unusable memory (0xFEA0-0xFEFF)
	case address >= 0xFEA0 && address <= 0xFEFF:
		m.Log.Errorf("unhandled read from address %04x", address)
		return 0xff
	// IO (0xFF00-0xFF3F)
	case address >= 0xFF00 && address <= 0xFF3F:
		// panic(fmt.Sprintf("unimplemented IO read at 0x%04X", address))
		switch {
		// Serial (0xFF01-0xFF02)
		case address == 0xFF01 || address == 0xFF02:
			return m.Serial.Read(address)
		// Sound (0xFF10-0xFF3F)
		case address >= 0xFF10 && address <= 0xFF3F:
			return m.Sound.Read(address)
		default:
			fmt.Printf("warning: unimplemented IO read at 0x%04X\n", address)
			return 0xFF
		}
	// GPU (0xFF40-0xFF4B)
	case address == 0xFF4E, address == 0xFF50:
		return 0xFF
	case address == 0xFF4D:
		if m.IsGBC() {
			return m.key1 | 0x7e
		} else {
			return 0xFF
		}
	// Zero page RAM (0xFF80-0xFFFE)
	case address >= 0xFF80 && address <= 0xFFFE:
		return m.zRAM.Read(address - 0xFF80)
	// all of the CGB registers that should return 0xFF when in DMG mode
	case address == 0xFF4C || address == 0xFF4F || address <= 0xFF7F:
		return 0xFF
	}
	panic(fmt.Sprintf("Invalid address 0x%04X", address))
}

// Write writes the given value to the given address. It handles all the memory
// banks, mirroring, I/O, etc.
func (m *MMU) Write(address uint16, value uint8) {
	if m.isMocking {
		m.mockBank.Write(address, value)
		return
	}
	// is it a hardware register?
	if address >= 0xFF00 && registers.Has(address) {
		registers.HardwareRegisters[address].Write(value)
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
	case address >= 0xC000 && address <= 0xCFFF:
		m.wRAM[0].Write(address-0xC000, value)
	// Working RAM (0xD000-0xDFFF) (switchable bank 1-7)
	case address >= 0xD000 && address <= 0xDFFF:
		if m.IsGBC() {
			m.wRAM[m.wRAMBank].Write(address-0xD000, value)
		} else {
			m.wRAM[1].Write(address-0xD000, value)
		}
	// Working RAM shadow (0xE000-0xFDFF)
	case address >= 0xE000 && address <= 0xFDFF:
		m.Log.Errorf("writing to shadow RAM at 0x%04X", address)
		// which bank is being written to?
		if address >= 0xE000 && address <= 0xEFFF {
			m.wRAM[0].Write(address&0x0FFF, value)
		} else if address >= 0xF000 && address <= 0xFDFF {
			m.wRAM[1].Write(address&0x0FFF, value) // TODO how does GBC handle this? (bank 1-7)
		}
	// OAM (0xFE00-0xFE9F)
	case address >= 0xFE00 && address <= 0xFE9F:
		m.Video.Write(address, value)
	// I/O (0xFF00-0xFF7F)
	case address <= 0xFF7F:
		switch address {
		case 0xFF01:
			m.Serial.Write(address, value)
		case 0xFF10, 0xFF11, 0xFF12, 0xFF13, 0xFF14, 0xFF15, 0xFF16, 0xFF17, 0xFF18, 0xFF19, 0xFF1A, 0xFF1B, 0xFF1C, 0xFF1D, 0xFF1E, 0xFF1F, 0xFF20, 0xFF21, 0xFF22, 0xFF23, 0xFF24, 0xFF25, 0xFF26:
			m.Sound.Write(address, value)
			// waveform RAM
		case 0xFF30, 0xFF31, 0xFF32, 0xFF33, 0xFF34, 0xFF35, 0xFF36, 0xFF37, 0xFF38, 0xFF39, 0xFF3A, 0xFF3B, 0xFF3C, 0xFF3D, 0xFF3E, 0xFF3F:
			m.Sound.Write(address, value)
		case 0xFF50:
			m.biosFinished = true
		case 0xFF4D:
			if m.IsGBC() {
				m.key1 |= value & uint8(types.Bit0)
			}
		default:
			fmt.Printf("warning: unimplemented IO write at 0x%04X\n", address)
		}
	// Zero page RAM (0xFF80-0xFFFE)
	case address >= 0xFF80 && address <= 0xFFFE:
		m.zRAM.Write(address-0xFF80, value)
	default:
		panic(fmt.Sprintf("mmu\t illegal write to 0x%04X", address))
	}
}

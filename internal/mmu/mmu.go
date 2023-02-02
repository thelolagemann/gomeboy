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

	bootROM   *boot.ROM
	isMocking bool

	// 64kB address space
	raw [65536]*types.Address
	// (0x0000-0x3FFF) - ROM bank 0
	Cart *cartridge.Cartridge
	// (0x4000-0x7FFF) - ROM bank 1 TODO implement ROM bank switching

	// (0x8000-0x9FFF) - VRAM
	// TODO redirect to video component

	// (0xA000-0xBFFF) - external RAM TODO implement RAM bank switching

	// (0xC000-0xCFFF) - internal RAM bank 0 fixed

	// (0xD000-0xDFFF) - internal switchable RAM bank 1 - 7
	wRAM *WRAM

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
	registers.RegisterHardware(
		registers.BDIS,
		func(v uint8) {
			// it's assumed any write to this register will disable the boot rom
			m.biosFinished = true
		}, registers.NoRead)
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
			registers.KEY1,
			func(v uint8) {
				m.key1 |= v & types.Bit0 // only lower bit is writable
			}, func() uint8 {
				return m.key1 | 0x7e // upper bits are always set
			},
		)
	}

	// setup raw memory

	// 0x0000 - 0x7FFF - ROM (16kB)
	for i := 0x0000; i < 0x8000; i++ {
		m.raw[i] = &types.Address{
			Read:  m.readCart,
			Write: m.Cart.Write,
		}
	}

	// 0xA000 - 0xBFFF - external RAM (8kB)
	for i := 0xA000; i < 0xC000; i++ {
		m.raw[i] = &types.Address{
			Read:  m.Cart.Read,
			Write: m.Cart.Write,
		}
	}

	// 0xC000 - 0xDFFF - internal RAM (8kB)
	for i := 0xC000; i < 0xFE00; i++ {
		m.raw[i] = &types.Address{
			Read:  m.wRAM.Read,
			Write: m.wRAM.Write,
		}
	}

	// 0xFEA0 - 0xFEFF - unusable memory (96B)
	for i := 0xFEA0; i < 0xFF00; i++ {
		m.raw[i] = &types.Address{
			Read: types.Unreadable,
		}
	}

	// 0xFF80 - 0xFFFE - Zero Page RAM (127B)
	for i := 0xFF80; i < 0xFFFF; i++ {
		m.raw[i] = &types.Address{
			Read:  readOffset(m.zRAM.Read, 0xFF80),
			Write: writeOffset(m.zRAM.Write, 0xFF80),
		}
	}

	// 0xFFFF - interrupt enable register
	m.raw[0xFFFF] = &types.Address{
		Read:  registers.Read,
		Write: registers.Write,
	}
}

func readOffset(read func(uint16) uint8, offset uint16) func(uint16) uint8 {
	return func(addr uint16) uint8 {
		return read(addr - offset)
	}
}

func writeOffset(write func(uint16, uint8), offset uint16) func(uint16, uint8) {
	return func(addr uint16, v uint8) {
		write(addr-offset, v)
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
		wRAM:         NewWRAM(),

		zRAM: ram.NewRAM(0x80), // 128 bytes

		Serial: serial,
		Sound:  sound,
		Log:    l,
		isGBC:  cart.Header().Hardware() == "CGB",
	}
	m.HDMA = NewHDMA(m)

	// load boot depending on cartridge type
	if cart.Header().Hardware() == "CGB" {
		m.bootROM = boot.NewBootROM(boot.CGBBootROM[:], boot.CGBBootROMChecksum)
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

	// 0x8000 - 0x9FFF - VRAM (8kB)
	for i := 0x8000; i < 0xA000; i++ {
		m.raw[i] = &types.Address{
			Read:  m.Video.Read,
			Write: m.Video.Write,
		}
	}

	// 0xFE00 - 0xFE9F - sprite attribute table (OAM) (160B)
	for i := 0xFE00; i < 0xFEA0; i++ {
		m.raw[i] = &types.Address{
			Read:  m.Video.Read,
			Write: m.Video.Write,
		}
	}

	// 0xFF00 - 0xFF7F - I/O (128B) (Needs to be registered after the video component is attached)
	for i := 0xFF00; i < 0xFF80; i++ {
		if registers.Has(registers.HardwareAddress(i)) {
			m.raw[i] = &types.Address{
				Read:  registers.Read,
				Write: registers.Write,
			}
		} else {
			m.raw[i] = &types.Address{
				Read: types.Unreadable,
			}
		}
	}

	// 0xFF30 - 0xFF3F - Wave Pattern RAM (16B)
	for i := 0xFF30; i < 0xFF40; i++ {
		m.raw[i] = &types.Address{
			Read:  m.Sound.Read,
			Write: m.Sound.Write,
		}
	}
}

func (m *MMU) IsGBC() bool {
	return m.isGBC
}

// EnableMock enables the mock bank.
func (m *MMU) EnableMock() {
	m.isMocking = true
	m.mockBank = ram.NewRAM(0xFFFF)
}

func (m *MMU) readCart(address uint16) uint8 {
	if !m.biosFinished {
		if address < 0x100 {
			return m.bootROM.Read(address)
		}
		if m.isGBC && address >= 0x200 && address < 0x900 {
			return m.bootROM.Read(address)
		}
	}

	return m.Cart.Read(address)
}

// Read returns the value at the given address. It handles all the memory
// banks, mirroring, I/O, etc.
func (m *MMU) Read(address uint16) uint8 {
	return m.raw[address].Read(address)
	switch {
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
	}
	panic(fmt.Sprintf("Invalid address 0x%04X", address))
}

func (m *MMU) Write(address uint16, value uint8) {
	if m.raw[address].Write == nil {
		return
	}
	m.raw[address].Write(address, value)
}

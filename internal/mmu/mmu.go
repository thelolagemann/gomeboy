// Package mmu provides a memory management unit for the Game Boy. The
// MMU is unaware of the other components, and handles all the memory
// reads and writes via the IOBus interface.
package mmu

import (
	"github.com/sirupsen/logrus"
	"github.com/thelolagemann/go-gameboy/internal/boot"
	"github.com/thelolagemann/go-gameboy/internal/cartridge"
	"github.com/thelolagemann/go-gameboy/internal/ram"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/thelolagemann/go-gameboy/pkg/log"
)

// IOBus is the interface that the MMU uses to communicate with the other
// components.
type IOBus interface {
	Read(address uint16) uint8
	Write(address uint16, value uint8)
}

// MMU is the memory management unit for the Game Boy. It handles all
// memory reads and writes to the Game Boy's 64kB of memory, and
// delegates to the other components through the IOBus interface.
type MMU struct {
	// 64kB address space
	raw [65536]*types.Address

	// 0x0000 - 0x00FF/0x0900 - BOOT ROM (256B/2304B)
	bootROM     *boot.ROM
	bootROMDone bool

	// 0x0000 - 0x7FFF - ROM (16kB)
	// 0xA000 - 0xBFFF - External RAM (8kB)
	Cart *cartridge.Cartridge

	// 0x8000 - 0x9FFF - Video RAM (8kB)
	// 0xFE00 - 0xFE9F - Sprite Attribute Table (160B)
	Video IOBus

	// 0xC000 - 0xDFFF - Work RAM (8kB)
	// 0xE000 - 0xFDFF - Echo RAM (7.5kB)
	wRAM *WRAM

	// 0xFF00 - 0xFF7F - I/O Registers
	registers types.HardwareRegisters

	// 0xFF30 - 0xFF3F - Wave Pattern RAM (16B)
	Sound IOBus

	// 0xFF80 - 0xFFFE - Zero Page RAM (127B)
	zRAM *ram.RAM

	// (0xFFFF) - interrupt enable register

	Log log.Logger

	HDMA *HDMA

	key0  uint8
	key1  uint8
	isGBC bool
}

func (m *MMU) init() {
	// setup registers
	types.RegisterHardware(
		types.BDIS,
		func(v uint8) {
			// it's assumed any write to this register will disable the boot rom
			m.bootROMDone = true
		}, types.NoRead)
	// CGB registers
	if m.IsGBC() {
		types.RegisterHardware(
			types.KEY0,
			func(v uint8) {
				m.key0 = v & 0xf // only lower nibble is writable
			}, func() uint8 {
				return m.key0
			})
		types.RegisterHardware(
			types.KEY1,
			func(v uint8) {
				m.key1 |= v & types.Bit0 // only lower bit is writable
			}, func() uint8 {
				return m.key1 | 0x7e // upper bits are always set
			},
		)
	}

	// setup raw memory
	addresses := []types.Address{
		{Read: m.readCart, Write: m.Cart.Write},
		{Read: m.Cart.Read, Write: m.Cart.Write},
		{Read: m.wRAM.Read, Write: m.wRAM.Write},
		{Read: readOffset(m.zRAM.Read, 0xFF80), Write: writeOffset(m.zRAM.Write, 0xFF80)},
		{Read: func(address uint16) uint8 {
			return 0xff
		}},
	}

	// 0x0000 - 0x7FFF - ROM (16kB)
	for i := 0x0000; i < 0x8000; i++ {
		if i <= 0x900 {
			m.raw[i] = &addresses[0]
		} else {
			m.raw[i] = &addresses[1]
		}
	}

	// 0xA000 - 0xBFFF - external RAM (8kB)
	for i := 0xA000; i < 0xC000; i++ {
		m.raw[i] = &addresses[1]
	}

	// 0xC000 - 0xDFFF - internal RAM (8kB)
	for i := 0xC000; i < 0xFE00; i++ {
		m.raw[i] = &addresses[2]
	}

	// 0xFEA0 - 0xFEFF - unusable memory (96B)
	for i := 0xFEA0; i < 0xFF00; i++ {
		m.raw[i] = &addresses[2]
	}

	// 0xFF80 - 0xFFFE - Zero Page RAM (127B)
	for i := 0xFF80; i < 0xFFFF; i++ {
		m.raw[i] = &addresses[3]
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
func NewMMU(cart *cartridge.Cartridge, sound IOBus) *MMU {
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	l.Formatter = &logrus.TextFormatter{
		DisableColors:    true,
		DisableTimestamp: true,
		DisableSorting:   true,
		DisableQuote:     true,
	}
	m := &MMU{
		Cart: cart,
		wRAM: NewWRAM(),

		zRAM: ram.NewRAM(0x80), // 128 bytes

		Sound: sound,
		Log:   l,
		isGBC: cart.Header().Hardware() == "CGB",
	}

	if cart.Header().Hardware() == "CGB" {
		// load boot depending on cartridge type
		m.HDMA = NewHDMA(m)
		// m.bootROM = boot.NewBootROM(boot.CGBBootROM[:], boot.CGBBootROMChecksum)
	} else {
		// load dmg boot
		// m.bootROM = boot.NewBootROM(boot.DMGBootROM[:], boot.DMBBootROMChecksum)
	}

	m.init()

	return m
}

func (m *MMU) SetBootROM(rom []byte) {
	m.bootROM = boot.LoadBootROM(rom)
	if len(rom) == 0x900 {
		m.isGBC = true
		m.HDMA = NewHDMA(m)
	}
}

func (m *MMU) SetModel(model uint8) {
	switch model {
	case 1:
		m.isGBC = false
	case 2:
		m.isGBC = true
		m.HDMA = NewHDMA(m)
	}
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

	// collect hardware registers
	m.registers = types.CollectHardwareRegisters()

	addresses := []types.Address{
		{Read: m.Video.Read, Write: m.Video.Write},
		{Read: m.registers.Read, Write: m.registers.Write},
		{Read: m.Sound.Read, Write: m.Sound.Write},
	}

	// 0xFF00 - 0xFF7F - I/O (128B)
	for i := 0xFF00; i < 0xFF80; i++ {
		m.raw[i] = &addresses[1]
	}
	// 0xFFFF - interrupt enable register
	m.raw[0xFFFF] = &types.Address{
		Read:  m.registers.Read,
		Write: m.registers.Write,
	}

	// 0x8000 - 0x9FFF - VRAM (8kB)
	for i := 0x8000; i < 0xA000; i++ {
		m.raw[i] = &addresses[0]
	}

	// 0xFE00 - 0xFE9F - sprite attribute table (OAM) (160B)
	for i := 0xFE00; i < 0xFEA0; i++ {
		m.raw[i] = &addresses[0]
	}

	// 0xFF30 - 0xFF3F - Wave Pattern RAM (16B)
	for i := 0xFF30; i < 0xFF40; i++ {
		m.raw[i] = &addresses[2]
	}
}

func (m *MMU) IsGBC() bool {
	return m.isGBC
}

func (m *MMU) readCart(address uint16) uint8 {
	// handle the boot ROM (if enabled)
	if m.bootROM != nil && !m.bootROMDone {
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
}

func (m *MMU) Write(address uint16, value uint8) {
	m.raw[address].Write(address, value)
}

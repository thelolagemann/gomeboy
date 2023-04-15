// Package mmu provides a memory management unit for the Game Boy. The
// MMU is unaware of the other components, and handles all the memory
// reads and writes via the IOBus interface.
package mmu

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/boot"
	"github.com/thelolagemann/go-gameboy/internal/cartridge"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/thelolagemann/go-gameboy/pkg/log"
	"reflect"
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
	BootROM     *boot.ROM
	bootROMDone bool

	// 0x0000 - 0x7FFF - ROM (16kB)
	// 0xA000 - 0xBFFF - External RAM (8kB)
	Cart *cartridge.Cartridge

	// 0x8000 - 0x9FFF - Video RAM (8kB)
	// 0xFE00 - 0xFE9F - Sprite Attribute Table (160B)
	Video IOBus

	// 0xC000 - 0xDFFF - Work RAM (8kB)
	// 0xE000 - 0xFDFF - Echo RAM (7.5kB)
	wRAM []uint8

	// 0xFF00 - 0xFF7F - I/O Registers
	registers types.HardwareRegisters

	// 0xFF30 - 0xFF3F - Wave Pattern RAM (16B)
	Sound IOBus

	// 0xFF80 - 0xFFFE - Zero Page RAM (127B)

	// (0xFFFF) - interrupt enable register

	Log log.Logger

	key0        uint8
	key1        uint8
	isGBCCompat bool
	isGBC       bool

	flatMemory []uint8

	loggedReads []int
	IsMBC1      bool
	wRAMOffset  uint16
	wRAMBank    uint8
	// undocumented registers
	ff72, ff73, ff74, ff75 uint8
}

func (m *MMU) init() {
	// setup registers
	types.RegisterHardware(
		types.BDIS,
		func(v uint8) {
			// it's assumed any write to this register will disable the boot rom
			m.bootROMDone = true
			//m.Log.Infof("boot ROM disabled with write to BDIS register: %v", v)
		}, func() uint8 {
			// TODO return different values depending on hardware (DMG/SGB/CGB)
			if m.bootROMDone {

				return 0xFF

			}
			return 0x00
		})
	// CGB registers

	types.RegisterHardware(
		types.KEY0,
		func(v uint8) {
			if m.isGBC && !m.bootROMDone { // only r/w when boot rom is running TODO: verify this
				m.key0 = v & 0b0000_1101
			}
		}, func() uint8 {
			if m.isGBC && !m.bootROMDone {
				return m.key0 | 0b1111_0010
			}
			return 0xFF
		})
	types.RegisterHardware(
		types.KEY1,
		func(v uint8) {
			if m.isGBC {
				m.key1 = v & types.Bit0 // only lower bit is writable
			}
		}, func() uint8 {
			if m.isGBC {
				return m.key1 | 0x7e // upper bits are always set
			}
			return 0xFF
		},
	)
	types.RegisterHardware(
		types.SVBK,
		func(v uint8) {
			if m.isGBC {
				m.wRAMBank = v & 0x07 // only lower 3 bits are writable
				if m.wRAMBank == 0 {
					m.wRAMBank = 1
				}
				m.wRAMOffset = uint16(m.wRAMBank) * 4096
			}
		}, func() uint8 {
			if m.isGBC {
				return m.wRAMBank | 0b1111_1000 // upper bits are always set
			}
			return 0xFF
		},
	)
	types.RegisterHardware(
		types.FF72,
		func(v uint8) {
			if m.isGBCCompat {
				m.ff72 = v
			}
		}, func() uint8 {
			if m.isGBCCompat {
				return m.ff72
			}
			return 0xFF
		},
	)
	types.RegisterHardware(
		types.FF73,
		func(v uint8) {
			if m.isGBCCompat {
				m.ff73 = v
			}
		},
		func() uint8 {
			if m.isGBCCompat {
				return m.ff73
			}
			return 0xFF
		},
	)
	types.RegisterHardware(
		types.FF74,
		func(v uint8) {
			if m.isGBCCompat {
				m.ff74 = v
			}
		},
		func() uint8 {
			return 0xFF // always returns 0xFF TODO: verify this
		},
	)
	types.RegisterHardware(
		types.FF75,
		func(v uint8) {
			if m.isGBCCompat {
				// only bits 4-6 are writable
				m.ff75 = v & 0b0111_0000
			}
		},
		func() uint8 {
			if m.isGBCCompat {
				return m.ff75 | 0b1000_1111
			}
			return 0xFF
		},
	)

}

// NewMMU returns a new MMU.
func NewMMU(cart *cartridge.Cartridge, sound IOBus) *MMU {
	m := &MMU{
		Cart: cart,
		Log:  log.New(),

		Sound:       sound,
		isGBCCompat: cart.Header().Hardware() == "CGB",
		bootROMDone: true, // only set to false if boot rom is enabled
		flatMemory:  make([]uint8, 65536),

		loggedReads: make([]int, 65536),
		IsMBC1:      reflect.TypeOf(cart.MemoryBankController) == reflect.TypeOf(&cartridge.MemoryBankedCartridge1{}),
		wRAM:        make([]uint8, 32768),
		wRAMOffset:  0x4000,
		wRAMBank:    1,
	}

	m.init()

	return m
}

func (m *MMU) SetLogger(l log.Logger) {
	m.Log = l
}

// Map is to be called after all components have been initialized.
// This will map the memory addresses to the correct components.
func (m *MMU) Map() {
	// setup raw memory
	addresses := []types.Address{
		{Read: m.readCart, Write: m.Cart.Write},
		{Read: m.Cart.Read, Write: m.Cart.Write},
		{Read: nil, Write: m.Cart.Write},
	}

	if m.BootROM != nil {
		addresses[0] = types.Address{Read: m.readCart, Write: m.Cart.Write}
	}

	// 0x4000 - 0x7FFF - ROM (16kB)
	for i := 0x0000; i < 0x8000; i++ {
		if i <= 0x900 {
			m.raw[i] = &addresses[0]
		} else {
			m.raw[i] = &addresses[1]
		}
		if i < 0x4000 {
			m.raw[i] = &addresses[2]
			m.flatMemory[i] = m.Cart.Read(uint16(i))
		}
	}

	// 0xA000 - 0xBFFF - external RAM (8kB)
	for i := 0xA000; i < 0xC000; i++ {
		m.raw[i] = &addresses[1]
	}

	// 0xFEA0 - 0xFEFF - unusable memory (96B)
	for i := 0xFEA0; i < 0xFF00; i++ {
		m.raw[i] = &types.Address{
			Read: func(uint16) uint8 { return 0xFF },
			Write: func(uint16, uint8) {
				//m.Log.Errorf("write to unusable memory at 0x%04X", i)
			},
		}
	}

	// collect hardware registers
	m.registers = types.CollectHardwareRegisters()

	addresses2 := []types.Address{
		{Read: m.Video.Read, Write: m.Video.Write},
		{Read: m.registers.Read, Write: m.registers.Write},
		{Read: m.Sound.Read, Write: m.Sound.Write},
	}

	// 0xFF00 - 0xFF7F - I/O (128B)
	for i := 0xFF00; i < 0xFF80; i++ {
		m.raw[i] = &addresses2[1]
	}
	// 0xFFFF - interrupt enable register
	m.raw[0xFFFF] = &types.Address{
		Read:  m.registers.Read,
		Write: m.registers.Write,
	}

	// 0x8000 - 0x9FFF - VRAM (8kB)
	for i := 0x8000; i < 0xA000; i++ {
		m.raw[i] = &addresses2[0]
	}

	// 0xFE00 - 0xFE9F - sprite attribute table (OAM) (160B)
	for i := 0xFE00; i < 0xFEA0; i++ {
		m.raw[i] = &addresses2[0]
	}

	// 0xFF30 - 0xFF3F - Wave Pattern RAM (16B)
	for i := 0xFF30; i < 0xFF40; i++ {
		m.raw[i] = &addresses2[2]
	}

	m.isGBC = m.IsGBC()
}

func (m *MMU) SetBootROM(rom []byte) {
	m.BootROM = boot.LoadBootROM(rom)
	m.bootROMDone = false
	if len(rom) == 0x900 {
		m.isGBCCompat = true
	} else {
		m.isGBCCompat = false
	}
}

func (m *MMU) SetModel(model types.Model) {
	switch model {
	case types.DMG0, types.DMGABC:
		m.isGBCCompat = false
	case types.CGB0, types.CGBABC:
		m.isGBCCompat = true
	}
}

func (m *MMU) Key() uint8 {
	return m.key1
}

func (m *MMU) SetKey(key uint8) {
	m.key1 = key
}

func (m *MMU) IsBootROMDone() bool {
	return m.bootROMDone
}

// AttachVideo attaches the video component to the MMU.
func (m *MMU) AttachVideo(video IOBus) {
	m.Video = video
}

func (m *MMU) IsGBCCompat() bool {
	return m.isGBCCompat
}

func (m *MMU) IsGBC() bool {
	return m.isGBCCompat && m.Cart.Header().GameboyColor()
}

func (m *MMU) readCart(address uint16) uint8 {
	// handle the boot ROM (if enabled)
	if m.BootROM != nil && !m.bootROMDone {
		if address < 0x100 {
			// fmt.Printf("read from boot rom at 0x%04X\n", address)
			return m.BootROM.Read(address)
		}
		if m.isGBCCompat && address >= 0x200 && address < 0x900 {
			return m.BootROM.Read(address)
		}
	}

	return m.Cart.Read(address)
}

// Read returns the value at the given address. It handles all the memory
// banks, mirroring, I/O, etc.
func (m *MMU) Read(address uint16) uint8 {
	if address >= 0xFF00 && address < 0xFF80 && address != 0xFF44 {
		//fmt.Printf("read from 0x%04X\n", address)
	}
	// m.loggedReads[address]++
	switch {
	case address < 0x4000:
		return m.readCart(address)
	case address >= 0xC000 && address < 0xD000:
		return m.wRAM[address-0xC000]
	case address >= 0xD000 && address < 0xE000:
		return m.wRAM[m.wRAMOffset+(address-0xD000)]
	case address >= 0xE000 && address < 0xF000:
		return m.wRAM[address-0xE000]
	case address >= 0xF000 && address < 0xFE00:
		return m.wRAM[m.wRAMOffset+(address-0xF000)]
	case address >= 0xFF80 && address < 0xFFFF:
		return m.flatMemory[address]
	}
	return m.raw[address].Read(address)
}

// PrintLoggedReads prints the number of times each address was in
// descending order.
func (m *MMU) PrintLoggedReads() {
	for address, count := range m.loggedReads {
		if count > 1000 {
			fmt.Printf("%04X: %d\n", address, count)
		}
	}
}

func (m *MMU) Write(address uint16, value uint8) {
	switch {
	case address >= 0xC000 && address < 0xD000:
		m.wRAM[address-0xC000] = value
	case address >= 0xD000 && address < 0xE000:
		m.wRAM[m.wRAMOffset+address-0xD000] = value
	case address >= 0xE000 && address < 0xF000:
		m.wRAM[address-0xE000] = value
	case address >= 0xF000 && address < 0xFE00:
		m.wRAM[m.wRAMOffset+address-0xF000] = value
	case address >= 0xFF80 && address < 0xFFFF:
		m.flatMemory[address] = value
	default:
		if m.raw[address] == nil {
			panic(fmt.Sprintf("nil address at %04X", address))
		}
		m.raw[address].Write(address, value)
	}
}

var _ types.Stater = (*MMU)(nil)

func (m *MMU) Load(s *types.State) {
	m.key0 = s.Read8()
	m.key1 = s.Read8()
	m.bootROMDone = s.ReadBool()
	m.Cart.MemoryBankController.Load(s)
}

func (m *MMU) Save(s *types.State) {
	s.Write8(m.key0)
	s.Write8(m.key1)
	s.WriteBool(m.bootROMDone)
	m.Cart.MemoryBankController.Save(s)
}

func (m *MMU) Set(i types.HardwareAddress, v interface{}) {
	register := m.registers[i&0xFF]
	if register == nil {
		panic(fmt.Sprintf("nil address at %04X", i))
	}
	if register.CanSet() {
		register.Set(v)
	} else {
		register.Write(v.(uint8))
	}
}

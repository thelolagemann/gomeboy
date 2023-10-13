package io

import (
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
	"math/rand"
)

type Bus struct {
	data       [0x10000]byte   // 64 KiB memory
	lockedData [0x10000]byte   // 64 KiB memory lock buffer
	wRAM       [8][0x1000]byte // 8 banks of 4 KiB each
	vRAM       [2][0x2000]byte // 2 banks of 8 KiB each

	ppuWrite func(addr uint16, v byte)
	ppuRead  func(addr uint16) byte
	apuRead  func(addr uint16) byte

	writeHandlers [0x100]WriteHandler
	blockWriters  [16]func(uint16, byte)
	setHandlers   [0x100]SetHandler

	bootROMDone bool
	model       types.Model
	isGBC       bool
	s           *scheduler.Scheduler
}

func NewBus(s *scheduler.Scheduler) *Bus {
	b := &Bus{
		s: s,
	}

	return b
}

func (b *Bus) Map(m types.Model, ppuWrite func(uint16, byte), ppuRead, apuRead func(uint16) byte) {
	b.model = m
	b.isGBC = m == types.CGB0 || m == types.CGBABC

	b.ReserveAddress(types.BDIS, func(v byte) byte {
		// any write to BDIS will disable the boot rom
		b.bootROMDone = true
		return 0xFF
	})
	b.ReserveAddress(types.IF, func(v byte) byte {
		return v&0x1F | 0xE0 // upper bits are always 1
	})

	// setup CGB only registers
	if m == types.CGB0 || m == types.CGBABC {
		b.ReserveAddress(types.KEY0, func(v byte) byte {
			// KEY0 is only writable when boot ROM is running TODO verify
			if !b.bootROMDone {
				return v | 0b1111_0010
			}

			return 0xFF
		})
		b.ReserveAddress(types.KEY1, func(v byte) byte {
			// only least significant bit is writable
			return v&0x01 | 0b0111_1110
		})
		b.ReserveAddress(types.VBK, func(v byte) byte {
			// copy currently banked data to VRAM
			copy(b.vRAM[b.data[types.VBK]&0x1][:], b.data[0x8000:0x9FFF])

			// copy VRAM to currently banked data
			copy(b.data[0x8000:0x9FFF], b.vRAM[v&0x1][:])

			return v | 0b1111_1110
		})
		b.ReserveAddress(types.SVBK, func(v byte) byte {
			// copy currently banked data to WRAM
			copy(b.wRAM[b.data[types.SVBK]&0x7][:], b.data[0xD000:0xDFFF])

			// copy WRAM to currently banked data
			copy(b.data[0xD000:0xDFFF], b.wRAM[v&0x7][:])

			return v | 0b1111_1000
		})

		for i := types.FF72; i < types.FF74; i++ {
			b.ReserveAddress(i, func(v byte) byte {
				return v
			})
		}
		b.ReserveAddress(types.FF75, func(v byte) byte {
			// only bits 4-6 are writable
			return v & 0x70
		})
	}

	b.writeHandlers[types.LCDC&0xFF](0x91)
	b.writeHandlers[types.STAT&0xFF](0x87)
	b.writeHandlers[types.BGP&0xFF](0xFC)
	b.data[types.IF] = 0xE1

	b.ppuWrite = ppuWrite
	b.ppuRead = ppuRead
	b.apuRead = apuRead
}

// Boot sets up the bus to the state that it would be
// in after having completed the boot ROM.
func (b *Bus) Boot() {
	// set initial HW
	ioRegs := b.model.IO()
	for i := 0xFF00; i < 0xFF80; i++ {
		// has the model provided a set value?
		if ioRegs[types.HardwareAddress(i)] != nil {
			// is there a set handler registered for this address?
			if handler := b.setHandlers[i&0xFF]; handler != nil {
				handler(ioRegs[types.HardwareAddress(i)])
				fmt.Printf("set handler for address %04X\n", i)
			} else if wHandler := b.writeHandlers[i&0xFF]; wHandler != nil {
				b.data[i] = wHandler(ioRegs[types.HardwareAddress(i)].(byte))
			}
		} else if wHandler := b.writeHandlers[i&0xFF]; wHandler == nil {
			// default to 0xFF if no write handler exists
			b.data[i] = 0xFF
		}
	}

	// set initial tile data
	for i := 0; i < len(initialTileData)*2; i += 2 {
		// every other byte is 0x00, whilst the other byte is set from
		// the initial tile data
		b.data[0x8000+i+16] = initialTileData[i/2]
	}
	// set initial tile map
	for i := 0; i < len(initialTileMap); i++ {
		b.data[0x9904+i] = initialTileMap[i]
	}

	// WRAM is randomised on boot
	for i := 0xC000; i < 0xFDFF; i++ {
		// not accurate to hardware, but random enough to
		// pass most anti emulator checks
		b.data[i] = byte(rand.Intn(256))
	}
}

// WriteHandler is a function that handles writing to a memory address.
// It should return the new value to be written back to the memory address.
type WriteHandler func(byte) byte

// SetHandler is a function that handles setting a value at a memory address.
type SetHandler func(any)

// ReserveAddress reserves a memory address on the bus.
func (b *Bus) ReserveAddress(addr uint16, handler func(byte) byte) {
	// check to make sure address hasn't already been reserved
	if ok := b.writeHandlers[addr&0xFF]; ok != nil {
		// SB can be reserved again for debug purposes
		if addr != types.SB {
			panic(fmt.Sprintf("address %04X has already been reserved", addr))

		}
	}
	b.writeHandlers[addr&0xFF] = handler
}

func (b *Bus) ReserveSetAddress(addr uint16, handler SetHandler) {
	b.setHandlers[addr&0xFF] = handler
}

// Write writes to the specified memory address. This function
// calls the write handler if it exists.
func (b *Bus) Write(addr uint16, value byte) {
	if addr >= 0xFF10 && addr <= 0xFF2F {
		//fmt.Printf("%04x -> %02x\n", addr, value)
	}
	switch {
	case addr <= 0x7FFF || addr >= 0xA000 && addr <= 0xBFFF:
		if w := b.blockWriters[addr/0x1000]; w != nil {
			w(addr, value)
		} else {
			panic("wop")
		}
		return
	// 0x8000-0x9FFF VRAM
	case addr >= 0x8000 && addr <= 0x9FFF:
		b.ppuWrite(addr, value)
	// 0xA000-0xBFFF ERAM (RAM on cartridge)
	// 0xC000-0xFDFF WRAM & mirror
	case addr >= 0xC000 && addr <= 0xFDFF:
		b.data[addr&0xDFFF] = value

		return
	// 0xFE00-0xFE9F OAM
	case addr >= 0xFE00 && addr <= 0xFE9F:
		b.ppuWrite(addr, value)
		return
	// 0xFEA0-0xFEFF Unusable
	case addr >= 0xFEA0 && addr <= 0xFEFF:
		return
	// 0xFF00-0xFFFF IO & HRAM
	case addr >= 0xFF00:
		// is there a write handler for this address?
		if handler := b.writeHandlers[addr&0xFF]; handler != nil {
			value = handler(value)
		} else if addr <= 0xff7f {
			return
		}
	}
	b.data[addr] = value
}

// Read reads from the specified memory address. Some addresses
// need special handling.
func (b *Bus) Read(addr uint16) byte {
	if addr >= 0xFF10 && addr <= 0xFF2F {
		//fmt.Printf("%04x -> %02x\n", addr, b.data[addr])
		//return b.apuRead(addr)
	}
	switch {
	case addr >= 0xFE00 && addr <= 0xFE9F:
		return b.ppuRead(addr)
	case addr >= 0xE000 && addr <= 0xFDFF:
		return b.data[addr&0xDDFF]
	case addr >= 0xFF30 && addr <= 0xFF3F:
		return b.apuRead(addr)
	case addr >= 0xFF00 && addr <= 0xFF7F:
		if addr == types.DIV {
			return uint8(b.s.SysClock() >> 8)
		}
		if addr != 0xFF44 && addr != 0xFF00 {
			//fmt.Printf("special read handle %04x %02x\n", addr, b.data[addr])
		}
	}

	return b.data[addr]
}

// Get gets the value at the specified memory address.
func (b *Bus) Get(addr uint16) byte {
	switch addr {
	case types.VBK:
		if !b.isGBC {
			return 0xFE // return VRAM bank 0 always
		}
	}
	return b.data[addr]
}

// Set sets the value at the specified memory address. This function
// ignores the write handler and just sets the value.
func (b *Bus) Set(addr uint16, value byte) {
	b.data[addr] = value
}

// SetBit sets the bit at the specified memory address.
func (b *Bus) SetBit(addr uint16, bit byte) {
	b.data[addr] |= bit
}

// ClearBit clears the bit at the specified memory address.
func (b *Bus) ClearBit(addr uint16, bit byte) {
	b.data[addr] &= ^bit
}

// TestBit tests the bit at the specified memory address.
func (b *Bus) TestBit(addr uint16, bit byte) bool {
	return b.data[addr]&bit != 0
}

// RaiseInterrupt raises the specified interrupt.
func (b *Bus) RaiseInterrupt(interrupt byte) {
	b.data[types.IF] |= interrupt
}

// HasInterrupts returns true if there are pending interrupts.
func (b *Bus) HasInterrupts() bool {
	return b.data[types.IE]&b.data[types.IF] != 0
}

// IRQVector returns the currently serviced interrupt vector,
// or 0 if no interrupt is being serviced. This function
// will also clear the corresponding bit in the types.IF
// register.
func (b *Bus) IRQVector(irq byte) uint16 {
	for i := uint8(0); i < 5; i++ {
		// get the flag for the current interrupt
		flag := uint8(1 << i)

		// check if the interrupt is requested and enabled
		if irq&b.data[types.IF] == flag {
			// clear the flag
			b.data[types.IF] &= ^flag

			// return vector
			return uint16(0x0040 + i*8)
		}
	}

	return 0
}

// Model returns the current model.
func (b *Bus) Model() types.Model {
	return b.model
}

// IsGBC returns true if the current model is a
// GBC model.
func (b *Bus) IsGBC() bool {
	return b.isGBC
}

// LockRange locks the specified memory range so that
// any reads to it will return 0xFF. It does this by
// first copying the existing data to a buffer and
// then overwriting the data with 0xFF. When UnlockRange
// is called, the data is copied back to the original location.
func (b *Bus) LockRange(start, end uint16) {
	// copy existing data to lock buffer
	copy(b.lockedData[start:end], b.data[start:end])

	switch end - start {
	// 8KiB lock
	case 0x2000:
		copy(b.data[start:end], lock8KiB)
	}

}

// UnlockRange unlocks the specified memory range.
func (b *Bus) UnlockRange(start, end uint16) {
	return
	// copy lock buffer back to original location
	copy(b.data[start:end], b.lockedData[start:end])
}

// CopyFrom copies the specified memory range to the specified
// destination.
func (b *Bus) CopyFrom(start, end uint16, dest []byte) {
	copy(dest, b.data[start:end])
}

// CopyTo copies the specified memory range from the specified
// source.
func (b *Bus) CopyTo(start, end uint16, src []byte) {
	copy(b.data[start:end], src)
}

func (b *Bus) ReserveBlockWriter(start uint16, h func(uint16, byte)) {
	b.blockWriters[start/0x1000] = h
}

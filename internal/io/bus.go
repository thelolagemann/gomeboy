package io

import (
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
	"math/rand"
)

// Bus is the main component responsible for handling IO
// operations on the Game Boy. The Game Boy has a 16-bit
// address bus, allowing for a 64KiB memory space.
//
// The memory space is mapped as so:
//
//		Start  | End	| Name
//	 ----------------------------
//		0x0000 | 0x7FFF | ROM
//		0x8000 | 0x9FFF | VRAM
//		0xA000 | 0xBFFF | External RAM
//		0xC000 | 0xDFFF | Work RAM
//		0xE000 | 0xFDFF | Work RAM Mirror
//		0xFE00 | 0xFE9F | OAM
//		0xFEA0 | 0xFEFF | Not used
//		0xFF00 | 0xFF7F | IO Registers
//		0xFF80 | 0xFFFE | High RAM
//		0xFFFF | 0xFFFF | Interrupt Enable Register
type Bus struct {
	data [0x10000]byte   // 64 KiB memory
	wRAM [8][0x1000]byte // 8 banks of 4 KiB each
	vRAM [2][0x2000]byte // 2 banks of 8 KiB each

	writeHandlers [0x100]WriteHandler
	setHandlers   [0x100]SetHandler
	lazyReaders   [0x100]LazyReader
	blockWriters  [16]func(uint16, byte)

	model     types.Model
	isGBC     bool
	isGBCCart bool
	s         *scheduler.Scheduler

	gbcHandlers []func()

	// DMA related stuff
	dmaSource, dmaDestination uint16
	dmaActive, dmaRestarting  bool
	dmaConflict               uint8
	dmaEnabled                bool
	oamChanged                bool
	dmaConflicts              [16]bool
	rLocks                    [16]bool
	wLocks                    [16]bool

	// HDMA/GDMA related stuff
	hdmaSource, hdmaDestination uint16
	dmaLength                   uint8
	dmaRemaining                uint8
	dmaComplete, dmaPaused      bool

	// various IO
	buttonState uint8
	ime         bool
	bootROMDone bool
	wRAMBank    uint8 // 1 - 7 in CGB mode
}

// NewBus creates a new Bus instance.
func NewBus(s *scheduler.Scheduler) *Bus {
	b := &Bus{
		s:           s,
		gbcHandlers: make([]func(), 0),
		dmaConflict: 0xff,
	}

	// setup DMA events
	s.RegisterEvent(scheduler.DMATransfer, b.doDMATransfer)
	s.RegisterEvent(scheduler.DMAStartTransfer, b.startDMATransfer)
	s.RegisterEvent(scheduler.DMAEndTransfer, b.endDMATransfer)

	return b
}

func (b *Bus) Map(m types.Model, cartCGB bool) {
	b.model = m
	b.isGBC = m == types.CGBABC || m == types.CGB0
	b.isGBCCart = cartCGB

	b.ReserveLazyReader(types.DIV, func() byte {
		return byte(b.s.SysClock() >> 8)
	})
	b.ReserveAddress(types.DMA, func(v byte) byte {
		// source address is v << 8
		b.dmaSource = uint16(v) << 8

		// set conflicting bus
		if b.dmaSource >= 0x8000 && b.dmaSource < 0xA000 {
			b.dmaConflicts[8] = true
			b.dmaConflicts[9] = true
		} else if b.dmaSource < 0x8000 || b.dmaSource >= 0xA000 && b.dmaSource <= 0xFEFF {
			for i := 0; i < 16; i++ {
				if i == 8 || i == 9 {
					b.dmaConflicts[i] = false
					continue
				}
				b.dmaConflicts[i] = true
			}
		}

		if b.dmaSource >= 0xE000 && b.dmaSource < 0xFE00 {
			b.dmaSource &= 0xDDFF // account for mirroring
		} else if b.dmaSource >= 0xFE00 {
			b.dmaSource -= 0x2000 // why
		}

		// mark DMA as being inactive
		b.dmaActive = false

		// handle DMA restarts
		b.dmaRestarting = b.dmaEnabled

		// reset destination
		b.dmaDestination = 0xFE00

		// deschedule any existing DMA transfers
		if b.dmaRestarting {
			b.s.DescheduleEvent(scheduler.DMATransfer)
			b.s.DescheduleEvent(scheduler.DMAStartTransfer)
			b.s.DescheduleEvent(scheduler.DMAEndTransfer)
		}

		// mark DMA as being enabled (not the same as active)
		b.dmaEnabled = true

		// schedule DMA transfer
		b.s.ScheduleEvent(scheduler.DMAStartTransfer, 8) // TODO find out why 8 instead of 4?

		// DMA always returns the last value written
		// https://github.com/Gekkio/mooneye-test-suite/blob/main/acceptance/oam_dma/reg_read.s
		return v
	})

	// setup CGB only registers
	if b.isGBC && b.isGBCCart {
		b.ReserveAddress(types.KEY0, func(v byte) byte {
			// KEY0 is only writable when boot ROM is running TODO verify
			if !b.bootROMDone {
				return v | 0b1111_0010
			}

			return 0xFF
		})
		b.ReserveAddress(types.KEY1, func(v byte) byte {
			return b.Get(types.KEY1)&types.Bit7 | v&0x1 | 0x7e
		})
		b.Set(types.KEY1, 0x7e)

		// setup hdma registers
		b.ReserveAddress(types.HDMA1, func(v byte) byte {
			b.hdmaSource &= 0xF0
			b.hdmaSource |= uint16(v) << 8
			if b.hdmaSource >= 0xE000 {
				b.hdmaSource |= 0xF000
			}
			return 0xff
		})
		b.ReserveAddress(types.HDMA2, func(v byte) byte {
			b.hdmaSource &= 0xFF00
			b.hdmaSource |= uint16(v & 0xF0)

			return 0xff
		})
		b.ReserveAddress(types.HDMA3, func(v byte) byte {
			b.hdmaDestination &= 0x00F0
			b.hdmaDestination |= uint16(v) << 8

			return 0xff
		})
		b.ReserveAddress(types.HDMA4, func(v byte) byte {
			b.hdmaDestination &= 0x1F00
			b.hdmaDestination |= uint16(v & 0xF0)

			return 0xff
		})
		b.ReserveAddress(types.HDMA5, func(v byte) byte {
			// update the length
			b.dmaLength = (v & 0x7F) + 1

			// if bit 7 is set, we are starting a new HDMA transfer
			if v&types.Bit7 != 0 {
				b.dmaRemaining = b.dmaLength // set the remaining length

				// reset the DMA flags
				b.dmaComplete = false
				b.dmaPaused = false

				// if the LCD is disabled, one HDMA transfer is performed immediately
				// and the rest are performed during the next HBlank period
				if b.Get(types.LCDC)&types.Bit7 != types.Bit7 && b.dmaRemaining > 0 {
					b.newDMA(1)
					b.dmaRemaining--
				}

				// if the PPU is already in the HBlank period, then the HDMA would not be
				// performed by the scheduler until the next HBlank period, so we perform
				// the transfer immediately here and decrement the remaining length
				if b.Get(types.LCDC)&types.Bit7 == types.Bit7 && b.LazyRead(types.STAT)&0b11 == 0 && b.dmaRemaining > 0 {
					b.newDMA(1)
					b.dmaRemaining--
				}
			} else {
				// if bit 7 is not set, we are starting a new GDMA transfer
				if b.dmaRemaining > 0 {
					// if we're in the middle of a HDMA transfer, pause it
					b.dmaPaused = true

					b.dmaRemaining = b.dmaLength
				} else {
					// if we're not in the middle of a HDMA transfer, perform a GDMA transfer
					b.newDMA(b.dmaLength)
				}
			}

			if b.dmaComplete {
				return 0xFF
			} else {
				v := uint8(0)
				if b.dmaPaused {
					v |= types.Bit7
				}
				return v | (b.dmaRemaining-1)&0x7F
			}
		})
		b.ReserveLazyReader(types.HDMA5, func() byte {
			if b.dmaComplete || b.dmaRemaining == 0 {
				return 0xFF
			} else {
				v := uint8(0)
				if b.dmaPaused {
					v |= types.Bit7
				}
				return v | (b.dmaRemaining-1)&0x7F
			}
		})

		b.ReserveAddress(types.SVBK, func(v byte) byte {
			// copy currently banked data to WRAM
			copy(b.wRAM[b.data[types.SVBK]&0x7][:], b.data[0xD000:0xE000])
			if v == 0 {
				v = 1
			}
			// copy WRAM to currently banked data
			copy(b.data[0xD000:0xE000], b.wRAM[v&0x7][:])

			return v | 0b1111_1000
		})
		b.Set(types.SVBK, 0xF9)

	}

	// setup cgb model registers
	if b.model == types.CGBABC || b.model == types.CGB0 {
		b.ReserveAddress(types.VBK, func(v byte) byte {
			// copy currently banked data to VRAM
			copy(b.vRAM[b.data[types.VBK]&0x1][:], b.data[0x8000:0x9FFF])

			// copy VRAM to currently banked data
			copy(b.data[0x8000:0x9FFF], b.vRAM[v&0x1][:])

			return v | 0b1111_1110
		})
		b.Set(types.VBK, 0xFE)
		for i := types.FF72; i < types.FF74; i++ {
			b.ReserveAddress(i, func(v byte) byte {
				return v
			})
		}
		b.ReserveAddress(types.FF74, func(b byte) byte {
			return 0xFF
		})
		b.ReserveAddress(types.FF75, func(v byte) byte {
			// only bits 4-6 are writable
			return v&0x70 | 0x8F
		})
		b.Set(types.FF75, 0x8F)

		for _, f := range b.gbcHandlers {
			f()
		}
	}

	b.data[0xff7f] = 0xff
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
			} else if wHandler := b.writeHandlers[i&0xFF]; wHandler != nil {
				b.data[i] = wHandler(ioRegs[types.HardwareAddress(i)].(byte))
			} else if _, ok := ioRegs[types.HardwareAddress(i)].(byte); ok {
				// set data as is
				b.data[i] = ioRegs[types.HardwareAddress(i)].(byte)
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
		b.blockWriters[8](uint16(0x8000+i+16), initialTileData[i/2])
	}
	// set initial tile map
	for i := 0; i < len(initialTileMap); i++ {
		b.data[0x9904+i] = initialTileMap[i]
		b.blockWriters[8](uint16(0x9904+i), initialTileMap[i])
	}

	// WRAM is randomised on boot
	for i := 0xC000; i < 0xFDFF; i++ {
		// not accurate to hardware, but random enough to
		// pass most anti emulator checks
		b.data[i] = byte(rand.Intn(256))
	}

	// setup starting events for scheduler
	events := b.model.Events()
	if len(events) > 0 {
		for i := scheduler.APUChannel1; i <= scheduler.JoypadDownRelease; i++ {
			b.s.DescheduleEvent(i)
		}
		// set starting event for scheduler
		for _, e := range events {
			b.s.ScheduleEvent(e.Type, e.Cycle)
		}
	}

	// handle special case registers
	b.data[types.BDIS] = 0xFF
	b.bootROMDone = true
	b.writeHandlers[types.LCDC&0xFF](0x91)
	b.writeHandlers[types.STAT&0xFF](0x87)
	b.writeHandlers[types.BGP&0xFF](0xFC)
	b.data[types.IF] = 0xE1
}

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

func (b *Bus) ReserveLazyReader(addr uint16, handler LazyReader) {
	b.lazyReaders[addr&0xFF] = handler
}

func (b *Bus) ReserveSetAddress(addr uint16, handler SetHandler) {
	b.setHandlers[addr&0xFF] = handler
}

// Write writes to the specified memory address. This function
// calls the write handler if it exists.
func (b *Bus) Write(addr uint16, value byte) {
	switch {
	// IO & HRAM can't be locked or conflicted
	case addr >= 0xFF00:
		switch addr {
		// handle addresses on the bus
		case types.P1:
			d := uint8(0xC0)
			if value&types.Bit4 == 0 {
				d |= b.buttonState >> 4 & 0xf
				d |= types.Bit4
			}
			if value&types.Bit5 == 0 {
				d |= b.buttonState & 0xf
				d |= types.Bit5
			}

			d ^= 0xf

			value = d
		case types.BDIS:
			if b.bootROMDone {
				return
			}
			// any write to BDIS will unmap the boot ROM,
			// and it should always read 0xFF
			b.bootROMDone = true
			value = 0xFF

			// copy rom contents back
			b.CopyTo(0, 0x900, b.data[0xE000:0xE900])

			// load colourisation palette into PPU (if in CGB mode with a DMG cart)
			if b.isGBC && !b.isGBCCart {
				b.Write(0xFF7F, 0xFF)
			}
		case types.IF:
			value = value | 0xE0 // upper bits are always 1
			if b.ime && b.data[types.IE]&value&0x1f != 0 {
				b.Write(0xFF7e, 0)
			}
		case types.IE:
			if b.ime && b.data[types.IF]&value != 0 {
				b.Write(0xFF7e, 0)
			}
		default:
			// check to see if a component has reserved this address
			if handler := b.writeHandlers[addr&0xFF]; handler != nil {
				value = handler(value)
			} else if addr <= 0xff7f {
				// otherwise do nothing
				return
			}
		}
	default:
		// address <= 0xFDFF can be locked or conflicted
		if b.isDMATransferring() && b.dmaConflicts[addr>>12] {
			return
		}
		switch {
		// 0x0000 - 0x7FFF ROM
		// 0xA000 - 0xBFFF ERAM (RAM on cartridge)
		case addr <= 0x7FFF || addr >= 0xA000 && addr <= 0xBFFF:
			b.blockWriters[addr/0x1000](addr, value)
			return
		// 0x8000 - 0x9FFF VRAM
		case addr >= 0x8000 && addr <= 0x9FFF:
			// if locked, return
			if b.wLocks[addr>>12] {
				return
			}
			b.blockWriters[addr/0x1000](addr, value)
		// 0xC000-0xFDFF WRAM & mirror
		case addr >= 0xC000 && addr <= 0xFDFF:
			b.data[addr&0xDFFF] = value
			b.data[addr&0xDDFF|0xE000] = value

			return
		// 0xFE00-0xFE9F OAM
		case addr >= 0xFE00 && addr <= 0xFE9F:
			if b.wLocks[0xF] || b.isDMATransferring() {
				return
			}
			b.oamChanged = true
		}
	}

	b.data[addr] = value
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

func (b *Bus) LazyRead(addr uint16) byte {
	if handler := b.lazyReaders[addr&0xFF]; handler != nil {
		return handler()
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
	b.data[addr] &^= bit
}

// RLock locks the specified bus from being read.
func (b *Bus) RLock(bus MemoryRegion) {
	for _, lock := range bus.BusLocks() {
		b.rLocks[lock] = true
	}
}

// RUnlock unlocks the specified bus from being read.
func (b *Bus) RUnlock(bus MemoryRegion) {
	for _, lock := range bus.BusLocks() {
		b.rLocks[lock] = false
	}
}

// WLock locks the specified bus from being written.
func (b *Bus) WLock(bus MemoryRegion) {
	for _, lock := range bus.BusLocks() {
		b.wLocks[lock] = true
	}
}

// WUnlock unlocks the specified bus from being written.
func (b *Bus) WUnlock(bus MemoryRegion) {
	for _, lock := range bus.BusLocks() {
		b.wLocks[lock] = false
	}
}

// Lock locks the specified bus from being read and written.
func (b *Bus) Lock(bus MemoryRegion) {
	b.RLock(bus)
	b.WLock(bus)
}

// Unlock unlocks the specified bus from being read and written.
func (b *Bus) Unlock(bus MemoryRegion) {
	b.RUnlock(bus)
	b.WUnlock(bus)
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

func (b *Bus) WhenGBC(f func()) {
	b.gbcHandlers = append(b.gbcHandlers, f)
}

func (b *Bus) IsGBCCart() bool {
	return b.isGBCCart
}

// ClockedRead clocks the Game Boy and reads a byte from the
// bus.
func (b *Bus) ClockedRead(addr uint16) byte {
	b.s.Tick(4)

	value := b.data[addr]

	switch {
	case addr <= 0x9FFF || addr >= 0xC000 && addr <= 0xFDFF:
		if b.rLocks[(addr>>12)&0xe] || (b.isDMATransferring() && b.dmaConflicts[addr>>12]) {
			value = b.dmaConflict
		}
	case addr <= 0xBFFF:
		// ERAM can't be conflicted so additional check for rLock here
		if b.rLocks[addr>>12] {
			value = 0xff
		}
	// HRAM/IO can't be locked or conflicted
	case addr >= 0xFF00:
		// does this register need evaluating?
		if f := b.lazyReaders[addr&0xff]; f != nil {
			value = f()
		}

	// OAM can be read locked by the PPU and a DMA transfer
	case addr >= 0xFE00 && addr <= 0xFE9F:
		if b.rLocks[0xF] || b.isDMATransferring() {
			value = 0xff
		}
	}

	// if we've managed to fall through to here, we should be
	// able to read the data as it is on the bus
	return value
}

// ClockedWrite clocks the Game Boy and writes a byte to the
// bus.
func (b *Bus) ClockedWrite(address uint16, value byte) {
	b.s.Tick(4)

	if address >= types.NR10 && address <= types.NR52 {
		//fmt.Printf("%04x %02x %16b\n", address, value, b.s.SysClock())
	}
	b.Write(address, value)
}

func (b *Bus) IsBooting() bool {
	return !b.bootROMDone
}

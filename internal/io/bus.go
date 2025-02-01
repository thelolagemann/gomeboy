package io

import (
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/utils"
	"math/rand"
)

const (
	VRAM uint16 = 0b0000_0011_0000_0000
	OAM         = 0b1000_0000_0000_0000
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
	InterruptCallback func(v uint8) // used to notify CPU of interrupts

	data [0x10000]byte   // 64 KiB memory
	wRAM [7][0x1000]byte // 7 banks of 4 KiB each (bank 0 is fixed)
	vRAM [2][0x2000]byte // 2 banks of 8 KiB each

	writeHandlers [0x100]func(byte) byte
	lazyReaders   [0x100]func() byte

	bootHandlers []func()

	c *Cartridge

	model types.Model
	isGBC bool
	s     *scheduler.Scheduler

	gbcHandlers []func()

	// DMA related stuff
	dmaSource, dmaDestination  uint16
	dmaActive, dmaRestarting   bool
	dmaConflict                uint8
	dmaEnabled                 bool
	regionLocks, dmaConflicted uint16

	// HDMA/GDMA related stuff
	hdmaSource, hdmaDestination uint16
	dmaLength                   uint8
	dmaRemaining                uint8
	dmaComplete, dmaPaused      bool

	// various IO
	buttonState  uint8
	ime          bool
	bootROMDone  bool
	vRAMBankMask uint8
	debug        bool
	Debugging    bool

	// cheats
	LoadedCheats   []Cheat
	GameGenieCodes []GameGenieCode
	GameSharkCodes []GameSharkCode
}

func (b *Bus) Debugf(f string, a ...interface{}) {
	if b.Debugging && b.s.Cycle() < 101938 {
		//fmt.Printf(fmt.Sprintf("%d:", b.s.Cycle())+f, a...)
	}
}

// NewBus creates a new Bus instance.
func NewBus(s *scheduler.Scheduler, rom []byte) *Bus {
	b := &Bus{
		s:           s,
		dmaConflict: 0xff,
		Debugging:   true,
	}
	b.c = NewCartridge(rom, b)
	b.ReserveLazyReader(types.DIV, func() byte { return byte(b.s.SysClock() >> 8) })

	s.RegisterEvent(scheduler.DMATransfer, b.doDMATransfer)
	s.RegisterEvent(scheduler.DMAStartTransfer, b.startDMATransfer)
	s.RegisterEvent(scheduler.DMAEndTransfer, b.endDMATransfer)
	s.RegisterEvent(scheduler.CameraShoot, func() { b.c.Camera.Registers[CameraShoot] &^= 1 })

	return b
}

func (b *Bus) Map(m types.Model) {
	b.model = m
	b.isGBC = m == types.CGBABC || m == types.CGB0

	// setup CGB only registers
	if b.isGBC && b.c.IsCGBCartridge() {
		b.ReserveAddress(types.KEY0, func(v byte) byte {
			// KEY0 is only writable when boot ROM is running TODO verify
			if !b.bootROMDone {
				return v | 0b1111_0010
			}

			return 0xFF
		})
		b.ReserveAddress(types.KEY1, func(v byte) byte { return b.data[types.KEY1]&types.Bit7 | v&0x1 | 0x7e })

		// setup hdma registers
		b.ReserveAddress(types.HDMA1, func(v byte) byte {
			b.hdmaSource = b.hdmaSource&0x00F0 | uint16(v)<<8
			if b.hdmaSource >= 0xE000 {
				b.hdmaSource |= 0xF000
			}
			return 0xff
		})
		b.ReserveAddress(types.HDMA2, func(v byte) byte {
			b.hdmaSource = b.hdmaSource&0xFF00 | uint16(v&0xF0)
			return 0xff
		})
		b.ReserveAddress(types.HDMA3, func(v byte) byte {
			b.hdmaDestination = b.hdmaDestination&0x00F0 | uint16(v)<<8
			return 0xff
		})
		b.ReserveAddress(types.HDMA4, func(v byte) byte {
			b.hdmaDestination = b.hdmaDestination&0xFF00 | uint16(v&0xF0)
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
					b.colorDMA(1)
					b.dmaRemaining--
				}

				// if the PPU is already in the HBlank period, then the HDMA would not be
				// performed by the scheduler until the next HBlank period, so we perform
				// the transfer immediately here and decrement the remaining length
				if b.Get(types.LCDC)&types.Bit7 == types.Bit7 && b.LazyRead(types.STAT)&0b11 == 0 && b.dmaRemaining > 0 {
					b.colorDMA(1)
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
					b.colorDMA(b.dmaLength)
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
			copy(b.wRAM[utils.ZeroAdjust(b.data[types.SVBK]&7)-1][:], b.data[0xD000:0xE000]) // bus -> wRAM
			copy(b.data[0xD000:0xE000], b.wRAM[utils.ZeroAdjust(v&7)-1][:])                  // wRAM -> bus
			return v | 0xF8
		})
		b.Set(types.SVBK, 0xF8)
	}

	// setup cgb model registers
	if b.model == types.CGBABC || b.model == types.CGB0 {
		b.vRAMBankMask = 1
		b.ReserveAddress(types.VBK, func(v byte) byte {
			if b.IsGBCCart() || b.IsBooting() { // CGB boot ROM makes use of both banks
				copy(b.vRAM[b.data[types.VBK]&0x1][:], b.data[0x8000:0xA000])
				copy(b.data[0x8000:0xA000], b.vRAM[v&0x1][:])
				return v | 0xfe
			}
			return 0xff
		})
		b.ReserveAddress(types.FF72, func(v byte) byte { return v })
		b.ReserveAddress(types.FF73, func(v byte) byte { return v })
		b.ReserveAddress(types.FF75, func(v byte) byte { return v&0x70 | 0x8F })
		b.Set(types.FF75, 0x8F)

		for _, f := range b.gbcHandlers {
			f()
		}
	}

	for i := 0xFF00; i < 0xFF80; i++ {
		if wHandler := b.writeHandlers[i&0xFF]; wHandler == nil {
			b.data[i] = 0xFF // default to 0xff if no write handler exists
		}
	}
}

// Boot the Bus to the state it would be in after execution of the boot ROM.
func (b *Bus) Boot() {
	ioRegs := make(map[types.HardwareAddress]interface{})
	for k, v := range types.CommonIO {
		ioRegs[k] = v
	}
	for k, v := range types.ModelIO[b.model] {
		ioRegs[k] = v
	}
	for i := types.HardwareAddress(0xFF00); i < 0xFF80; i++ {
		// has the model provided a value?
		if ioRegs[i] != nil {
			if i == types.DIV { // special case for DIV
				b.s.OverrideDiv(ioRegs[i].(uint16))
			} else if wHandler := b.writeHandlers[i&0xFF]; wHandler != nil {
				b.data[i] = wHandler(ioRegs[i].(byte))
			} else if _, ok := ioRegs[i].(byte); ok {
				b.data[i] = ioRegs[i].(byte)
			}
		}
	}

	// unpack logo data
	logoData := b.data[0x0104:0x0134]
	var unpackedLogoData []byte
	for i := 0; i < len(logoData); i++ {
		var currentData [8]uint8 // every other byte is 0
		for bit := uint8(0); bit < 8; bit++ {
			n := logoData[i] >> bit & 1
			currentData[0] |= n<<(2*(bit-4)) | n<<(2*(bit-4)+1)
			currentData[4] |= n<<(2*bit) | n<<(2*bit+1)
		}
		currentData[2], currentData[6] = currentData[0], currentData[4] // double bytes
		unpackedLogoData = append(unpackedLogoData, currentData[:]...)
	}
	copy(b.data[0x8010:], append(unpackedLogoData, 0x3C, 0, 0x42, 0, 0xB9, 0, 0xA5, 0, 0xB9, 0, 0xA5, 0, 0x42, 0, 0x3C))
	for i := uint8(0); i < 12; i++ {
		b.data[0x9904+uint16(i)] = i + 1
		b.data[0x9924+uint16(i)] = i + 13
	}
	b.data[0x9910] = 0x19

	// wRAM is randomized on boot (not accurate to hardware, but random enough to pass most anti-emu checks)
	for i := 0; i < 0x2000; i++ {
		v := byte(rand.Intn(256))
		b.data[0xC000+i] = v
		if i <= 0x1dff {
			b.data[0xE000+i] = v
		}
	}

	if b.model == types.CGBABC || b.model == types.CGB0 {
		b.Set(types.VBK, 0xFE)

		if !b.IsGBCCart() {
			b.vRAMBankMask = 0
		}
	}

	b.bootROMDone = true
	b.data[types.IF] = 0xE1
}

func (b *Bus) ReserveAddress(addr uint16, f func(byte) byte) { b.writeHandlers[addr&0xff] = f }             // reserve IO address
func (b *Bus) ReserveLazyReader(addr uint16, f func() byte)  { b.lazyReaders[addr&0xff] = f }               // reserve IO lazy reader
func (b *Bus) RegisterBootHandler(f func())                  { b.bootHandlers = append(b.bootHandlers, f) } // called after boot ROM
func (b *Bus) RegisterGBCHandler(f func())                   { b.gbcHandlers = append(b.gbcHandlers, f) }   // called when model is CGB

// Write writes to the specified memory address. This function
// calls the write handler if it exists.
func (b *Bus) Write(addr uint16, value byte) {
	switch {
	// IO & HRAM can't be locked or conflicted
	case addr >= 0xFF00:
		switch addr {
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
			b.bootROMDone = true
			value = 0xff
			if b.isGBC && !b.IsGBCCart() {
				b.vRAMBankMask = 0
			}

			for _, f := range b.bootHandlers {
				f()
			}
		case types.IF:
			value = value | 0xE0 // upper bits are always 1
			if b.ime && b.data[types.IE]&value&0x1f != 0 {
				b.InterruptCallback(0)
			}
		case types.DMA:
			b.dmaSource = uint16(value) << 8

			if !b.isGBC {
				// set conflicting bus
				if b.dmaSource >= 0x8000 && b.dmaSource < 0xA000 {
					b.dmaConflicted = VRAM
				} else if b.dmaSource < 0x8000 || b.dmaSource >= 0xA000 && b.dmaSource <= 0xFEFF {
					b.dmaConflicted = ^VRAM
				}
			}

			if b.dmaSource >= 0xE000 && b.dmaSource < 0xFE00 {
				b.dmaSource &= 0xDDFF // account for mirroring
			} else if b.dmaSource >= 0xFE00 {
				b.dmaSource -= 0x2000 // why
			}

			b.dmaActive = false
			b.dmaRestarting = b.dmaEnabled
			b.dmaDestination = 0xFE00

			// de-schedule any existing DMA transfers
			if b.dmaRestarting {
				b.s.DescheduleEvent(scheduler.DMATransfer)
				b.s.DescheduleEvent(scheduler.DMAStartTransfer)
				b.s.DescheduleEvent(scheduler.DMAEndTransfer)
			}

			b.dmaEnabled = true
			b.s.ScheduleEvent(scheduler.DMAStartTransfer, 8) // TODO find out why 8 instead of 4?
		case types.IE:
			if b.ime && b.data[types.IF]&value != 0 {
				b.InterruptCallback(0)
			}
		default:
			if handler := b.writeHandlers[addr&0xFF]; handler != nil {
				// check to see if a component has reserved this address
				value = handler(value)
			} else if addr <= 0xff7f {
				return
			}
		}
	default:
		// address <= 0xFDFF can be locked or conflicted
		if b.isDMATransferring() && b.dmaConflicted&(1<<(addr>>12)) > 0 {
			return
		}
		switch {
		// 0x0000 - 0x7FFF ROM
		// 0xA000 - 0xBFFF ERAM (RAM on cartridge)
		case addr <= 0x7FFF || addr >= 0xA000 && addr <= 0xBFFF:
			b.c.Write(addr, value)
			return
		// 0x8000 - 0x9FFF VRAM
		case addr <= 0x9FFF:
			if (b.regionLocks<<8)&(1<<(addr>>12)) > 0 {
				return
			}
			b.vRAM[b.data[types.VBK]&b.vRAMBankMask][addr&0x1fff] = value
		// 0xC000-0xFDFF WRAM & mirror
		case addr <= 0xFDFF:
			b.data[addr&0xDFFF] = value
			b.data[addr&0xDDFF|0xE000] = value

			return
		// 0xFE00-0xFE9F OAM
		case addr <= 0xFE9F:
			if (b.regionLocks<<8)&OAM > 0 || b.isDMATransferring() {
				return
			}
		}
	}

	b.data[addr] = value
}

func (b *Bus) GetVRAM(address uint16, bank uint8) uint8 {
	return b.vRAM[bank&b.vRAMBankMask][address]
}

func (b *Bus) LazyRead(addr uint16) byte {
	if handler := b.lazyReaders[addr&0xFF]; handler != nil {
		return handler()
	}

	return b.data[addr]
}

func (b *Bus) ClearBit(addr uint16, bit byte) { b.data[addr] &^= bit } // clear bit at address
func (b *Bus) Get(addr uint16) byte           { return b.data[addr] }  // get value at address
func (b *Bus) Set(addr uint16, value byte)    { b.data[addr] = value } // set value at address
func (b *Bus) SetBit(addr uint16, bit byte)   { b.data[addr] |= bit }  // set bit at address

func (b *Bus) Block(region uint16, block bool) {
	if block {
		b.Lock(region)
	} else {
		b.Unlock(region)
	}
}
func (b *Bus) RBlock(region uint16, block bool) {
	if block {
		b.RLock(region)
	} else {
		b.RUnlock(region)
	}
}
func (b *Bus) WBlock(region uint16, block bool) {
	if block {
		b.WLock(region)
	} else {
		b.WUnlock(region)
	}
}

func (b *Bus) RLock(region uint16)   { b.regionLocks |= region }              // locks reading from region
func (b *Bus) RUnlock(region uint16) { b.regionLocks &^= region }             // unlocks reading from region
func (b *Bus) WLock(region uint16)   { b.regionLocks |= region >> 8 }         // locks writing to region
func (b *Bus) WUnlock(region uint16) { b.regionLocks &^= region >> 8 }        // unlocks writing to region
func (b *Bus) Lock(region uint16)    { b.regionLocks |= region | region>>8 }  // lock read/writing from region
func (b *Bus) Unlock(region uint16)  { b.regionLocks &^= region | region>>8 } // unlock read/writing from region

func (b *Bus) CopyFrom(start, end uint16, dest []byte) { copy(dest, b.data[start:end]) } // copy from bus -> dest
func (b *Bus) CopyTo(start, end uint16, src []byte) {
	copy(b.data[start:end], src)

	// check to see if any game genie ROM patches should be applied to the src
	if len(b.GameGenieCodes) > 0 {
		for _, c := range b.GameGenieCodes {
			if c.Address >= start && c.Address <= end {
				if b.data[c.Address] == c.OldData {
					b.data[c.Address] = c.NewData
				}
			}
		}
	}
}

func (b *Bus) Read(addr uint16) byte {
	switch {
	case addr <= 0x9FFF || addr >= 0xC000 && addr <= 0xFDFF:
		addrBitmask := uint16(1 << (addr >> 12))
		if b.regionLocks&0xff00&(addrBitmask&0x7fff) > 0 || (b.isDMATransferring() && b.dmaConflicted&addrBitmask > 0) {
			return b.dmaConflict
		}
	case addr <= 0xBFFF:
		switch b.c.CartridgeType {
		case MBC3TIMERBATT, MBC3TIMERRAMBATT:
			if b.c.rtc.enabled && b.c.rtc.register != 0 {
				return b.c.RAM[b.c.RAMSize+int(b.c.rtc.register-3)]
			}
		case MBC7:
			return b.c.readMBC7RAM(addr)
		case POCKETCAMERA:
			return b.c.readCameraRAM(addr)
		case HUDSONHUC1:
			if b.c.huc1.irMode {
				return 0xc0 // no light
			} else {
				return b.data[addr]
			}
		default:
			if !b.c.ramEnabled {
				return 0xff
			}
		}
	// HRAM/IO can't be locked or conflicted
	case addr >= 0xFF00:
		// does this register need evaluating?
		if f := b.lazyReaders[addr&0xff]; f != nil {
			return f()
		}
	// OAM can be read locked by the PPU and a DMA transfer
	case addr <= 0xFE9F:
		if b.regionLocks&OAM > 0 || b.isDMATransferring() {
			return 0xff
		}
	}

	// if we've managed to fall through to here, we should be
	// able to read the data as it is on the bus
	return b.data[addr]
}

// ClockedRead clocks the Game Boy and reads a byte from the
// bus.
func (b *Bus) ClockedRead(addr uint16) byte {
	b.s.Tick(4)
	return b.Read(addr)
}

// ClockedWrite clocks the Game Boy and writes a byte to the
// bus.
func (b *Bus) ClockedWrite(address uint16, value byte) {
	b.s.Tick(4)

	b.Write(address, value)
}

func (b *Bus) Cartridge() *Cartridge { return b.c }                  // returns the Cartridge
func (b *Bus) IsBooting() bool       { return !b.bootROMDone }       // returns boot status
func (b *Bus) IsGBC() bool           { return b.isGBC }              // returns if in CGB mode
func (b *Bus) IsGBCCart() bool       { return b.c.IsCGBCartridge() } // returns if cart supports CGB
func (b *Bus) Model() types.Model    { return b.model }              // returns the current model

func (b *Bus) isDMATransferring() bool { return b.dmaActive || b.dmaRestarting } // DMA transfer in progress

// startDMATransfer initiates a DMA transfer.
func (b *Bus) startDMATransfer() {
	b.dmaActive = true
	b.dmaRestarting = false
	b.doDMATransfer()
	b.s.ScheduleEvent(scheduler.DMAEndTransfer, 640)
}

// doDMATransfer performs a single DMA operation, copying a byte from the source to OAM.
func (b *Bus) doDMATransfer() {
	b.dmaConflict = b.data[b.dmaSource]
	b.data[b.dmaDestination] = b.dmaConflict

	b.dmaSource++
	b.dmaDestination++

	if b.dmaDestination < 0xfea0 {
		b.s.ScheduleEvent(scheduler.DMATransfer, 4)
	}
}

// endDMATransfer ends a DMA transfer.
func (b *Bus) endDMATransfer() {
	b.dmaActive, b.dmaEnabled = false, false
	b.dmaConflicted = 0
	b.dmaConflict = 0xff
}

// colorDMA performs a GDMA/HDMA transfer of length, transferring from source to vRAM.
func (b *Bus) colorDMA(length uint8) {
	for i := uint8(0); i < length; i++ {
		for j := uint8(0); j < 16; j++ {
			// tick the scheduler
			if b.s.DoubleSpeed() {
				b.s.Tick(4)
			} else {
				b.s.Tick(2)
			}

			// perform the transfer
			b.Write(b.hdmaDestination&0x1fff|0x8000, b.Get(b.hdmaSource))

			// increment the source and destination
			b.hdmaSource++
			b.hdmaDestination++
		}
	}
}

func (b *Bus) HandleHDMA() {
	// is there any remaining data to transfer and
	// has the DMA not been paused?
	if b.dmaRemaining > 0 && !b.dmaPaused {
		// update HDMA5 register as the next DMA will tick
		b.Set(types.HDMA5, b.Get(types.HDMA5)&0x80|(b.dmaRemaining-1)&0x7f)
		b.colorDMA(1)
		b.dmaRemaining--
	} else if !b.dmaPaused {
		b.dmaRemaining = 0
		b.dmaComplete = true
		b.Set(types.HDMA5, 0xFF)
	}
}

const (
	VBlankINT = types.Bit0 // ppu vblank
	LCDINT    = types.Bit1 // lcd stat
	TimerINT  = types.Bit2 // timer overflow
	SerialINT = types.Bit3 // serial transfer
	JoypadINT = types.Bit4 // joypad
)

// EnableInterrupts sets IME
func (b *Bus) EnableInterrupts() {
	b.ime = true

	if b.HasInterrupts() {
		b.InterruptCallback(0)
	}
}

// RaiseInterrupt sets the requested interrupt high in types.IF
func (b *Bus) RaiseInterrupt(interrupt uint8) {
	b.data[types.IF] |= interrupt
	if interrupt == VBlankINT || b.CanInterrupt() {
		b.InterruptCallback(interrupt)
	}

	if interrupt == VBlankINT && len(b.GameSharkCodes) > 0 {
		for _, c := range b.GameSharkCodes {
			if c.Address < 0xD000 {
				b.data[c.Address] = c.NewData
			} else if c.Address < 0xE000 {
				if utils.ZeroAdjust(b.Get(types.SVBK)&7) == c.ExternalRAMBank { // banked data - write to bus
					b.data[c.Address] = c.NewData
				} else { // not banked so write to sRAM
					b.wRAM[c.ExternalRAMBank][c.Address&0x0fff] = c.NewData
				}
			}
		}
	}
}

// IRQVector returns the current interrupt vector and clears the corresponding
// interrupt from types.IF.
//
// When an interrupt occurs, there is a chance for the interrupt vector to change
// during the execution of the dispatch handler.
// https://mgba.io/2018/03/09/holy-grail-bugs-revisited/
func (b *Bus) IRQVector(irq uint8) uint16 {
	for i := uint8(0); i < 5; i++ {
		f := uint8(1 << i)

		if irq&b.data[types.IF]&f == f {
			b.data[types.IF] &^= f

			return uint16(0x0040 + i<<3)
		}
	}

	return 0
}

func (b *Bus) CanInterrupt() bool      { return b.ime && b.HasInterrupts() }                 // IME set & pending interrupts
func (b *Bus) DisableInterrupts()      { b.ime = false }                                     // resets IME
func (b *Bus) HasInterrupts() bool     { return b.data[types.IE]&b.data[types.IF]&0x1F > 0 } // pending interrupts
func (b *Bus) InterruptsEnabled() bool { return b.ime }                                      // IME set

type Button = uint8

const (
	ButtonA Button = iota
	ButtonB
	ButtonSelect
	ButtonStart
	ButtonRight
	ButtonLeft
	ButtonUp
	ButtonDown
)

func (b *Bus) Press(i uint8)   { b.buttonState |= 1 << i; b.RaiseInterrupt(JoypadINT) } // presses the requested button
func (b *Bus) Release(i uint8) { b.buttonState &^= 1 << i }                             // releases the requested button

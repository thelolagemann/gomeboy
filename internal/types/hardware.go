package types

import (
	"fmt"
	"sync"
)

var (
	hardwareRegisters = HardwareRegisters{}
	Lock              sync.Mutex
)

// HardwareRegisters is a slice of hardware IO, which
// can be read and written to. The slice is indexed by the
// address of the hardware register ANDed with 0x007F.
type HardwareRegisters [0x80]*HardwareRegister

// Read returns the value of the hardware register for
// the given address. If the hardware register is not
// readable, it returns 0xFF.
func (h HardwareRegisters) Read(address uint16) uint8 {
	// is the hardware register the IE register? as the HardwareRegisters
	// slice is indexed by the address ANDed with 0x007F, the IE register
	// is at index 0x7F, so we need to check for the IE register separately
	if address == 0xFFFF {
		return h[0x7F].Read()
	}
	// does the hardware register exist? if not, return 0xFF
	if h[address&0x007F] == nil || address == 0xFF7F {
		return 0xFF
	}
	return h[address&0x007F].Read()
}

// Write writes the given value to the hardware register
// for the given address. If the hardware register is not
// writable, it does nothing.
func (h HardwareRegisters) Write(address uint16, value uint8) {
	if h[address&0x007F] == nil {
		return
	}
	if address >= 0xFF40 && address <= 0xFF7F {
		//fmt.Printf("Writing %02X to %04X\n", value, address)
	}
	h[address&0x007F].Write(value)
}

// CollectHardwareRegisters collects the registered hardware IO
// and returns them as a slice of HardwareRegisters type. The defined
// hardware IO are then cleared, so that they can be redefined
// (for example, when running multiple instances of the emulator).
func CollectHardwareRegisters() HardwareRegisters {
	hardware := hardwareRegisters
	hardwareRegisters = HardwareRegisters{}
	return hardware
}

// HardwareRegister represents a hardware register of the Game
// Boy. The hardware IO are used to control and
// read the state of the hardware.
type HardwareRegister struct {
	address HardwareAddress
	write   func(v uint8)
	read    func() uint8
	set     func(v interface{})

	writeHandler WriteHandler
}

// HardwareOpt is a function that configures a hardware register,
// such as making it readable, writable, or both.
type HardwareOpt func(*HardwareRegister)

// RegisterHardware IO a hardware register with the given
// address and read/write functions. The read and write functions
// are optional, and may be nil, in which case the register is
// read-only or write-only, respectively. The read and write
// functions are called with the address of the register, and
// the value to be written, or the value to be read, respectively.
func RegisterHardware(address HardwareAddress, write func(v uint8), read func() uint8, opts ...HardwareOpt) {
	h := &HardwareRegister{
		address: address,
		write:   write,
		read:    read,
	}
	for _, opt := range opts {
		opt(h)
	}

	// add hardware register to global map of hardware IO
	hardwareRegisters[address&0x007F] = h
}

func WithWriteHandler(writeHandler func(writeFn func())) HardwareOpt {
	return func(h *HardwareRegister) {
		h.writeHandler = writeHandler
	}
}

func WithSet(set func(v interface{})) HardwareOpt {
	return func(h *HardwareRegister) {
		h.set = set
	}
}

type WriteHandler func(writeFn func())

func (h *HardwareRegister) Read() uint8 {
	// was the hardware register read function write?
	if h.read != nil {
		return h.read()
	}

	// the hardware register is not readable, a panic is thrown
	panic(fmt.Sprintf("hardware: no read function for address 0x%04X", h.address))
}

func (h *HardwareRegister) Write(value uint8) {
	// did the hardware register have a write handler?
	if h.writeHandler != nil {
		// was the hardware register write function write?
		if h.write != nil {
			h.writeHandler(func() {
				h.write(value)
			})
		} else {
			panic(fmt.Sprintf("hardware: no write function for address 0x%04X", h.address))
		}
	} else {
		// was the hardware register write function write?
		if h.write != nil {
			h.write(value)
		} else {
			panic(fmt.Sprintf("hardware: no write function for address 0x%04X", h.address))
		}
	}
}

func (h *HardwareRegister) Set(v interface{}) {
	if h.set != nil {
		h.set(v)
	} else {
		panic(fmt.Sprintf("hardware: no set function for address 0x%04X", h.address))
	}
}

func (h *HardwareRegister) CanSet() bool {
	return h.set != nil
}

// NoRead is a convenience function to return a read function that
// always returns 0xFF. This is useful for hardware IO that
// are not readable.
func NoRead() uint8 {
	return 0xFF
}

// NoWrite is a convenience function to return a write function that
// does nothing. This is useful for hardware IO that are not
// writable.
func NoWrite(v uint8) {
	// do nothing
}

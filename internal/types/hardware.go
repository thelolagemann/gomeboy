package types

import (
	"fmt"
)

var (
	hardwareRegisters = HardwareRegisters{}
)

// HardwareRegisters is a slice of hardware registers, which
// can be read and written to. The slice is indexed by the
// address of the hardware register ANDed with 0x007F.
type HardwareRegisters [0x80]*Hardware

// Read returns the value of the hardware register for
// the given address. If the hardware register is not
// readable, it returns 0xFF.
func (h HardwareRegisters) Read(address uint16) uint8 {
	if h[address&0x007F] == nil {
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
	h[address&0x007F].Write(value)
}

// CollectHardwareRegisters collects the registered hardware registers
// and returns them as a slice of HardwareRegisters type. The defined
// hardware registers are then cleared, so that they can be redefined
// (for example, when running multiple instances of the emulator).
func CollectHardwareRegisters() HardwareRegisters {
	hardware := hardwareRegisters
	hardwareRegisters = HardwareRegisters{}
	return hardware
}

// Hardware represents a hardware register of the Game
// Boy. The hardware registers are used to control and
// read the state of the hardware.
type Hardware struct {
	address HardwareAddress
	set     func(v uint8)
	get     func() uint8

	writeHandler WriteHandler
}

// HardwareOpt is a function that configures a hardware register,
// such as making it readable, writable, or both.
type HardwareOpt func(*Hardware)

// RegisterHardware registers a hardware register with the given
// address and read/write functions. The read and write functions
// are optional, and may be nil, in which case the register is
// read-only or write-only, respectively. The read and write
// functions are called with the address of the register, and
// the value to be written, or the value to be read, respectively.
func RegisterHardware(address HardwareAddress, set func(v uint8), get func() uint8, opts ...HardwareOpt) {
	h := &Hardware{
		address: address,
		set:     set,
		get:     get,
	}
	for _, opt := range opts {
		opt(h)
	}

	// add hardware register to global map of hardware registers
	hardwareRegisters[address&0x007F] = h
}

func WithWriteHandler(writeHandler func(writeFn func())) HardwareOpt {
	return func(h *Hardware) {
		h.writeHandler = writeHandler
	}
}

type WriteHandler func(writeFn func())

func (h *Hardware) Read() uint8 {
	// was the hardware register get function set?
	if h.get != nil {
		return h.get()
	}

	// the hardware register is not readable, a panic is thrown
	panic(fmt.Sprintf("hardware: no read function for address 0x%04X", h.address))
}

func (h *Hardware) Write(value uint8) {
	// did the hardware register have a write handler?
	if h.writeHandler != nil {
		// was the hardware register write function set?
		if h.set != nil {
			h.writeHandler(func() {
				h.set(value)
			})
		} else {
			panic(fmt.Sprintf("hardware: no write function for address 0x%04X", h.address))
		}
	} else {
		// was the hardware register write function set?
		if h.set != nil {
			h.set(value)
		} else {
			panic(fmt.Sprintf("hardware: no write function for address 0x%04X", h.address))
		}
	}
}

// NoRead is a convenience function to return a read function that
// always returns 0xFF. This is useful for hardware registers that
// are not readable.
func NoRead() uint8 {
	return 0xFF
}

// NoWrite is a convenience function to return a write function that
// does nothing. This is useful for hardware registers that are not
// writable.
func NoWrite(v uint8) {
	// do nothing
}
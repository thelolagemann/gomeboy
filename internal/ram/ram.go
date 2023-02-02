// Package ram provides a basic RAM implementation for the
// Game Boy. It is used for the internal RAM and various
// other hardware components.
package ram

import "fmt"

// RAM represents a block of RAM. It is used for the internal
// RAM and various other hardware components. It has a maximum
// size of 65536 bytes, which is the maximum addressable memory
// range of the Game Boy.
type RAM struct {
	data []uint8
	size uint16
}

// NewRAM returns a new RAM instance with the given size.
func NewRAM(size uint16) *RAM {
	return &RAM{
		data: make([]uint8, size),
		size: size,
	}
}

// Read reads the value from the RAM at the given address.
// If the address is out of bounds, it panics.
func (r *RAM) Read(address uint16) uint8 {
	if address > r.size {
		panic(fmt.Sprintf("RAM: address out of bounds: %X", address))
	}
	return r.data[address]
}

// Write writes the given value to the RAM at the given address.
// If the address is out of bounds, it panics.
func (r *RAM) Write(address uint16, value uint8) {
	if address > r.size {
		panic(fmt.Sprintf("RAM: address out of bounds: %d with len %d", address, r.size))
	}
	r.data[address] = value
}

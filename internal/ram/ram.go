// Package ram provides a basic RAM implementation for the
// Game Boy. It is used for the internal RAM and various
// other hardware components.
package ram

import "fmt"

// RAM represents a block of RAM.
type RAM interface {
	Read(address uint16) uint8
	Write(address uint16, value uint8)
}

type Ram struct {
	data map[uint16]uint8
	size uint32
}

// NewRAM returns a new RAM instance with the given size.
func NewRAM(size uint32) *Ram {
	return &Ram{
		data: make(map[uint16]uint8, size),
		size: size,
	}
}

// Read returns the value at the given address.
func (r *Ram) Read(address uint16) uint8 {
	if uint32(address) > r.size {
		panic(fmt.Sprintf("RAM: address out of bounds: %X", address))
	}
	if v, ok := r.data[address]; ok {
		return v
	}
	return 0
}

// Write writes the value to the given address.
func (r *Ram) Write(address uint16, value uint8) {
	if address > uint16(r.size) {
		panic(fmt.Sprintf("RAM: address out of bounds: %d with len %d", address, r.size))
	}
	r.data[address] = value
}

// Package ram provides a basic RAM implementation.
package ram

// RAM represents a block of RAM.
type RAM interface {
	Read(address uint16) uint8
	Write(address uint16, value uint8)
}

type ram struct {
	data map[uint16]uint8
}

// NewRAM returns a new RAM.
func NewRAM(size uint32) RAM {
	return &ram{
		data: make(map[uint16]uint8, size),
	}
}

// Read returns the value at the given address.
func (r *ram) Read(address uint16) uint8 {
	if v, ok := r.data[address]; ok {
		return v
	}
	return 0
}

// Write writes the value to the given address.
func (r *ram) Write(address uint16, value uint8) {
	r.data[address] = value
}

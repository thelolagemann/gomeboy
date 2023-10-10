package io

import "fmt"

type Bus struct {
	data [0xFFFF]byte

	writeHandlers [0xFF]WriteHandler
}

// WriteHandler is a function that handles writing to a memory address.
// It should return the new value to be written back to the memory address.
type WriteHandler func(byte) byte

// ReserveAddress reserves a memory address on the bus.
func (b *Bus) ReserveAddress(addr uint16, handler func(byte) byte) {
	// check to make sure address hasn't already been reserved
	if ok := b.writeHandlers[addr]; ok != nil {
		panic(fmt.Sprintf("address %04X has already been reserved", addr))
	}
	b.writeHandlers[0xFF00+addr] = handler
}

// Get gets the value at the specified memory address.
func (b *Bus) Get(addr uint16) byte {
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

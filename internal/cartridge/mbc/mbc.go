package mbc

// MemoryBankController represents a Memory Bank Controller.
type MemoryBankController interface {
	Read(address uint16) uint8
	Write(address uint16, value uint8)
}

// MBC1 represents a MBC1.
type MBC1 struct {
	rom []byte

	MemoryBankController
}

// NewMBC1 returns a new MBC1.
func NewMBC1(rom []byte) *MBC1 {
	return &MBC1{
		rom: rom,
	}
}

// Read returns the value at the given address.
func (m *MBC1) Read(address uint16) uint8 {
	return m.rom[address]
}

// Write writes the value to the given address.
func (m *MBC1) Write(address uint16, value uint8) {
	//TODO implement me
}

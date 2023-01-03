// Package cartridge provides a Cartridge interface for the DMG and CGB.
// The cartridge holds the game ROM and any external RAM.
package cartridge

import "fmt"

// Cartridge represents a basic game cartridge.
type Cartridge interface {
	Read(address uint16) uint8
	Write(address uint16, value uint8)

	Header() Header
	Title() string
}

type baseCartridge struct {
	rom    []byte
	header Header
}

func (c *baseCartridge) Header() Header {
	return c.header
}

// Title returns an escaped string of the cartridge title.
func (c *baseCartridge) Title() string {
	return c.header.Title
}

func (c *baseCartridge) Read(address uint16) uint8 {
	return c.rom[address]
}

func (c *baseCartridge) Write(address uint16, value uint8) {}

func NewCartridge(rom []byte) Cartridge {
	// parse the cartridge header (0x0100 - 0x014F)
	header := parseHeader(rom[0x100:0x150])

	// print some information about the cartridge
	fmt.Println("Cartridge:")
	fmt.Printf("\t%s\n", header.String())
	switch header.CartridgeType {
	case ROM:
		return &baseCartridge{
			rom:    rom,
			header: header,
		}
	}

	panic("unhandled cartridge type")
}

// NewEmptyCartridge returns an empty cartridge.
func NewEmptyCartridge() Cartridge {
	return &baseCartridge{
		rom:    []byte{},
		header: Header{},
	}
}

// Package cartridge provides a Cartridge interface for the DMG and CGB.
// The cartridge holds the game ROM and any external RAM.
package cartridge

import (
	"fmt"
)

type Cartridge struct {
	MemoryBankController
	header Header
}

func (c *Cartridge) Header() Header {
	return c.header
}

// Title returns an escaped string of the cartridge title.
func (c *Cartridge) Title() string {
	return c.header.Title
}

func NewCartridge(rom []byte) *Cartridge {
	// parse the cartridge header (0x0100 - 0x014F)
	header := parseHeader(rom[0x100:0x150])

	// print some information about the cartridge
	fmt.Println("Cartridge:")
	fmt.Printf("\t%s\n", header.String())
	cart := &Cartridge{}
	switch header.CartridgeType {
	case ROM:
		cart.MemoryBankController = NewROMCartridge(rom)
	case MBC1:
		cart.MemoryBankController = NewMBC1(rom)
	default:
		panic(fmt.Sprintf("cartridge type %d not implemented", header.CartridgeType))
	}

	return cart
}

// NewEmptyCartridge returns an empty cartridge.
func NewEmptyCartridge() *Cartridge {
	return &Cartridge{
		MemoryBankController: NewROMCartridge([]byte{}),
		header:               Header{},
	}
}

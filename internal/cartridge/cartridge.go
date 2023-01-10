// Package cartridge provides a Cartridge interface for the DMG and CGB.
// The cartridge holds the game ROM and any external RAM.
package cartridge

import (
	"crypto/md5"
	"fmt"
)

type Cartridge struct {
	MemoryBankController
	header *Header
}

type MemoryBankController interface {
	Read(address uint16) uint8
	Write(address uint16, value uint8)
}

type RAMController interface {
	Load([]byte)
	Save() []byte
}

func (c *Cartridge) Header() *Header {
	return c.header
}

// Title returns an escaped string of the cartridge title.
func (c *Cartridge) Title() string {
	return c.header.Title
}

// Filename returns the filename for the save file. This is
// simply an md5 hash of the cartridge title.
func (c *Cartridge) Filename() string {
	return string(md5.New().Sum([]byte(c.Title()))) + ".sav"
}

func NewCartridge(rom []byte) *Cartridge {
	// parse the cartridge header (0x0100 - 0x014F)
	header := parseHeader(rom[0x100:0x150])

	// print some information about the cartridge
	fmt.Println("Cartridge:")
	fmt.Printf("\t%s\n", header.String())
	cart := &Cartridge{header: header}
	switch header.CartridgeType {
	case ROM:
		cart.MemoryBankController = NewROMCartridge(rom)
	case MBC1, MBC1RAM, MBC1RAMBATT:
		cart.MemoryBankController = NewMemoryBankedCartridge1(rom, header)
	case MBC3, MBC3RAM, MBC3RAMBATT, MBC3TIMERBATT, MBC3TIMERRAMBATT:
		cart.MemoryBankController = NewMemoryBankedCartridge3(rom)
	default:
		panic(fmt.Sprintf("cartridge type %d not implemented", header.CartridgeType))
	}

	// handle saving and loading of external RAM
	switch header.CartridgeType {
	case MBC1RAMBATT, MBC2BATT, ROMRAMBATT, MBC3TIMERBATT:
	}

	return cart
}

// NewEmptyCartridge returns an empty cartridge.
func NewEmptyCartridge() *Cartridge {
	return &Cartridge{
		MemoryBankController: NewROMCartridge([]byte{}),
		header:               &Header{},
	}
}

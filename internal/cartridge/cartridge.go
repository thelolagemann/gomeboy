// Package cartridge provides a Cartridge interface for the DMG and CGB.
// The cartridge holds the emu ROM and any external RAM.
package cartridge

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
)

type Cartridge struct {
	MemoryBankController
	header *Header
	MD5    string
}

// TODO add IsDirty() bool to RAMController interface
type RAMController interface {
	LoadRAM([]byte)
	SaveRAM() []byte
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
	hash := md5.Sum([]byte(c.Title()))
	return fmt.Sprintf("%s", hex.EncodeToString(hash[:]))
}

func NewCartridge(rom []byte) *Cartridge {
	if len(rom) < 0x150 {
		return NewEmptyCartridge()
	}
	// parse the cartridge header (0x0100 - 0x014F)
	header := parseHeader(rom[0x100:0x150])

	// print some information about the cartridge
	cart := &Cartridge{header: header}
	switch header.CartridgeType {
	case ROM:
		cart.MemoryBankController = NewROMCartridge(rom)
	case MBC1, MBC1RAM, MBC1RAMBATT:
		cart.MemoryBankController = NewMemoryBankedCartridge1(rom, header)
	case MBC2, MBC2BATT:
		cart.MemoryBankController = NewMemoryBankedCartridge2(rom, header)
	case MBC3, MBC3RAM, MBC3RAMBATT, MBC3TIMERBATT, MBC3TIMERRAMBATT:
		cart.MemoryBankController = NewMemoryBankedCartridge3(rom, header)
	case MBC5, MBC5RAM, MBC5RAMBATT, MBC5RUMBLERAMBATT, MBC5RUMBLE, MBC5RUMBLERAM:
		cart.MemoryBankController = NewMemoryBankedCartridge5(rom, header)
	default:
		panic(fmt.Sprintf("cartridge type %s (%02x) not implemented", header.CartridgeType.String(), header.CartridgeType))
	}

	// calculate the md5 hash of the cartridge
	hash := md5.Sum(rom)
	cart.MD5 = hex.EncodeToString(hash[:])

	return cart
}

// NewEmptyCartridge returns an empty cartridge.
func NewEmptyCartridge() *Cartridge {
	return &Cartridge{
		MemoryBankController: NewROMCartridge([]byte{}),
		header:               &Header{},
	}
}

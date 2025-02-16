package types

import (
	"crypto/md5"
	"strings"
)

type Model int // The Model used in emulation.

const (
	Unset  Model = iota // Unset - Model hasn't been set - behaves as DMGABC
	DMG0                // DMG0 - early Game Boy, only released in Japan
	DMGABC              // DMGABC - Standard Game Boy
	CGB0                // CGB0 -  early Game Boy Colour, only released in Japan
	CGBABC              // CGBABC - Standard Game Boy Colour
	MGB                 // MGB - Pocket Game Boy
	SGB                 // SGB - Super Game Boy
	SGB2                // SGB2 - Super Game Boy 2
	AGB                 // AGB - Game Boy Advance
)

var ModelNames = map[Model]string{
	DMG0:   "DMG0",
	DMGABC: "DMG",
	CGB0:   "CGB0",
	CGBABC: "CGB",
	MGB:    "MGB",
	SGB:    "SGB",
	SGB2:   "SGB2",
	AGB:    "AGB",
	Unset:  "Unset",
}

// StringToModel converts a string to a Model.
func StringToModel(s string) Model {
	for m, n := range ModelNames {
		if n == strings.ToUpper(s) {
			return m
		}
	}

	return Unset
}

func (m Model) String() string {
	return ModelNames[m]
}

// ModelBootROMChecksums - MD5 checksums of the boot ROMs of each model
var ModelBootROMChecksums = map[[16]byte]Model{
	{0xA8, 0xF8, 0x4A, 0x0A, 0xC4, 0x4D, 0xA5, 0xD3, 0xF0, 0xEE, 0x19, 0xF9, 0xCE, 0xA8, 0x0A, 0x8C}: DMG0,
	{0x32, 0xfb, 0xbd, 0x84, 0x16, 0x8d, 0x34, 0x82, 0x95, 0x6e, 0xb3, 0xc5, 0x05, 0x16, 0x37, 0xf5}: DMGABC,
	{0x71, 0xa3, 0x78, 0xe7, 0x1f, 0xf3, 0x0b, 0x2d, 0x8a, 0x1f, 0x02, 0xbf, 0x5c, 0x78, 0x96, 0xaa}: MGB,
	{0xd5, 0x74, 0xd4, 0xf9, 0xc1, 0x2f, 0x30, 0x50, 0x74, 0x79, 0x8f, 0x54, 0xc0, 0x91, 0xa8, 0xb4}: SGB,
	{0xe0, 0x43, 0x0b, 0xca, 0x99, 0x25, 0xfb, 0x98, 0x82, 0x14, 0x8f, 0xd2, 0xdc, 0x24, 0x18, 0xc1}: SGB2,
	{0x7c, 0x77, 0x3f, 0x3c, 0x0b, 0x01, 0xcb, 0x73, 0xbc, 0xa8, 0xe8, 0x32, 0x27, 0x28, 0x7b, 0x7f}: CGB0,
	{0xdb, 0xfc, 0xe9, 0xdb, 0x9d, 0xea, 0xa2, 0x56, 0x7f, 0x6a, 0x84, 0xfd, 0xe5, 0x5f, 0x96, 0x80}: CGBABC,
	{0xe6, 0xce, 0xfb, 0x5f, 0x7d, 0x35, 0x2f, 0xab, 0x66, 0x81, 0x98, 0x97, 0x63, 0x91, 0x7c, 0x73}: AGB,
}

func Which(rom []byte) Model {
	if m, ok := ModelBootROMChecksums[md5.Sum(rom)]; ok {
		return m
	}
	return Unset
}

// ModelIO - model specific starting IO registers.
var ModelIO = map[Model]map[HardwareAddress]interface{}{
	Unset:  {DIV: uint16(0xABC9)},
	DMG0:   {DIV: uint16(0x182F), LY: uint8(0x92)},
	DMGABC: {DIV: uint16(0xABC9)},
	CGBABC: {P1: uint8(0xFF), DIV: uint16(0x2675), BCPS: uint8(0xC8), OCPS: uint8(0xD0)},
	CGB0:   {DIV: uint16(0x2881)},
	SGB:    {P1: uint8(0xFF), DIV: uint16(0xD85F), NR52: uint8(0xF0), STAT: uint8(0x85), LY: uint8(0x00)},
	SGB2:   {DIV: uint16(0xD84F)},
	AGB:    {DIV: uint16(0x267B)},
}

// ModelRegisters - model specific starting CPU registers.
var ModelRegisters = map[Model][]uint8{
	Unset:  {0x01, 0x00, 0xFF, 0x13, 0x00, 0xC1, 0x84, 0x03}, // default to DMG registers
	DMG0:   {0x01, 0x00, 0xFF, 0x13, 0x00, 0xC1, 0x84, 0x03},
	DMGABC: {0x01, 0xB0, 0x00, 0x13, 0x00, 0xD8, 0x01, 0x4D},
	CGB0:   {0x11, 0x80, 0x00, 0x00, 0x00, 0x08, 0x00, 0x7C}, // TODO does CGB0 have the same starting registers?
	CGBABC: {0x11, 0x80, 0x00, 0x00, 0x00, 0x08, 0x00, 0x7C},
	MGB:    {0xFF, 0xB0, 0x00, 0x13, 0x00, 0xD8, 0x01, 0x4D},
	SGB:    {0x01, 0x00, 0x00, 0x14, 0x00, 0x00, 0xC0, 0x60},
	SGB2:   {0xFF, 0x00, 0x00, 0x14, 0x00, 0x00, 0xC0, 0x60},
	AGB:    {0x11, 0x00, 0x01, 0x00, 0x00, 0x08, 0x00, 0x7C},
}

// CommonIO - common starting IO registers.
var CommonIO = map[HardwareAddress]interface{}{
	P1:   uint8(0xCF),
	TAC:  uint8(0xF8),
	NR10: uint8(0x80),
	NR11: uint8(0xBF),
	NR12: uint8(0xF3),
	NR14: uint8(0x00),
	NR21: uint8(0x3F),
	NR22: uint8(0x00),
	NR24: uint8(0xBF),
	NR30: uint8(0x7F),
	NR31: uint8(0xFF),
	NR32: uint8(0x9F),
	NR33: uint8(0xBF),
	NR41: uint8(0xFF),
	NR42: uint8(0x00),
	NR43: uint8(0x00),
	NR50: uint8(0x77),
	NR51: uint8(0xF3),
	NR52: uint8(0xF1),
	BGP:  uint8(0xFC),
	LCDC: uint8(0x91),
	IF:   uint8(0xE1),
	STAT: uint8(0x87),
}

package types

import (
	"github.com/thelolagemann/gomeboy/internal/scheduler"
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

// ModelEvents - model specific starting events (this isn't accurate at all:)
var ModelEvents = map[Model][]Event{
	DMG0: {
		{scheduler.APUChannel1, 48},
		{scheduler.APUSample, 93},
		{scheduler.PPUStartVBlank, 252},
		{scheduler.APUFrameSequencer, 984},
		{scheduler.APUChannel3, 984},
	},
	SGB: {
		{scheduler.APUSample, 64},
		{scheduler.PPUHBlank, 196},
		{scheduler.APUFrameSequencer, 952},
		{scheduler.APUChannel3, 952},
	},
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

type Event struct {
	Type  scheduler.EventType
	Cycle uint64
}

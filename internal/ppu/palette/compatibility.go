package palette

import "fmt"

// "The ROM assigns color palettes to certain monochrome Game Boy games by computing a hash of the ROM's header title
// for every Nintendo Licensee game and checking it against an internal database of hashes. The resulting index is then
// used to obtain an entry ID (from 0x00 up to and including 0x1C) and shuffling flags (a 3-bit bitfield). An entry is
// a triplet of palettes, and the "shuffling flags" replace some of the triplet's palettes with others. In particular,
// shuffling flag value 0x05 causes all 3 members of a triplet to be used, and 0x00 causes both OBJ palettes to be
// overwritten with copies of the BG palette (which never budges). Since bit 2 of the shuffling flags overrides bit 1,
// values 0x06 and 0x07 are never used. " - https://tcrf.net/Game_Boy_Color_Bootstrap_ROM#Unused_Palette_Configurations

// A CompatibilityPaletteEntry represents a single entry in the compatibility palette table. It contains the
// background, object 0 and object 1 palettes.
type CompatibilityPaletteEntry struct {
	BG, OBJ0, OBJ1 [4]RGB
}

type TableEntry map[uint8]CompatibilityPaletteEntry

type Table map[uint8]TableEntry

var CompatibilityPalettes = Table{
	0x00: {
		0x03: {
			BG: [4]RGB{
				{0xFF, 0xFF, 0xFF},
				{0xAD, 0xAD, 0x84},
				{0x42, 0x73, 0x7B},
				{0x00, 0x00, 0x00},
			},
		},
	},
	0x05: {
		0x03: {
			BG: [4]RGB{
				{0xFF, 0xFF, 0xFF},
				{0x52, 0xFF, 0x00},
				{0xFF, 0x42, 0x00},
				{0x00, 0x00, 0x00},
			},
			OBJ0: [4]RGB{
				{0xFF, 0xFF, 0xFF},
				{0xFF, 0x84, 0x84},
				{0x94, 0x3A, 0x3A},
				{0x00, 0x00, 0x00},
			},
			OBJ1: [4]RGB{
				{0xFF, 0xFF, 0xFF},
				{0xFF, 0x84, 0x84},
				{0x94, 0x3A, 0x3A},
				{0x00, 0x00, 0x00},
			},
		},
	},
	0x07: {
		0x00: {
			BG: [4]RGB{
				{0xFF, 0xFF, 0xFF},
				{0xFF, 0xFF, 0x00},
				{0xFF, 0x00, 0x00},
				{0x00, 0x00, 0x00},
			},
			OBJ0: [4]RGB{
				{0xFF, 0xFF, 0xFF},
				{0xFF, 0xFF, 0x00},
				{0xFF, 0x00, 0x00},
				{0x00, 0x00, 0x00},
			},
			OBJ1: [4]RGB{
				{0xFF, 0xFF, 0xFF},
				{0xFF, 0xFF, 0x00},
				{0xFF, 0x00, 0x00},
				{0x00, 0x00, 0x00},
			},
		},
	},
	0x09: {
		0x05: {
			BG: [4]RGB{
				{0xFF, 0xFF, 0xCE},
				{0x63, 0xEF, 0xEF},
				{0x9c, 0x84, 0x31},
				{0x5A, 0x5A, 0x5A},
			},
			OBJ0: [4]RGB{
				{0xFF, 0xFF, 0xFF},
				{0xFF, 0x73, 0x00},
				{0x94, 0x42, 0x00},
				{0x00, 0x00, 0x00},
			},
			OBJ1: [4]RGB{
				{0xFF, 0xFF, 0xFF},
				{0x63, 0xAF, 0xFF},
				{0x00, 0x00, 0xFF},
				{0x00, 0x00, 0x00},
			},
		},
	},
	0x10: {
		0x01: {
			BG: [4]RGB{
				{0xFF, 0xFF, 0xFF},
				{0xFF, 0x84, 0x84},
				{0x94, 0x3A, 0x3A},
				{0x00, 0x00, 0x00},
			},
			OBJ0: [4]RGB{
				{0xFF, 0xFF, 0xFF},
				{0x7B, 0xFF, 0x31},
				{0x00, 0x84, 0x00},
				{0x00, 0x00, 0x00},
			},
			OBJ1: [4]RGB{
				{0xFF, 0xFF, 0xFF},
				{0xFF, 0x84, 0x84},
				{0x94, 0x3A, 0x3A},
				{0x00, 0x00, 0x00},
			},
		},
	},
	0x1C: {
		0x03: {
			BG: [4]RGB{
				{0xFF, 0xFF, 0xFF},
				{0x7B, 0xFF, 0x31},
				{0x00, 0x63, 0xC6},
				{0x00, 0x00, 0x00},
			},
			OBJ0: [4]RGB{
				{0xFF, 0xFF, 0xFF},
				{0xFF, 0x84, 0x84},
				{0x94, 0x39, 0x39},
				{0x00, 0x00, 0x00},
			},
			OBJ1: [4]RGB{
				{0xFF, 0xFF, 0xFF},
				{0xFF, 0x84, 0x84},
				{0x94, 0x39, 0x39},
				{0x00, 0x00, 0x00},
			},
		},
	},
}

type hashEntry struct {
	EntryID        uint8
	Disambiguation uint8

	CompatibilityPaletteEntry
}

var CompatibilityHashEntries = []hashEntry{
	{0x00, 0x03, CompatibilityPalettes[0x00][0x03]},

	// Mario & Yoshi (E)
	// Yoshi (USA)
	// Yoshi no Tamago (J)
	{0x3D, 0x00, CompatibilityPalettes[0x05][0x03]},
	{0x6A, 0x49, CompatibilityPalettes[0x05][0x03]},

	// Pocket Monsters - Pickachu (J) (Rev 0A)
	// Pocket Monsters - Pickachu (J) (Rev B)
	// Pocket Monsters - Pickachu (J) (Rev C)
	// Pocket Monsters - Pickachu (J) (Rev D)
	// Tetris (World)
	// Tetris (World) (Rev A)
	{0x15, 0x00, CompatibilityPalettes[0x07][0x00]},
	{0xDB, 0x00, CompatibilityPalettes[0x07][0x00]},

	// Super Mario Land 2 - 6 Golden Coins (USA, Europe)
	// Super Mario Land 2 - 6 Golden Coins (USA, Europe) (Rev A)
	// Super Mario Land 2 - 6 Golden Coins (USA, Europe) (Rev B)
	// Super Mario Land 2 - 6-tsu no Kinka (Japan)
	// Super Mario Land 2 - 6-tsu no Kinka (Japan) (Rev B)
	{0xC9, 0x00, CompatibilityPalettes[0x09][0x05]},
	// Game Boy Camera Gold (USA)
	// Pocket Monsters Aka (Japan)
	// Pocket Monsters Aka (Japan) (Rev A)
	// Pokemon - Edicion Roja (Spain)
	// Pokemon - Red Version (USA, Europe)
	// Pokemon - Rote Edition (Germany)
	// Pokemon - Version Rouge (France)
	// Pokemon - Versione Rossa (Italy)
	{0x14, 0x00, CompatibilityPalettes[0x10][0x01]},
}

func GetCompatibilityPaletteEntry(hash uint16) (CompatibilityPaletteEntry, bool) {
	for _, entry := range CompatibilityHashEntries {
		if entry.EntryID == uint8(hash>>8) && (entry.Disambiguation == 0 || entry.Disambiguation == uint8(hash)) {
			fmt.Printf("found entry: %x\n", entry.EntryID)
			return entry.CompatibilityPaletteEntry, true
		}
	}

	return CompatibilityPaletteEntry{}, false
}

type RGB [3]uint8

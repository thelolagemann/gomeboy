package io

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/thelolagemann/gomeboy/internal/types"
)

// bootROMModels is a map of boot rom models, with the key
// being the boot ROM checksum, and the value being the model type
var bootROMModels = map[string]types.Model{
	DMG0:    types.DMG0,
	DMG:     types.DMGABC,
	MGB:     types.MGB,
	SGB:     types.SGB,
	SGB2:    types.SGB2,
	CGB0:    types.CGB0,
	CGB:     types.CGBABC,
	CGB_AGB: types.AGB,
	// emulate clones as DMG for now
	FORTUNE:      types.DMGABC,
	GAME_FIGHTER: types.DMGABC,
	MAX_STATION:  types.DMGABC,
}

// Which returns the model type of the boot ROM.
func Which(rom []byte) types.Model {
	sum := md5.Sum(rom)
	if m, ok := bootROMModels[hex.EncodeToString(sum[:])]; ok {
		return m
	}
	return types.DMGABC
}

const (
	// DMG0 is the checksum of the DMG early boot ROM,
	// a variant that was found in very early DMG units and
	// only ever sold in Japan. It has a different behaviour
	// than the DMG boot ROM, in that in the case of a boot
	// failure, it will flash the screen, rather than hanging
	// after the Nintendo logo.
	DMG0 = "a8f84a0ac44da5d3f0ee19f9cea80a8c"
	// DMG is the checksum of the DMG boot rom, which is
	// the most common boot ROM found in the original DMG-01
	// models.
	DMG = "32fbbd84168d3482956eb3c5051637f5"
	// MGB is the checksum of the MGB boot ROM, which differs
	// only by a single byte from the DMG boot ROM, loading
	// the value 0xFF into the A register, rather than 0x01.
	// This can be used by games to detect that it is running
	// on MGB hardware, rather than a DMG.
	MGB = "71a378e71ff30b2d8a1f02bf5c7896aa"
	// SGB is the checksum of the SGB boot ROM, which has
	// significant differences in behaviour to the DMG boot ROM.
	// Instead of showing a logo animation, it instead sends the
	// ROM cartridge header to the SNES via the SGB, and the
	// SNES then shows an animation before displaying the game.
	SGB = "d574d4f9c12f305074798f54c091a8b4"
	// SGB2 is the checksum of the SGB2 boot ROM, similar in
	// differences as the MGB boot ROM is to the DMG boot ROM,
	// differing only by a single byte, which loads the value
	// 0xFF into the A register, rather than 0x01. This can be
	// used by games to detect that it is running on SGB2
	// hardware, rather than on the original SGB.
	SGB2 = "e0430bca9925fb9882148fd2dc2418c1"
	// CGB0 is the checksum of the CGB early boot ROM, a variant
	// that was found in very early CGB units. It has a few
	// differences in behaviour to the CGB boot ROM
	//  - it does not initialize Wave RAM
	//  - has two redundant writes to RAM
	//  - uses less optimized code to load the logo
	CGB0 = "7c773f3c0b01cb73bca8e83227287b7f"
	// CGB is the checksum of the CGB boot rom, which is
	// the boot rom found in the most common CGB models. It
	// has a larger size than the DMG boot ROMs (2304 bytes),
	// and has increased functionality to support the CGB
	// hardware.
	CGB = "dbfce9db9deaa2567f6a84fde55f9680"
	// CGB_AGB is the checksum of the boot ROM found in the GBA's
	// GBC compatibility mode.
	CGB_AGB = "e6cefb5f7d352fab6681989763917c73"
	// FORTUNE is the checksum of the boot ROM found in the
	// Game Boy clone "Fortune/Bitman 3000B".
	FORTUNE = "92ed4eca17d61fcd53f8a64c3ce84743"
	// GAME_FIGHTER is the checksum of the boot ROM found in the
	// Game Boy clone "Game Fighter".
	GAME_FIGHTER = "6a7b8ee12a793f66a969c6a2b8926cc9"
	// MAX_STATION is the checksum of the boot ROM found in the
	// Game Boy clone "Maxstation".
	MAX_STATION = "77a7021db824010a678791f6d062943d"
)

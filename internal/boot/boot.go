// Package boot provides a boot ROM implementation for the Game Boy. Whilst
// this package is not strictly required for the emulator to function, it
// can be used to emulate the boot process of the Game Boy.
package boot

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
)

// ROM represents a boot ROM for the Game Boy. When the Game Boy first
// powers on, the boot ROM is mapped to memory addresses 0x0000 -
// 0x00FF (or 0x0000 - 0x00FF & 0x0200 - 0x08FF for the CGB).
//
// The boot ROM performs a series of tasks, such as initializing the
// hardware, setting the stack pointer, scrolling the Nintendo logo, etc.
//
// Once the boot ROM has completed its tasks, it is unmapped from memory
// (by writing to the types.BDIS register), and the cartridge is mapped
// over the boot ROM, thus starting the cartridge execution, and preventing
// the boot ROM from being executed again.
type ROM struct {
	raw      []byte // the raw boot rom
	checksum string // the MD5 checksum of the boot rom
}

// LoadBootROM loads a boot ROM into a new ROM struct and returns a
// pointer to it. The function ensures that the input raw slice has a
// valid length for either DMG/MGB/SGB (256 bytes) or CGB (2304 bytes).
// If the length is invalid, the function will panic. The function also
// calculates the MD5 checksum of the boot rom, and stores it in the
// ROM struct.
func LoadBootROM(b []byte) *ROM {
	// ensure correct lengths
	if len(b) != 256 && len(b) != 2304 { // 256 bytes for DMG/MGB/SGB, 2304 bytes for CGB
		panic(fmt.Sprintf("boot: invalid boot rom length: %d", len(b)))
	}

	// calculate checksum
	bootChecksum := md5.Sum(b)

	return &ROM{
		raw:      b,
		checksum: hex.EncodeToString(bootChecksum[:]),
	}
}

// Read returns the byte at the given address.
func (b *ROM) Read(addr uint16) byte {
	return b.raw[addr]
}

// Checksum returns the MD5 checksum of the boot rom.
func (b *ROM) Checksum() string {
	if b == nil {
		return ""
	}
	return b.checksum
}

// Model returns the model of the boot rom. The model
// is determined by the checksum of the boot rom.
func (b *ROM) Model() string {
	if b == nil {
		return "none"
	}
	if model, ok := knownBootROMChecksums[b.checksum]; ok {
		return model
	}
	return "unknown"
}

// knownBootROMChecksums is a map of known boot rom checksums,
// with the key being the checksum, and the value being the
// model of the boot rom.
var knownBootROMChecksums = map[string]string{
	DMG0:         "Game Boy (DMG-0)",
	DMG:          "Game Boy (DMG-01)",
	MGB:          "Game Boy Pocket",
	SGB:          "Super Game Boy",
	SGB2:         "Super Game Boy 2",
	CGB0:         "Game Boy Color (CGB-0)",
	CGB:          "Game Boy Color (CGB-A/B/C/D/E)",
	CGB_AGB:      "Game Boy Advance (AGB-001)",
	FORTUNE:      "Fortune/Bitman 3000B",
	GAME_FIGHTER: "Game Fighter",
	MAX_STATION:  "Max Station",
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

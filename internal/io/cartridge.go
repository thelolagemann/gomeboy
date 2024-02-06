//go:generate stringer -type=CartridgeType -output=cartridge_string.go
package io

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/types"
	"strings"
)

type CGBFlag = uint8 // CGBFlag specifies the level of CGB support in a Cartridge.

const (
	CGBFlagEnhanced CGBFlag = iota // The game supports CGB enhancements, but is backwards compatible.
	CGBFlagCGBOnly                 // The game works on CGB only.
	CGBFlagUnset                   // No CGB support has been specified, most likely a regular Game Boy game.
)

type CartridgeType uint8 // CartridgeType represents the hardware present in a Cartridge.

const (
	ROM               CartridgeType = 0x00
	MBC1              CartridgeType = 0x01
	MBC1RAM           CartridgeType = 0x02
	MBC1RAMBATT       CartridgeType = 0x03
	MBC2              CartridgeType = 0x05
	MBC2BATT          CartridgeType = 0x06
	ROMRAM            CartridgeType = 0x08
	ROMRAMBATT        CartridgeType = 0x09
	MMM01             CartridgeType = 0x0B
	MMM01RAM          CartridgeType = 0x0C
	MMM01RAMBATT      CartridgeType = 0x0D
	MBC3TIMERBATT     CartridgeType = 0x0F
	MBC3TIMERRAMBATT  CartridgeType = 0x10
	MBC3              CartridgeType = 0x11
	MBC3RAM           CartridgeType = 0x12
	MBC3RAMBATT       CartridgeType = 0x13
	MBC5              CartridgeType = 0x19
	MBC5RAM           CartridgeType = 0x1A
	MBC5RAMBATT       CartridgeType = 0x1B
	MBC5RUMBLE        CartridgeType = 0x1C
	MBC5RUMBLERAM     CartridgeType = 0x1D
	MBC5RUMBLERAMBATT CartridgeType = 0x1E
	POCKETCAMERA      CartridgeType = 0xFC
	BANDAITAMA5       CartridgeType = 0xFD
	HUDSONHUC3        CartridgeType = 0xFE
	HUDSONHUC1        CartridgeType = 0xFF
)

type Cartridge struct {
	ROM []byte
	RAM []byte

	// header fields - credits
	// https://gbdev.io/pandocs/The_Cartridge_Header.html
	// https://gbdev.gg8.se/wiki/articles/The_Cartridge_Header

	Title                string  // $0134-$0143 Title of the game in uppercase ASCII.
	ManufacturerCode     string  // $013F-$0142 4-character ManufacturerCode (in uppercase ASCII) - purpose remains unknown
	CGBFlag                      // $0142 - Indicates level of CGB support
	NewLicenseeCode      [2]byte // $0144-$0145 2-character ASCII "licensee code"
	SGBFlag              bool    // $0146 - Specifies whether the game supports SGB functions
	CartridgeType                // $0147 - Specifies the hardware present on a Cartridge.
	ROMSize              int     // $0148 - Specifies how much ROM is on the Cartridge, calculated by 32 KiB x (1<<value)
	RAMSize              int     // $0149 - Specifies how much RAM is present on the Cartridge, if any.
	DestinationCode      byte    // $014A - Specifies whether the game is intended to be sold in Japan or elsewhere
	OldLicenseeCode      byte    // $014B - Specifies the game's publisher; see NewLicenseeCode if val == $33
	MaskROMVersionNumber uint8   // $014C - Specifies the version of the game. It is usually $00
	HeaderChecksum       uint8   // $014D - 8-Bit checksum of header bytes $0134-$014C
	GlobalChecksum       uint16  // $014E-$014F 16-bit (big endian) checksum of Cartridge ROM (excluding these bytes)

	RumbleCallback func(bool)

	// internal fields
	ramEnabled bool
	rtcEnabled bool

	mbc1BankShift uint8
	mbc1Bank1     uint8
	mbc1Bank2     uint8
	mbc1Mode      bool
	mbc1MultiCart bool

	// rtc
	rtcS, rtcM, rtcH, rtcDL, rtcDH                                    uint8
	rtcLatchedS, rtcLatchedM, rtcLatchedH, rtcLatchedDL, rtcLatchedDH uint8
	rtcRegister                                                       uint8
	rtcLatchValue                                                     uint8
	rtcLastUpdate                                                     uint64
	rtcLatched                                                        bool

	ramOffset uint32
	romOffset uint32
	b         *Bus
}

// detectMultiCart uses heuristics to detect if the cartridge is a multi-cart ROM.
func (c *Cartridge) detectMultiCart() {
	if c.ROMSize == (1024 * 1024) {
		logoCounts := 0
		compare := true

		// copy what should be the first logo from the ROM
		logo := make([]byte, 48)
		copy(logo, c.ROM[0x0104:0x0134])

		for bank := 0; bank < 4; bank++ {
			if !bytes.Equal(logo, c.ROM[bank*0x4000+0x0104:bank*0x4000+0x0134]) {
				compare = false
				break
			}

			if compare {
				logoCounts += 1
			}
		}

		// more than 1 logo is likely a multicart
		if logoCounts > 1 {
			c.mbc1MultiCart = true
			c.mbc1BankShift = 4
		}
	}
}

// parseHeader parses the cartridge header from Cartridge.ROM.
func (c *Cartridge) parseHeader() {
	// parse the mode of the cartridge to determine how to parse the title accordingly
	switch c.ROM[0x0143] {
	case 0x80:
		c.CGBFlag = CGBFlagEnhanced
	case 0xC0:
		c.CGBFlag = CGBFlagCGBOnly
	default:
		c.CGBFlag = CGBFlagUnset

		// TODO why was i setting the colourisation palette in here?
	}

	// CGB cartridge header reduced the title length to 15, and then some months later to 11
	if c.CGBFlag == CGBFlagUnset {
		c.Title = string(c.ROM[0x0134:0x0144])
	} else {
		c.Title = string(c.ROM[0x0134:0x0143]) // TODO determine which games used manufacturer code?
	}

	// the title would be padded with $00 bytes if it was shorter than the title length
	c.Title = strings.Replace(c.Title, "\x00", "", -1)

	c.ManufacturerCode = string(c.ROM[0x013F:0x0143])
	c.NewLicenseeCode = [2]byte{c.ROM[0x0144], c.ROM[0x0145]}
	c.SGBFlag = c.ROM[0x0146] == 3
	c.CartridgeType = CartridgeType(c.ROM[0x0147])
	c.ROMSize = (32 * 1024) * (1 << c.ROM[0x0148])
	c.RAMSize = map[uint8]int{
		0x00: 0,          // 0KiB
		0x01: 0,          // 0KiB
		0x02: 8 * 1024,   // 8KiB
		0x03: 32 * 1024,  // 32KiB
		0x04: 128 * 1024, // 128KiB
		0x05: 64 * 1024,  // 64KiB
	}[c.ROM[0x0149]]
	c.DestinationCode = c.ROM[0x014A]
	c.OldLicenseeCode = c.ROM[0x014B]
	c.MaskROMVersionNumber = c.ROM[0x014C]
	c.HeaderChecksum = c.ROM[0x014D]
	c.GlobalChecksum = binary.BigEndian.Uint16(c.ROM[0x014E:0x0150])
}

// updateROMBank sets the appropriate rom offset to be used depending on the provided
// bank value and the length of Cartridge.ROM.
func (c *Cartridge) updateROMBank(bank uint16) {
	c.romOffset = (uint32(bank) * 0x4000) % uint32(len(c.ROM))
	c.b.CopyTo(0x4000, 0x8000, c.ROM[c.romOffset:])
}

// updateRAMBank sets the appropriate ram offset to be used depending on the provided
// bank value and the length of Cartridge.RAM.
func (c *Cartridge) updateRAMBank(bank uint8) {
	// do nothing if the cartridge has no ram
	if len(c.RAM) == 0 || len(c.RAM) == 8192 {
		return
	}

	c.b.CopyFrom(0xA000, 0xC000, c.RAM[c.ramOffset:])          // copy current RAM bank from bus -> cart
	c.ramOffset = (uint32(bank) * 0x2000) % uint32(len(c.RAM)) // set new RAM offset
	c.b.CopyTo(0xA000, 0xC000, c.RAM[c.ramOffset:])            // copy new RAM bank from cart -> bus
}

// updateRTC sets the RTC registers based on how many cycles have passed since the last read.
// TODO make configurable to sync to host
func (c *Cartridge) updateRTC() {
	delta := c.b.s.Cycle() - c.rtcLastUpdate

	if c.rtcDH>>6&1 == 0 && delta > 4194304 {
		fmt.Println("a second has passed, time to update", delta/4194304, c.rtcS, c.b.s.Cycle(), c.rtcLastUpdate)

		var days uint16

		deltaSeconds := uint8(delta / 4194304)
		c.rtcS += deltaSeconds % 60
		if c.rtcS >= 60 {
			c.rtcS -= 60
			c.rtcM++
		}
		deltaSeconds /= 60
		c.rtcM += deltaSeconds % 60
		if c.rtcM >= 60 {
			c.rtcM -= 60
			c.rtcH++
		}
		deltaSeconds /= 60
		c.rtcH += deltaSeconds % 24
		if c.rtcH >= 24 {
			c.rtcH -= 24
			days++
		}
		deltaSeconds /= 24
		days += uint16(deltaSeconds) + uint16(c.rtcDH) + uint16(c.rtcDH&1)<<8
		if days >= 512 {
			days %= 512
			c.rtcDH ^= types.Bit7
		}

		c.rtcDL = uint8(days & 0xff)
		c.rtcDH &= 0xfe
		if days >= 256 {
			c.rtcDH |= 1
		}

		c.rtcLastUpdate = c.b.s.Cycle()
	}
}

// Destination returns the destination as specified in the cartridge header.
func (c *Cartridge) Destination() string {
	switch c.DestinationCode {
	case 0:
		return "Japanese"
	case 1:
		return "Non-Japanese"
	default:
		return "Unknown"
	}
}

// IsCGBCartridge returns true if the cartridge makes use of CGB features, optionally or not.
func (c *Cartridge) IsCGBCartridge() bool {
	return c.CGBFlag < CGBFlagUnset
}

// Licensee returns the Licensee of the cartridge, according to the parsed header data.
func (c *Cartridge) Licensee() string {
	if c.OldLicenseeCode == 0x33 {
		// infer 2 byte slice as ASCII string
		return newLicenseeCodeMap[string(c.NewLicenseeCode[:])]
	}

	return oldLicenseeCodeMap[c.OldLicenseeCode]
}

// SGB returns true if the Cartridge supports SGB functions.
func (c *Cartridge) SGB() bool {
	return c.SGBFlag && c.OldLicenseeCode == 0x33
}

// String implements the fmt.Stringer interface.
func (c *Cartridge) String() string {
	return fmt.Sprintf("%s (%s) | (%dKiB|%dKiB) %s", c.Title, c.Licensee(), c.ROMSize/1024, c.RAMSize/1024, c.CartridgeType)
}

// Write writes an 8-bit value to the cartridge.
func (c *Cartridge) Write(address uint16, value uint8) {
	switch {
	case c.CartridgeType == ROM:
		return // ROM is (R)ead-(O)nly (M)emory
	case address < 0x2000:
		switch c.CartridgeType {
		case MBC1RAM, MBC1RAMBATT, MBC3RAM, MBC3RAMBATT, MBC3TIMERBATT, MBC3TIMERRAMBATT, MBC5RAM, MBC5RAMBATT, MBC5RUMBLERAM, MBC5RUMBLERAMBATT:
			c.ramEnabled = value&0x0f == 0x0a && c.CartridgeType != MBC3TIMERBATT
			c.rtcEnabled = value&0x0f == 0x0a && (c.CartridgeType == MBC3TIMERBATT || c.CartridgeType == MBC3TIMERRAMBATT)
			return
		}

		fallthrough
	case address < 0x3000: // MBC5 being unique
		switch {
		case c.CartridgeType >= MBC5 && c.CartridgeType <= MBC5RUMBLERAMBATT:
			romBank := uint16(c.romOffset / 0x4000)
			romBank = romBank&0x0100 + uint16(value) // lower 8 bits
			c.updateROMBank(romBank)

			return
		}
		fallthrough
	case address < 0x4000:
		switch c.CartridgeType {
		case MBC1, MBC1RAM, MBC1RAMBATT:
			value &= 0x1f // 5-bit value

			// can't write a value of 0
			if value == 0 {
				value = 1
			}

			// and only a 4-bit value on multicarts
			if c.mbc1MultiCart {
				value &= 0x0f
			}

			c.mbc1Bank1 = value

			c.updateROMBank(uint16(c.mbc1Bank2<<c.mbc1BankShift | value))
		case MBC2, MBC2BATT:
			// writes with bit 8 set are ROM bank, otherwise RAM toggle
			if address&0x100 == 0x100 {
				value &= 0x0f // 4-bit

				// like MBC1, values of 0 can't be written
				if value == 0 {
					value = 1
				}
				c.updateROMBank(uint16(value))
			} else {
				c.ramEnabled = value&0x0f == 0x0a
			}
		case MBC3, MBC3RAM, MBC3RAMBATT, MBC3TIMERBATT, MBC3TIMERRAMBATT:
			if value == 0 {
				value = 1
			}
			c.updateROMBank(uint16(value))
		case MBC5, MBC5RAM, MBC5RAMBATT, MBC5RUMBLE, MBC5RUMBLERAM, MBC5RUMBLERAMBATT:
			romBank := uint16(c.romOffset / 0x4000)
			romBank = romBank&0x00ff + uint16(value&1)<<8
			c.updateROMBank(romBank)
		}
	case address < 0x6000:
		switch c.CartridgeType {
		case MBC1, MBC1RAM, MBC1RAMBATT:
			// bank2 (<0x6000) is a 2-bit value
			value &= 3

			c.mbc1Bank2 = value

			c.updateROMBank(uint16(c.mbc1Bank1 | c.mbc1Bank2<<c.mbc1BankShift))
			// if mode is true, then writes affect 0x0000 - 0x7fff & 0xa000 - 0xbfff
			if c.mbc1Mode {
				bankNumber := (value << c.mbc1BankShift) % (uint8(len(c.ROM) / 0x4000))
				c.b.CopyTo(0x0000, 0x4000, c.ROM[int(bankNumber)*0x4000:])

				if c.ramEnabled {
					c.updateRAMBank(value)
				}
			} else {
				c.b.CopyTo(0x0000, 0x4000, c.ROM)

				if c.ramEnabled {
					c.updateRAMBank(0)
				}
			}
		case MBC3RAM, MBC3RAMBATT, MBC3TIMERBATT, MBC3TIMERRAMBATT:
			if value <= 3 && c.ramEnabled {
				c.updateRAMBank(value & 3)
			} else if value >= 0x08 && value <= 0x0c && c.rtcEnabled {
				c.rtcRegister = value
			}
		case MBC5RAM, MBC5RAMBATT:
			c.updateRAMBank(value & 0x0f)
		case MBC5RUMBLE, MBC5RUMBLERAM, MBC5RUMBLERAMBATT:
			// bit 3 controls rumble on cartridges that feature a rumble motor
			c.RumbleCallback(value&types.Bit3 > 0)

			c.updateRAMBank(value & 7)
		}
	case address < 0x8000:
		switch c.CartridgeType {
		case MBC1, MBC1RAM, MBC1RAMBATT:
			c.mbc1Mode = value&1 == 1

			if c.mbc1Mode {
				c.updateRAMBank(c.mbc1Bank2)
			} else {
				c.updateRAMBank(0)
			}
		case MBC3TIMERBATT, MBC3TIMERRAMBATT:
			if c.rtcLatchValue == 0 && value == 1 {
				c.updateRTC()

				c.rtcLatchedS = c.rtcS
				c.rtcLatchedM = c.rtcM
				c.rtcLatchedH = c.rtcH
				c.rtcLatchedDL = c.rtcDL
				c.rtcLatchedDH = c.rtcDH
			}

			c.rtcLatchValue = value
		}
	case address >= 0xA000 && address < 0xC000:
		switch c.CartridgeType {
		// MBC2 features a unique 512 x 4 bit RAM array :)
		case MBC2, MBC2BATT:
			if c.ramEnabled {
				c.RAM[address&0x01ff] = value | 0xf0

				// account for wrap-around (could mask it in read but then that's another conditional on the read path)
				for i := uint16(0); i < 16; i++ {
					c.b.data[0xa000+(i*0x200)+(address&0x01ff)] = value | 0xf0
				}
			}
		case MBC3TIMERBATT, MBC3TIMERRAMBATT:
			if c.rtcEnabled {
				c.rtcLastUpdate = c.b.s.Cycle()
				switch c.rtcRegister {
				case 0x08:
					c.rtcS = value & 0x3f
				case 0x09:
					c.rtcM = value & 0x3f
				case 0x0A:
					c.rtcH = value & 0x1f
				case 0x0B:
					c.rtcDL = value
				case 0x0C:
					c.rtcDH = value & 0xc1
				}

				return
			}

			fallthrough
		default:
			// if there is no RAM or RAM is disabled, do nothing
			if len(c.RAM) == 0 || !c.ramEnabled {
				return
			}

			// write the value to cart RAM at the current RAM offset
			c.RAM[c.ramOffset+uint32(address&0x1fff)] = value
			c.b.data[address] = value
		}
	}
}

func (c *Cartridge) readRTC() byte {
	switch c.rtcRegister {
	case 0x08:
		return c.rtcLatchedS
	case 0x09:
		return c.rtcLatchedM
	case 0x0A:
		return c.rtcLatchedH
	case 0x0B:
		return c.rtcLatchedDL
	case 0x0C:
		return c.rtcLatchedDH
	default:
		return 0xff
	}
}

// NewCartridge creates a new cartridge from the provided ROM.
func NewCartridge(rom []byte, b *Bus) *Cartridge {
	// create cartridge and parse header
	c := &Cartridge{
		ROM:           rom,
		romOffset:     0x4000,
		b:             b,
		mbc1Bank1:     1,
		mbc1BankShift: 5,
	}
	c.parseHeader()

	if c.CartridgeType >= MBC1 && c.CartridgeType <= MBC1RAMBATT {
		c.detectMultiCart()
	}

	// override RAM size for MBC2
	if c.CartridgeType == MBC2 || c.CartridgeType == MBC2BATT {
		c.RAMSize = 512 // 512 4-bit
	}

	// create RAM
	c.RAM = make([]byte, c.RAMSize)

	// copy initial ROM contents to bus
	c.b.CopyTo(0, 0x8000, c.ROM)
	fmt.Println(c)
	return c
}

var oldLicenseeCodeMap = map[uint8]string{
	0x00: "None",
	0x01: "Nintendo",
	0x08: "Capcom",
	0x09: "Hot B Co.",
	0x0A: "Jaleco",
	0x0B: "Coconuts",
	0x0C: "Elite Systems",
	0x13: "Electronic Arts",
	0x18: "Hudson Soft",
	0x19: "ITC Entertainment",
	0x1A: "Yanoman",
	0x1D: "Clary",
	0x1F: "Virgin",
	0x24: "PCM Complete",
	0x25: "San-X",
	0x28: "Kotobuki Systems",
	0x29: "Seta",
	0x30: "Infogrames",
	0x31: "Nintendo",
	0x32: "Bandai",
	// 0x33 is used for new licensee code
	0x34: "Konami",
	0x35: "Hector",
	0x38: "Capcom",
	0x39: "Banpresto",
	0x3C: "Entertainment i",
	0x3E: "Gremlin",
	0x41: "Ubisoft",
	0x42: "Atlus",
	0x44: "Malibu",
	0x46: "Angel",
	0x47: "Spectrum Holoby",
	0x49: "Irem",
	0x4A: "Virgin",
	0x4D: "Malibu",
	0x4F: "U.S. Gold",
	0x50: "Absolute",
	0x51: "Acclaim",
	0x52: "Activision",
	0x53: "American Sammy",
	0x54: "GameTek",
	0x55: "Park Place",
	0x56: "LJN",
	0x57: "Matchbox",
	0x59: "Milton Bradley",
	0x5A: "Mindscape",
	0x5B: "Romstar",
	0x5C: "Naxat Soft",
	0x5D: "Tradewest",
	0x60: "Titus",
	0x61: "Virgin",
	0x67: "Ocean",
	0x69: "Electronic Arts",
	0x6E: "Elite Systems",
	0x6F: "Electro Brain",
	0x70: "Infogrames",
	0x71: "Interplay",
	0x72: "Broderbund",
	0x73: "Sculptured",
	0x75: "The Sales Curve",
	0x78: "THQ",
	0x79: "Accolade",
	0x7A: "Triffix Entertainment",
	0x7C: "Microprose",
	0x7F: "Kemco",
	0x80: "Misawa",
	0x83: "LOZC",
	0x86: "Tokuma Shoten i",
	0x8B: "Bullet-Proof",
	0x8C: "Vic Tokai",
	0x8E: "Ape",
	0x8F: "I'Max",
	0x91: "Chun Soft",
	0x92: "Video System",
	0x93: "Tsuburaya",
	0x95: "Varie",
	0x96: "Yonezawa/S'pal",
	0x97: "Kaneko",
	0x99: "Arc",
	0x9A: "Nihon Bussan",
	0x9B: "Tecmo",
	0x9C: "Imagineer",
	0x9D: "Banpresto",
	0x9F: "Nova",
	0xA1: "Hori Electric",
	0xA2: "Bandai",
	0xA4: "Konami",
	0xA6: "Kawada",
	0xA7: "Takara",
	0xA9: "Technos Japan",
	0xAA: "Broderbund",
	0xAC: "Toei Animation",
	0xAD: "Toho",
	0xAF: "Namco",
	0xB0: "Acclaim",
	0xB1: "Ascii or Nexoft",
	0xB2: "Bandai",
	0xB4: "Square Enix",
	0xB6: "HAL Laboratory",
	0xB7: "SNK",
	0xB9: "Pony Canyon",
	0xBA: "Culture Brain",
	0xBB: "Sunsoft",
	0xBD: "Sony Imagesoft",
	0xBF: "Sammy",
	0xC0: "Taito",
	0xC2: "Kemco",
	0xC3: "Squaresoft",
	0xC4: "Tokuma Shoten i",
	0xC5: "Data East",
	0xC6: "Tonkin House",
	0xC8: "Koei",
	0xC9: "UFL",
	0xCA: "Ultra",
	0xCB: "Vap",
	0xCC: "Use",
	0xCD: "Meldac",
	0xCE: "Pony Canyon or",
	0xCF: "Angel",
	0xD0: "Taito",
	0xD1: "Sofel",
	0xD2: "Quest",
	0xD3: "Sigma Enterprises",
	0xD4: "Ask Kodansha",
	0xD6: "Naxat Soft",
	0xD7: "Copya System",
	0xD9: "Banpresto",
	0xDA: "Tomy",
	0xDB: "LJN",
	0xDD: "NCs",
	0xDE: "Human",
	0xDF: "Altron",
	0xE0: "Jaleco",
	0xE1: "Towachiki",
	0xE2: "Uutaka",
	0xE3: "Varie",
	0xE5: "Epoch",
	0xE7: "Athena",
	0xE8: "Asmik",
	0xE9: "Natsume",
	0xEA: "King Records",
	0xEB: "Atlus",
	0xEC: "Epic/Sony Records",
	0xEE: "IGS",
	0xF0: "A Wave",
	0xF3: "Extreme Entertainment",
	0xFF: "LJN",
}

var newLicenseeCodeMap = map[string]string{
	"28": "Kemco Japan",
	"00": "None",
	"01": "Nintendo",
	"08": "Capcom",
	"13": "Electronic Arts",
	"18": "Hudson Soft",
	"19": "b-ai",
	"20": "KSS",
	"22": "pow",
	"24": "PCM Complete",
	"25": "san-x",
	"29": "Seta",
	"30": "Viacom",
	"31": "Nintendo",
	"32": "Bandai",
	"33": "Ocean/Acclaim",
	"34": "Konami",
	"35": "Hector",
	"37": "Taito",
	"38": "Hudson",
	"39": "Banpresto",
	"41": "Ubi Soft",
	"42": "Atlus",
	"44": "Malibu",
	"46": "angel",
	"47": "Bullet-Proof",
	"49": "irem",
	"50": "Absolute",
	"51": "Acclaim",
	"52": "Activision",
	"53": "American sammy",
	"54": "Konami",
	"55": "Hi tech entertainment",
	"56": "LJN",
	"57": "Matchbox",
	"58": "Mattel",
	"59": "Milton Bradley",
	"60": "Titus",
	"61": "Virgin",
	"67": "Ocean/Acclaim",
	"69": "Electronic Arts",
	"70": "Infogrames",
	"71": "Interplay",
	"72": "Broderbund",
	"73": "Sculptured",
	"75": "sci",
	"78": "THQ",
	"79": "Accolade",
	"80": "misawa",
	"83": "lozc",
	"86": "tokuma shoten i",
	"87": "tsukuda ori",
	"91": "Chun Soft",
	"92": "Video System",
	"93": "Ocean/Acclaim",
	"95": "Varie",
	"96": "Yonezawa/s'pal",
	"97": "Kaneo",
	"99": "Pack in soft",
	"A4": "Konami (Yu-Gi-Oh!)",
}

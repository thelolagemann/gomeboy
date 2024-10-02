//go:generate stringer -type=CartridgeType,CGBFlag -output=cartridge_string.go
package io

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/vladimirvivien/go4vl/device"
	"hash/crc32"
	"image"
	"slices"
	"strconv"
	"strings"
)

type CGBFlag uint8 // CGBFlag specifies the level of CGB support in a Cartridge.

const (
	CGBUnset    CGBFlag = iota // No CGB support has been specified, >99.9% a regular Game Boy game.
	CGBEnhanced                // The game supports CGB enhancements, but is backwards compatible.
	CGBOnly                    // The game works on CGB only.
)

type CartridgeType uint16 // CartridgeType represents the hardware present in a Cartridge.

const (
	ROM               CartridgeType = 0x00
	MBC1              CartridgeType = 0x01
	MBC1RAM           CartridgeType = 0x02
	MBC1RAMBATT       CartridgeType = 0x03
	MBC2              CartridgeType = 0x05
	MBC2BATT          CartridgeType = 0x06
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
	MBC6              CartridgeType = 0x20
	MBC7              CartridgeType = 0x22
	POCKETCAMERA      CartridgeType = 0xFC
	BANDAITAMA5       CartridgeType = 0xFD
	HUDSONHUC3        CartridgeType = 0xFE
	HUDSONHUC1        CartridgeType = 0xFF
	MBC1M             CartridgeType = 0x0100
	M161              CartridgeType = 0x0101
)

var batteryMappers = []CartridgeType{
	MBC1RAMBATT, MMM01RAMBATT, MBC2BATT, MBC3RAMBATT, MBC3TIMERRAMBATT, MBC3TIMERBATT, MBC5RAMBATT, MBC5RUMBLERAMBATT, MBC7,
} // mappers that feature battery backed RAM

type rtcRegister = int // represents one of the 5 rtc registers (latched is indexed +5)

const (
	rtcS  rtcRegister = iota // seconds
	rtcM                     // minutes
	rtcH                     // hours
	rtcDL                    // days lower
	rtcDH                    // days higher & control
)

type Cartridge struct {
	ROM []byte
	RAM []byte

	Title            string  // $0134-$0143 Title of the game in uppercase ASCII.
	ManufacturerCode string  // $013F-$0142 4-character ManufacturerCode (in uppercase ASCII) - purpose remains unknown
	CGBFlag                  // $0142 - Indicates level of CGB support
	NewLicenseeCode  [2]byte // $0144-$0145 2-character ASCII "licensee code"
	SGBFlag          bool    // $0146 - Specifies whether the game supports SGB functions
	CartridgeType            // $0147 - Specifies the hardware present on a Cartridge.
	ROMSize          int     // $0148 - Specifies how much ROM is on the Cartridge, calculated by 32 KiB x (1<<value)
	RAMSize          int     // $0149 - Specifies how much RAM is present on the Cartridge, if any.
	DestinationCode  byte    // $014A - Specifies whether the game is intended to be sold in Japan or elsewhere
	OldLicenseeCode  byte    // $014B - Specifies the game's publisher; see NewLicenseeCode if val == $33
	MaskROMVersion   uint8   // $014C - Specifies the version of the game, usually $00
	HeaderChecksum   uint8   // $014D - 8-Bit checksum of header bytes $0134-$014C
	GlobalChecksum   uint16  // $014E-$014F 16-bit (big endian) checksum of Cartridge ROM (excluding these bytes)

	Features struct {
		Accelerometer bool
		Battery       bool
		RAM           bool
		RTC           bool
		Rumble        bool
	}

	RumbleCallback func(bool)

	// internal fields
	ramEnabled bool
	romOffset  uint32
	ramOffset  uint32

	// mapper specific fields
	mbc1 struct {
		bankShift uint8
		bank1     uint8
		bank2     uint8
		mode      bool
	}

	m161Latched bool

	mbc7 struct {
		latchReady     bool
		ramEnabled     bool
		xLatch, yLatch uint16

		eeprom struct {
			do, di, clk, cs bool // (d)ata (o)ut, (d)ata (i)n, (c)(l)oc(k), (c)hip (s)elect
			writeEnabled    bool
			command         uint16 // 11 bits
			bitsIn, bitsOut uint16
			bitsLeft        uint8 // 5 bits
		}
	}

	camera struct {
		webcamImage     [CAMERA_SENSOR_W][CAMERA_SENSOR_H]int
		image           image.Image
		sensorImage     [CAMERA_SENSOR_W][CAMERA_SENSOR_H]int
		registers       [0x36]uint8
		registersMapped bool
	}

	rtc struct {
		enabled, latched, latching bool
		register                   uint8 // currently banked rtc register
		lastUpdate, heldTicks      uint64
	}

	AccelerometerX, AccelerometerY float32 // TODO refactor this to somewhere more appropriate

	b *Bus
}

// parseHeader parses the Cartridge header from Cartridge.ROM.
func (c *Cartridge) parseHeader() {
	c.CGBFlag = []CGBFlag{0x80: CGBEnhanced, 0xC0: CGBOnly, 0xFF: CGBUnset}[c.ROM[0x143]]

	// CGB cartridge header reduced the title length to 15, and then some months later to 11
	if c.CGBFlag == CGBUnset {
		c.Title = string(c.ROM[0x0134:0x0144])
	} else {
		c.Title = string(c.ROM[0x0134:0x0143]) // TODO determine which games used manufacturer code?
	}
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
	c.MaskROMVersion = c.ROM[0x014C]
	c.HeaderChecksum = c.ROM[0x014D]
	c.GlobalChecksum = binary.BigEndian.Uint16(c.ROM[0x014E:0x0150])

	// handle unique mappers that report as others :)
	switch {
	case crc32.ChecksumIEEE(c.ROM) == 0x0C38A775: // Mani 4 in 1 (China) (DMG-601 CHN)
		c.CartridgeType = M161
	case c.CartridgeType == MBC1 && c.ROMSize == (1024*1024) && bytes.Equal(c.ROM[0x0104:0x0134], c.ROM[0x4104:0x4134]):
		c.mbc1.bankShift = 4
		c.CartridgeType = MBC1M
	}
}

// updateROMBank sets the current ROM bank and copies it to the Bus [$4000:$8000]
func (c *Cartridge) updateROMBank(bank uint16) {
	c.romOffset = (uint32(bank) * 0x4000) % uint32(len(c.ROM))
	c.b.CopyTo(0x4000, 0x8000, c.ROM[c.romOffset:])
}

// updateRAMBank sets the appropriate ram offset to be used depending on the provided
// bank value and the length of Cartridge.RAM, and then copies it to the Bus [$A000:$C000]
func (c *Cartridge) updateRAMBank(bank uint8) {
	// do nothing if the cartridge has no ram or ram banks
	if c.RAMSize <= 8192 {
		return
	}

	c.b.CopyFrom(0xA000, 0xC000, c.RAM[c.ramOffset:])         // copy current RAM bank from bus -> cart
	c.ramOffset = (uint32(bank) * 0x2000) % uint32(c.RAMSize) // set new RAM offset
	c.b.CopyTo(0xA000, 0xC000, c.RAM[c.ramOffset:])           // copy new RAM bank from cart -> bus
}

// updateRTC sets the RTC registers based on how many cycles have passed since the last read.
// TODO make configurable to sync to host
func (c *Cartridge) updateRTC() {
	// is the RTC ticking?
	if c.RAM[c.RAMSize+rtcDH]&types.Bit6 > 0 {
		return // no
	}

	// get delta and determine how many seconds have passed
	delta := c.b.s.Cycle() - c.rtc.lastUpdate
	ticks := int(delta / 4194304)
	rB := c.RAM[c.RAMSize : c.RAMSize+rtcDH+1]

	for i := 0; i < ticks; i++ {
		rB[rtcS]++

		if rB[rtcS] == 60 {
			rB[rtcS] = 0
			rB[rtcM]++

			if rB[rtcM] == 60 {
				rB[rtcM] = 0
				rB[rtcH]++

				if rB[rtcH] == 24 {
					rB[rtcH] = 0

					if rB[rtcDL] == 255 {
						switch rB[rtcDH] & types.Bit0 {
						case 0: // 255 day rollover
							rB[rtcDH] |= types.Bit0
						case 1: // 512 day overflow
							rB[rtcDH] &^= types.Bit0
							rB[rtcDH] |= types.Bit7
						}
						rB[rtcDL] = 0
					} else {
						rB[rtcDL]++
					}
				}

				if rB[rtcH] > 31 {
					rB[rtcH] = 0
				}
			}

			if rB[rtcM] > 63 {
				rB[rtcM] = 0
			}
		}

		if rB[rtcS] > 63 {
			rB[rtcS] = 0 // invalid rollovers don't increment the next register
		}

	}

	if ticks > 0 {
		// copy modified registers back
		copy(c.RAM[c.RAMSize:], rB)
		c.rtc.lastUpdate = c.b.s.Cycle()
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
	return c.CGBFlag > CGBUnset
}

// Licensee returns the Licensee of the cartridge, according to the parsed header data.
func (c *Cartridge) Licensee() string {
	if c.OldLicenseeCode == 0x33 {
		if l, ok := newLicenseeCodeMap[string(c.NewLicenseeCode[:])]; ok {
			return l
		} else {
			n, _ := strconv.ParseInt(string(c.NewLicenseeCode[:]), 16, 8)
			return oldLicenseeCodeMap[uint8(n)]
		}
	}

	return oldLicenseeCodeMap[c.OldLicenseeCode]
}

// SGB returns true if the Cartridge supports SGB functions.
func (c *Cartridge) SGB() bool {
	return c.SGBFlag && c.OldLicenseeCode == 0x33
}

// String implements the fmt.Stringer interface.
func (c *Cartridge) String() string {
	return fmt.Sprintf("%s (%s) | (%dKiB|%dKiB) %s %s", c.Title, c.Licensee(), c.ROMSize/1024, c.RAMSize/1024, c.CartridgeType, c.CGBFlag)
}

// Write writes an 8-bit value to the cartridge.
func (c *Cartridge) Write(address uint16, value uint8) {
	switch c.CartridgeType {
	case ROM:
		return // ROM is (R)ead-(O)nly (M)emory
	case M161:
		if c.m161Latched {
			return // m161 only supports 1 bank switch per session
		}
		if address < 0x8000 {
			c.b.CopyTo(0x0000, 0x8000, c.ROM[int(value&7)*0x8000:])
			c.m161Latched = true
		}
	case MBC2, MBC2BATT:
		switch {
		case address < 0x4000:
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
		case address >= 0xA000 && address < 0xC000:
			if c.ramEnabled {
				c.RAM[address&0x01ff] = value | 0xf0

				// account for wrap-around (could mask it in read but then that's another conditional on the read path)
				for i := uint16(0); i < 16; i++ {
					c.b.data[0xa000+(i*0x200)+(address&0x01ff)] = value | 0xf0
				}
			}
		}
	case MBC7:
		switch {
		case address < 0x2000: // RAM enable 1
			c.ramEnabled = value == 0x0a
		case address < 0x4000: // ROM bank number
			c.updateROMBank(uint16(value))
		case address < 0x6000: // RAM enable 2
			c.mbc7.ramEnabled = value == 0x40
		case address >= 0xa000 && address < 0xB000:
			if !c.ramEnabled || !c.mbc7.ramEnabled { // MBC7 uses two RAM gates to enable access
				return
			}
			switch address >> 4 & 0xf {
			case 0:
				if value == 0x55 { // reset latched
					c.mbc7.latchReady = true
					c.mbc7.xLatch, c.mbc7.yLatch = 0x8000, 0x8000
				}
			case 1:
				if value == 0xAA { // latch values
					c.mbc7.latchReady = false

					// accelerometer values are centered around 0x81D0
					c.mbc7.xLatch = 0x81D0 + uint16(0x70*c.AccelerometerX)
					c.mbc7.yLatch = 0x81D0 + uint16(0x70*c.AccelerometerY)
				}
			case 8:
				c.mbc7.eeprom.cs = value&types.Bit7 > 0
				c.mbc7.eeprom.di = value&types.Bit1 > 0

				// is (c)hip (s)elect pulled high?
				if c.mbc7.eeprom.cs {
					// has the chip been (c)(l)oc(k)ed?
					if !c.mbc7.eeprom.clk && (value&types.Bit6 > 0) {
						// shift (d)ata (o)ut msb
						c.mbc7.eeprom.do = c.mbc7.eeprom.bitsOut>>15&1 > 0
						c.mbc7.eeprom.bitsOut <<= 1
						c.mbc7.eeprom.bitsOut |= 1

						// are there extra bits to shift in? (WRITE/WRAL need an extra 16 bits)
						if c.mbc7.eeprom.bitsLeft == 0 {
							// shift (d)ata (i)n into command msb
							c.mbc7.eeprom.command <<= 1
							if c.mbc7.eeprom.di {
								c.mbc7.eeprom.command |= 1
							}

							// commands are 10 bits long & preceded by a 1 bit, so when bit 10 is set we have a command
							if c.mbc7.eeprom.command&0x400 > 0 {
								idx := c.mbc7.eeprom.command & 0x7f * 2 // address used by commands

								switch c.mbc7.eeprom.command >> 6 & 0xf {
								case 0x8, 0x9, 0xA, 0xB: // READ 10_xAAA_AAAA
									c.mbc7.eeprom.bitsOut = uint16(c.RAM[idx]) | uint16(c.RAM[idx+1])<<8
									c.mbc7.eeprom.command = 0
								case 0x3: // EWEN 00_11xx_xxxx
									c.mbc7.eeprom.writeEnabled = true
									c.mbc7.eeprom.command = 0
								case 0x0: // EWDS 00_00xx_xxxx
									c.mbc7.eeprom.writeEnabled = false
									c.mbc7.eeprom.command = 0
								case 0x4, 0x5, 0x6, 0x7: // WRITE 01_xAAA_AAAA
									if c.mbc7.eeprom.writeEnabled {
										c.RAM[idx] = 0
										c.RAM[idx+1] = 0
									}
									c.mbc7.eeprom.bitsLeft = 16
								case 0xC, 0xD, 0xE, 0xF: // ERASE 11_xAAA_AAAA
									if c.mbc7.eeprom.writeEnabled {
										c.RAM[idx] = 0xff
										c.RAM[idx+1] = 0xff
										c.mbc7.eeprom.bitsOut = 0x3fff
									}
									c.mbc7.eeprom.command = 0
								case 0x2: // ERAL 00_10xx_xxxx
									if c.mbc7.eeprom.writeEnabled {
										for i := range c.RAM {
											c.RAM[i] = 0xff
										}
										c.mbc7.eeprom.bitsOut = 0xff
									}
									c.mbc7.eeprom.command = 0
								case 0x1: // WRAL 00_01xx_xxxx
									if c.mbc7.eeprom.writeEnabled {
										for i := range c.RAM {
											c.RAM[i] = 0
										}
									}
									c.mbc7.eeprom.bitsLeft = 16
								}
							}
						} else {
							// we still need to shift in another 16 bits
							c.mbc7.eeprom.bitsLeft--
							c.mbc7.eeprom.do = true

							// has (d)ata (i)n been set high?
							if c.mbc7.eeprom.di {
								c.mbc7.eeprom.bitsIn |= 1
							}

							// have we transferred 16 bits yet?
							if c.mbc7.eeprom.bitsLeft == 0 {
								idx := c.mbc7.eeprom.command & 0x7f * 2 // get address from command
								if c.mbc7.eeprom.command&0x100 > 0 {    // WRITE
									c.RAM[idx] = uint8(c.mbc7.eeprom.bitsIn)
									c.RAM[idx+1] = uint8(c.mbc7.eeprom.bitsIn >> 8)
									c.mbc7.eeprom.bitsOut = 0xff
								} else { // WRAL
									for i := 0; i < 0x7f; i++ {
										c.RAM[i] = uint8(c.mbc7.eeprom.bitsIn)
										c.RAM[i+1] = uint8(c.mbc7.eeprom.bitsIn >> 8)
									}
									c.mbc7.eeprom.bitsOut = 0x3fff
								}

								// reset incoming bits & command
								c.mbc7.eeprom.bitsIn = 0
								c.mbc7.eeprom.command = 0
							} else {
								// shift data in
								c.mbc7.eeprom.bitsIn <<= 1
							}
						}
					}
				}

				c.mbc7.eeprom.clk = value&types.Bit6 > 0
			}
		}
	default:
		switch {
		case address < 0x2000:
			switch c.CartridgeType {
			case MBC1RAM, MBC1RAMBATT, MBC3RAM, MBC3RAMBATT, MBC3TIMERBATT, MBC3TIMERRAMBATT, MBC5RAM, MBC5RAMBATT, MBC5RUMBLERAM, MBC5RUMBLERAMBATT, POCKETCAMERA:
				c.ramEnabled = value&0x0f == 0x0a && c.CartridgeType != MBC3TIMERBATT
				c.rtc.enabled = value&0x0f == 0x0a && c.Features.RTC
				return
			}
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
			case MBC1, MBC1RAM, MBC1RAMBATT, MBC1M:
				value &= 0x1f // 5-bit value

				// can't write a value of 0
				if value == 0 {
					value = 1
				}

				// and only a 4-bit value on multicarts (but 5-bit for 0->1)
				if c.CartridgeType == MBC1M {
					value &= 0x0f
				}

				c.mbc1.bank1 = value

				c.updateROMBank(uint16(c.mbc1.bank2<<c.mbc1.bankShift | value))
			case MBC3, MBC3RAM, MBC3RAMBATT, MBC3TIMERBATT, MBC3TIMERRAMBATT:
				if value == 0 {
					value = 1
				}
				c.updateROMBank(uint16(value))
			case MBC5, MBC5RAM, MBC5RAMBATT, MBC5RUMBLE, MBC5RUMBLERAM, MBC5RUMBLERAMBATT:
				romBank := uint16(c.romOffset / 0x4000)
				romBank = romBank&0x00ff + uint16(value&1)<<8
				c.updateROMBank(romBank)
			case POCKETCAMERA:
				c.updateROMBank(uint16(value))
			}
		case address < 0x6000:
			switch c.CartridgeType {
			case MBC1, MBC1M, MBC1RAM, MBC1RAMBATT:
				// bank2 is a 2-bit value
				value &= 3

				c.mbc1.bank2 = value

				c.updateROMBank(uint16(c.mbc1.bank1 | c.mbc1.bank2<<c.mbc1.bankShift))
				// if mode is true, then writes affect 0x0000 - 0x7fff & 0xa000 - 0xbfff
				if c.mbc1.mode {
					bankNumber := (value << c.mbc1.bankShift) % (uint8(len(c.ROM) / 0x4000))
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
				if value <= 3 {
					c.updateRAMBank(value & 3)
					c.rtc.register = 0
				} else if value >= 0x08 && value <= 0x0c {
					c.rtc.register = value
				}
			case MBC5RAM, MBC5RAMBATT:
				c.updateRAMBank(value & 0x0f)
			case MBC5RUMBLE, MBC5RUMBLERAM, MBC5RUMBLERAMBATT:
				// bit 3 controls rumble on cartridges that feature a rumble motor
				c.RumbleCallback(value&types.Bit3 > 0)

				c.updateRAMBank(value & 7)
			case POCKETCAMERA:
				c.camera.registersMapped = value&types.Bit4 > 0
				c.updateRAMBank(value & 0x0f)
			}
		case address < 0x8000:
			switch c.CartridgeType {
			case MBC1, MBC1M, MBC1RAM, MBC1RAMBATT:
				c.mbc1.mode = value&1 == 1

				if c.mbc1.mode {
					c.updateRAMBank(c.mbc1.bank2)
				} else {
					c.updateRAMBank(0)
				}
			case MBC3TIMERBATT, MBC3TIMERRAMBATT:
				if c.rtc.latching && value == 1 {
					c.updateRTC()
					copy(c.RAM[c.RAMSize+5:c.RAMSize+10], c.RAM[c.RAMSize:c.RAMSize+5])
				}

				c.rtc.latching = value == 0
			}
		case address >= 0xA000 && address < 0xC000:
			switch c.CartridgeType {
			case MBC3TIMERBATT, MBC3TIMERRAMBATT:
				if c.rtc.enabled && c.rtc.register != 0 {
					switch c.rtc.register {
					case 0x08:
						c.rtc.lastUpdate = c.b.s.Cycle() // cheeky hack not accurate at all
						c.RAM[c.RAMSize+rtcS] = value & 0x3f
					case 0x09:
						c.RAM[c.RAMSize+rtcM] = value & 0x3f
					case 0x0A:
						c.RAM[c.RAMSize+rtcH] = value & 0x1f
					case 0x0B:
						c.RAM[c.RAMSize+rtcDL] = value
					case 0x0C:
						if c.RAM[c.RAMSize+rtcDH]&types.Bit6 == 0 && value&types.Bit6 > 0 { // store ticks
							c.rtc.heldTicks = c.b.s.Cycle() - c.rtc.lastUpdate
						} else if c.RAM[c.RAMSize+rtcDH]&types.Bit6 > 0 && value&types.Bit6 == 0 { // restore ticks
							c.rtc.lastUpdate = c.b.s.Cycle() - c.rtc.heldTicks
							c.rtc.heldTicks = 0
						}

						c.RAM[c.RAMSize+rtcDH] = value & 0xc1
					}

					return
				}

				fallthrough
			case POCKETCAMERA:
				c.writeCameraRAM(address, value)
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

}

func (c *Cartridge) readMBC7RAM(addr uint16) uint8 {
	if !c.ramEnabled || !c.mbc7.ramEnabled || addr >= 0xb000 {
		return 0xff
	}

	switch addr >> 4 & 0xf {
	case 2:
		return uint8(c.mbc7.xLatch)
	case 3:
		return uint8(c.mbc7.xLatch >> 8)
	case 4:
		return uint8(c.mbc7.yLatch)
	case 5:
		return uint8(c.mbc7.yLatch >> 8)
	case 6:
		return 0
	case 8:
		var x uint8
		if c.mbc7.eeprom.do {
			x |= types.Bit0
		}
		if c.mbc7.eeprom.di {
			x |= types.Bit1
		}
		if c.mbc7.eeprom.clk {
			x |= types.Bit6
		}
		if c.mbc7.eeprom.cs {
			x |= types.Bit7
		}
		return x
	}

	return 0xff
}

// NewCartridge creates a new cartridge from the provided ROM.
func NewCartridge(rom []byte, b *Bus) *Cartridge {
	c := &Cartridge{
		ROM:       rom,
		romOffset: 0x4000,
		b:         b,
	}
	c.mbc1.bank1 = 1
	c.mbc1.bankShift = 5
	c.parseHeader()
	devices, err := device.GetAllDevicePaths()
	if err != nil {
		panic(err)
	}
	for _, d := range devices {
		cam, err := device.Open(d, device.WithBufferSize(1))
		if err != nil {
			panic(err)
		}
		if err := cam.Start(context.TODO()); err != nil {
			panic(err)
		}
		dev = cam
		break

	}

	var ramSize = c.RAMSize
	// override RAM sizes for oddball mbcs
	switch c.CartridgeType {
	case MBC2, MBC2BATT:
		c.RAMSize = 512 // has a 512 4-bit RAM
		ramSize = 512
	case MBC3TIMERBATT, MBC3TIMERRAMBATT:
		ramSize = c.RAMSize + 10
	case MBC7:
		c.RAMSize = 256 // 256 byte EEPROM
		ramSize = 256
	}

	// create RAM
	c.RAM = make([]byte, ramSize)

	// determine cartridge features
	c.Features.Accelerometer = c.CartridgeType == MBC7
	c.Features.Battery = slices.Contains(batteryMappers, c.CartridgeType)
	c.Features.RAM = len(c.RAM) > 0
	c.Features.RTC = c.CartridgeType == MBC3TIMERBATT || c.CartridgeType == MBC3TIMERRAMBATT
	c.Features.Rumble = c.CartridgeType == MBC5RUMBLE || c.CartridgeType == MBC5RUMBLERAM || c.CartridgeType == MBC5RUMBLERAMBATT

	c.b.CopyTo(0, 0x8000, c.ROM)

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
	"19": "b-ai",
	"20": "KSS",
	"22": "pow",
	"30": "Viacom",
	"33": "Ocean/Acclaim",
	"37": "Taito",
	"38": "Hudson",
	"47": "Bullet-Proof",
	"54": "Konami",
	"55": "Hi tech entertainment",
	"58": "Mattel",
	"59": "Milton Bradley",
	"67": "Ocean/Acclaim",
	"75": "sci",
	"87": "tsukuda ori",
	"93": "Ocean/Acclaim",
	"99": "Pack in soft",
	"A4": "Konami (Yu-Gi-Oh!)",
}

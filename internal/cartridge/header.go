package cartridge

import "fmt"

type Flag uint8

const (
	FlagOnlyDMG Flag = iota
	FlagSupportsCGB
	FlagOnlyCGB
)

var (
	ramMAP = map[uint8]uint{
		0x00: 0,
		0x02: 8 * 1024,
		0x03: 32 * 1024,
		0x04: 128 * 1024,
		0x05: 64 * 1024,
	}
)

type Type uint8

const (
	ROM               Type = 0x00
	MBC1              Type = 0x01
	MBC1RAM           Type = 0x02
	MBC1RAMBATT       Type = 0x03
	MBC2              Type = 0x05
	MBC2BATT          Type = 0x06
	ROMRAM            Type = 0x08
	ROMRAMBATT        Type = 0x09
	MMM01             Type = 0x0B
	MMM01RAM          Type = 0x0C
	MMM01RAMBATT      Type = 0x0D
	MBC3TIMERBATT     Type = 0x0F
	MBC3TIMERRAMBATT  Type = 0x10
	MBC3              Type = 0x11
	MBC3RAM           Type = 0x12
	MBC3RAMBATT       Type = 0x13
	MBC5              Type = 0x19
	MBC5RAM           Type = 0x1A
	MBC5RAMBATT       Type = 0x1B
	MBC5RUMBLE        Type = 0x1C
	MBC5RUMBLERAM     Type = 0x1D
	MBC5RUMBLERAMBATT Type = 0x1E
	POCKETCAMERA      Type = 0x1F
	BANDAITAMA5       Type = 0xFD
	HUDSONHUC3        Type = 0xFE
	HUDSONHUC1        Type = 0xFF
)

// Header represents the header of a cartridge, each cartridge has a header and is
// located at the address space 0x0100-0x014F. The header contains information about
// the cartridge itself, and the hardware it expects to run on.
type Header struct {
	// 0x0134-0x0143 - Title of the game
	Title string

	// 0x013F-0x0142 - ManufacturerCode of the game
	ManufacturerCode string

	// 0x0143 - CartridgeGBMode of the game. In older cartridges this byte was part
	// of the title, but the Colour Game Boy and later models interpret this byte
	// to determine if the cartridge is compatible with the Colour Game Boy.
	CartridgeGBMode Flag

	// 0x0144-0x0145 - NewLicenseeCode of the game. This is used to identify the
	// licensee of the game. This is used in newer cartridges, and is used in
	// conjunction with the CartridgeType to determine the type of cartridge.
	NewLicenseeCode string
	SGBFlag         bool
	CartridgeType   Type
	ROMSize         uint
	RAMSize         uint
	CountryCode     uint8
	OldLicenseeCode uint8
	MaskROMVersion  uint8
	HeaderChecksum  uint8
	GlobalChecksum  uint16

	raw [0x50]byte
}

// parseHeader parses the header of the given ROM and returns a Header.
func parseHeader(header []byte) Header {
	h := Header{}

	// check if the header is valid
	if len(header) != 0x50 {
		panic(fmt.Sprintf("invalid header length: %d", len(header)))
	}

	// parse the mode of the cartridge and parse the header accordingly
	switch header[0x43] {
	case 0x80:
		h.CartridgeGBMode = FlagSupportsCGB
	case 0xC0:
		h.CartridgeGBMode = FlagOnlyCGB
	default:
		h.CartridgeGBMode = FlagOnlyDMG
	}

	// parse the title
	if h.CartridgeGBMode == FlagOnlyDMG {
		h.Title = string(header[0x34:0x44])
	} else {
		h.Title = string(header[0x34:0x43])
	}

	// parse the manufacturer code
	h.ManufacturerCode = string(header[0x3F:0x43])

	// parse the new licensee code
	h.NewLicenseeCode = string(header[0x44:0x46])

	// parse the SGB flag
	h.SGBFlag = header[0x46] == 0x03

	// parse the cartridge type
	h.CartridgeType = Type(header[0x47])

	// parse the ROM size (calculated by 32kB x (1 << n))
	h.ROMSize = (32 * 1024) * (1 << header[0x48])

	// parse the RAM size
	h.RAMSize = ramMAP[header[0x49]]

	// parse the country code
	h.CountryCode = header[0x4A]

	// parse the old licensee code
	h.OldLicenseeCode = header[0x4B]

	// parse the mask ROM version
	h.MaskROMVersion = header[0x4C]

	// parse the header checksum
	h.HeaderChecksum = header[0x4D]

	// parse the global checksum
	h.GlobalChecksum = uint16(header[0x4E]) | uint16(header[0x4F])<<8

	return h
}

func (h *Header) GameboyColor() bool {
	return h.CartridgeGBMode == FlagOnlyCGB || h.CartridgeGBMode == FlagSupportsCGB
}

func (h *Header) Hardware() string {
	switch h.CartridgeGBMode {
	case FlagOnlyDMG:
		return "DMG"
	case FlagSupportsCGB:
		return "CGB"
	case FlagOnlyCGB:
		return "CGB"
	default:
		return "Unknown"
	}
}

func (h *Header) String() string {
	return fmt.Sprintf("%s Mode: %s | ROM Size: %dkB | RAM Size: %dkB", h.Title, h.Hardware(), h.ROMSize, h.RAMSize)
}

package cartridge

import (
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/ppu/palette"
	"strings"
)

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
	POCKETCAMERA      Type = 0xFC
	BANDAITAMA5       Type = 0xFD
	HUDSONHUC3        Type = 0xFE
	HUDSONHUC1        Type = 0xFF
)

func (t Type) String() string {
	if name, ok := nameMap[t]; ok {
		return name
	}
	return fmt.Sprintf("Unknown cartridge type: 0x%02X", uint8(t))
}

var nameMap = map[Type]string{
	ROM:               "ROM",
	MBC1:              "MBC1",
	MBC1RAM:           "MBC1+RAM",
	MBC1RAMBATT:       "MBC1+RAM+BATT",
	MBC2:              "MBC2",
	MBC2BATT:          "MBC2+BATT",
	ROMRAM:            "ROM+RAM",
	ROMRAMBATT:        "ROM+RAM+BATT",
	MMM01:             "MMM01",
	MMM01RAM:          "MMM01+RAM",
	MMM01RAMBATT:      "MMM01+RAM+BATT",
	MBC3TIMERBATT:     "MBC3+TIMER+BATT",
	MBC3TIMERRAMBATT:  "MBC3+TIMER+RAM+BATT",
	MBC3:              "MBC3",
	MBC3RAM:           "MBC3+RAM",
	MBC3RAMBATT:       "MBC3+RAM+BATT",
	MBC5:              "MBC5",
	MBC5RAM:           "MBC5+RAM",
	MBC5RAMBATT:       "MBC5+RAM+BATT",
	MBC5RUMBLE:        "MBC5+RUMBLE",
	MBC5RUMBLERAM:     "MBC5+RUMBLE+RAM",
	MBC5RUMBLERAMBATT: "MBC5+RUMBLE+RAM+BATT",
	POCKETCAMERA:      "POCKET CAMERA",
	BANDAITAMA5:       "BANDAI TAMA5",
	HUDSONHUC3:        "HUDSON HUC3",
	HUDSONHUC1:        "HUDSON HUC1",
}

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
	NewLicenseeCode [2]byte
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

	ColourisationPalette palette.CompatibilityPaletteEntry

	b *io.Bus
}

func (h *Header) Destination() string {
	if h.CountryCode == 0x00 {
		return "Japanese"
	} else if h.CountryCode == 0x01 {
		return "Non-Japanese"
	}
	return "Unknown"
}

func (h *Header) TitleChecksum() uint8 {
	var checksum uint8
	for _, c := range h.raw[0x34:0x43] {
		checksum += c
	}
	return checksum
}

// parseHeader parses the header of the given ROM and returns a Header.
func parseHeader(header []byte) *Header {
	h := &Header{
		raw: [0x50]byte{},
	}
	// copy header into header.raw
	copy(h.raw[:], header)

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

		h.ColourisationPalette = palette.LoadColourisationPalette(header[0x34:0x44])
	}

	// parse the title
	if h.CartridgeGBMode == FlagOnlyDMG {
		h.Title = string(header[0x34:0x44])
	} else {
		h.Title = string(header[0x34:0x43])
	}

	// strip any trailing null bytes
	h.Title = strings.Replace(h.Title, "\x00", "", -1)

	// parse the manufacturer code
	h.ManufacturerCode = string(header[0x3F:0x43]) // TODO map

	// parse the new licensee code
	h.NewLicenseeCode = [2]byte{header[0x44], header[0x45]}

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

	fmt.Println(h.CartridgeType.String())
	return h
}

func (h *Header) GameboyColor() bool {
	return h.CartridgeGBMode == FlagOnlyCGB || h.CartridgeGBMode == FlagSupportsCGB
}

func (h *Header) Hardware() string {
	switch h.CartridgeGBMode {
	case FlagOnlyDMG:
		return "DMG"
	case FlagOnlyCGB, FlagSupportsCGB:
		return "CGB"
	default:
		return "Unknown"
	}
}

func (h *Header) Licensee() string {
	if h.OldLicenseeCode == 0x33 {
		// infer 2 byte slice as ASCII string and return mapped value
		return newLicenseeCodeMap[string(h.NewLicenseeCode[:])]
	}

	return oldLicenseeCodeMap[h.OldLicenseeCode]
}

func (h *Header) String() string {
	return fmt.Sprintf("%s (%s) | mode: %s | ROM Size: %dkB | RAM Size: %dkB | Cart Type: %d | Mode: %s", h.Title, h.Licensee(), h.Hardware(), h.ROMSize/1024, h.RAMSize/1024, h.CartridgeType, h.Hardware())
}

// oldLicenseeCodeMap maps the old licensee code to the licensee name,
// this is used for cartridges prior to the release of the SGB. A value
// of 0x33 in the OldLicenseeCode field indicates that the NewLicenseeCode
// field should be used instead.
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

// newLicenseeCodeMap maps the new licensee code to the licensee name,
// used for cartridges released after the SGB, and have a value of 0x33
// in the field of their OldLicenseeCode.
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

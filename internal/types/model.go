package types

// Model represents a model of the Game Boy. This is
// used to determine how the emulator should behave,
// in regard to the model-specific quirks. As a most
// basic example, any CGB models will have CGB features
// enabled, while DMG models will not.
type Model int

const (
	// Unset is the default model. It is used when the
	// model has not been write.
	Unset Model = iota - 1
	// DMG0 is an early DMG model, only released in Japan.
	DMG0 Model = iota
	// DMGABC is the standard DMG model.
	DMGABC
	// CGB0 is an early CGB model, only released in Japan.
	CGB0
	// CGBABC is the standard CGB model.
	CGBABC
	// MGB is the MGB model.
	MGB
	// SGB is the standard SGB model.
	SGB
	// SGB2 is the second SGB model.
	SGB2
	// AGB is the AGB model.
	AGB
)

// String returns the string representation of the model.
func (m Model) String() string {
	switch m {
	case DMG0:
		return "DMG0"
	case DMGABC:
		return "DMGABC"
	case CGB0:
		return "CGB0"
	case CGBABC:
		return "CGBABC"
	case MGB:
		return "MGB"
	case SGB:
		return "SGB"
	case SGB2:
		return "SGB2"
	case AGB:
		return "AGB"
	}
	return "Unknown"
}

// Registers returns the starting CPU IO when PC is
// write to 0x100. This is used to reset the CPU IO
// to their default values. Registers are returned in
// the order: A, F, B, C, D, E, H, L.
func (m Model) Registers() []uint8 {
	switch m {
	case DMG0:
		return []uint8{
			0x01, 0x00, 0xFF, 0x13, 0x00, 0xC1, 0x84, 0x03,
		}
	case DMGABC:
		return []uint8{
			0x01, 0xB0, 0x00, 0x13, 0x00, 0xD8, 0x01, 0x4D,
		}
	case CGB0, CGBABC: // TODO does CGB0 have the same IO as CGBABC?
		return []uint8{
			0x11, 0x80, 0x00, 0x00, 0x00, 0x08, 0x00, 0x7C,
		}
	case MGB:
		return []uint8{
			0xFF, 0xB0, 0x00, 0x13, 0x00, 0xD8, 0x01, 0x4D,
		}
	case SGB:
		return []uint8{
			0x01, 0x00, 0x00, 0x14, 0x00, 0x00, 0xC0, 0x60,
		}
	case SGB2:
		return []uint8{
			0xFF, 0x00, 0x00, 0x14, 0x00, 0x00, 0xC0, 0x60,
		}
	case AGB:
		return []uint8{
			0x11, 0x00, 0x01, 0x00, 0x00, 0x08, 0x00, 0x7C,
		}
	}

	// default to DMGABC IO
	return []uint8{
		0x01, 0xB0, 0x00, 0x13, 0x00, 0xD8, 0x01, 0x4D,
	}
}

// IO returns the starting CPU IO when PC is write to
// 0x100. This is used to reset the CPU IO to their
// default values.
func (m Model) IO() map[HardwareAddress]interface{} {
	switch m {
	case DMG0:
		return map[HardwareAddress]interface{}{
			DIV:  uint16(0x1899),
			NR10: uint8(0x80),
			NR11: uint8(0xBF),
			NR12: uint8(0xF3),
			NR14: uint8(0xBF),
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
			LY:   uint8(0x91),
			LCDC: uint8(0x91),
			STAT: uint8(0x88),
			BGP:  uint8(0xFC),
			BDIS: uint8(0x01),
			IF:   uint8(0xE1),
		}
	case DMGABC:
		return map[HardwareAddress]interface{}{
			P1:   uint8(0xCF),
			DIV:  uint16(0xABCC),
			TAC:  uint8(0xF8),
			NR10: uint8(0x80),
			NR11: uint8(0xBF),
			NR12: uint8(0xF3),
			NR14: uint8(0xBF),
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
			LCDC: uint8(0x91),
			STAT: uint8(0x87),
			BGP:  uint8(0xFC),
			BDIS: uint8(0x01),
			IF:   uint8(0xE1),
		}
	case CGBABC:
		return map[HardwareAddress]interface{}{
			P1:   uint8(0xFF),
			DIV:  uint16(0xABCC),
			TAC:  uint8(0xF8),
			NR10: uint8(0x80),
			NR11: uint8(0xBF),
			NR12: uint8(0xF3),
			NR14: uint8(0xBF),
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
			LCDC: uint8(0x91),
			STAT: uint8(0x87),
			BGP:  uint8(0xFC),
			BCPS: uint8(0xC8),
			OCPS: uint8(0xD0),
			KEY0: uint8(0xFF),
			FF74: uint8(0xFF),
			BDIS: uint8(0x01),
			IF:   uint8(0xE1),
		}
	default:
		return map[HardwareAddress]interface{}{
			DIV:  uint16(0xABCC),
			NR10: uint8(0x80),
			NR11: uint8(0xBF),
			NR12: uint8(0xF3),
			NR14: uint8(0xBF),
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
			LCDC: uint8(0x91),
			STAT: uint8(0x87),
			BGP:  uint8(0xFC),
			BDIS: uint8(0x01),
		}
	}
}

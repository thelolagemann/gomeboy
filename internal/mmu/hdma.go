package mmu

import "fmt"

type Mode = uint8

const (
	GDMAMode Mode = iota
	HDMAMode
)

type HDMA struct {
	mode         Mode
	transferring bool
	copying      bool

	source      uint16
	destination uint16

	blocks uint8

	bus IOBus
}

func NewHDMA(bus IOBus) *HDMA {
	return &HDMA{
		mode:         GDMAMode,
		transferring: false,
		copying:      false,
		blocks:       1,

		bus: bus,
	}
}

func (h *HDMA) Read(address uint16) uint8 {
	switch address {
	case 0xFF51, 0xFF52, 0xFF53, 0xFF54, 0xFF55: // Reading from HDMA registers returns 0xFF
		return 0xFF
	}

	panic(fmt.Sprintf("hdma\tillegal read from address %04X", address))
}

func (h *HDMA) Write(address uint16, value uint8) {
	switch address {
	case 0xFF51: // HDMA1
		h.source = (h.source & 0x00FF) | uint16(value)<<8
	case 0xFF52: // HDMA2
		h.source = (h.source & 0xFF00) | uint16(value&0xF0)
	case 0xFF53: // HDMA3
		h.destination = (h.destination & 0x00FF) | uint16(value&0x1F)<<8
	case 0xFF54: // HDMA4
		h.destination = (h.destination & 0xFF00) | uint16(value&0xF0)
	case 0xFF55: // HDMA5
		// is HDMA copying?
		if h.mode == HDMAMode && h.copying {
			if value>>7 == GDMAMode {
				// stop the HDMA transfer
				h.transferring = false
			} else {
				// restart the HDMA transfer
				h.mode = value >> 7
				h.blocks = value&0x7F + 1
			}
		} else {
			// start copy
			h.mode = value >> 7
			h.blocks = value&0x7F + 1

			h.transferring = true
		}

		// start GDMA transfer immediately
		if h.mode == GDMAMode {
			h.copying = true
		}
	default:
		panic(fmt.Sprintf("hdma\tillegal write to address %04X", address))
	}
}

func (h *HDMA) Tick() {
	// write to vram
	h.bus.Write(h.destination+0x8000, h.bus.Read(h.source))
	h.destination++
	h.source++

	// has a block been copied?
	if h.destination&0xf == 0 {
		h.blocks--
		if h.blocks == 0 {
			h.transferring = false
			h.copying = false
			h.blocks = 0x80
		}

		if h.mode == HDMAMode {
			h.copying = false
		}
	}
}

// HasDoubleSpeed returns true as the HDMA controller responds to
// double speed mode.
func (h *HDMA) HasDoubleSpeed() bool {
	return true
}

// IsCopying returns true if the HDMA controller is currently copying
// data.
func (h *HDMA) IsCopying() bool {
	return h.copying
}

func (h *HDMA) SetHBlank() {
	if h.mode == HDMAMode && h.transferring {
		h.copying = true
	}
}

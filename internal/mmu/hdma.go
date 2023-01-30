package mmu

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

type Mode = uint8

const (
	GDMAMode Mode = iota
	HDMAMode
)

type HDMA struct {
	mode Mode

	transferring bool
	copying      bool

	blocks        uint8
	source        uint16
	destination   uint16
	bus           IOBus
	vRAMWriteFunc func(uint16, uint8)
}

func NewHDMA(bus IOBus) *HDMA {
	return &HDMA{
		mode:         GDMAMode,
		transferring: false,
		copying:      false,
		blocks:       0x01,

		bus: bus,
	}
}

func (h *HDMA) Read(address uint16) uint8 {
	//fmt.Printf("hdma\tread from address %04X\n", address)
	switch address {
	case 0xFF51, 0xFF52, 0xFF53, 0xFF54: // Reading from HDMA registers returns 0xFF
		return 0xFF
	case 0xFF55: // HDMA5
		// is HDMA transferring?
		if h.transferring {
			return h.blocks - 1
		} else {
			return types.Bit7 | h.blocks - 1
		}
	}

	panic(fmt.Sprintf("hdma\tillegal read from address %04X", address))
}

func (h *HDMA) Write(address uint16, value uint8) {
	//fmt.Printf("hdma\twrite %02X to address %04X\n", value, address)
	switch address {
	case 0xFF51: // HDMA1
		h.source = (h.source & 0x00FF) | (uint16(value) << 8)
	case 0xFF52: // HDMA2
		h.source = (h.source & 0xFF00) | uint16(value&0xF0)
	case 0xFF53: // HDMA3
		h.destination = (h.destination & 0x00FF) | (uint16(value&0x1F) << 8)
	case 0xFF54: // HDMA4
		h.destination = (h.destination & 0xFF00) | uint16(value&0xF0)
	case 0xFF55: // HDMA5
		// is HDMA copying?
		if h.mode == HDMAMode && h.transferring {
			if Mode(value>>7) == GDMAMode {
				// stop the HDMA transfer
				h.transferring = false
			} else {
				// restart the HDMA transfer
				h.mode = value >> 7
				h.blocks = (value & 0x7F) + 1
			}
		} else {
			// start copy
			h.mode = value >> 7
			h.blocks = (value & 0x7F) + 1

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
	h.vRAMWriteFunc(h.destination&0x1FFF, h.bus.Read(h.source))

	// increment source and destination
	h.destination++
	h.source++

	// has a block finished?
	if (h.destination & 0xf) == 0 {
		h.blocks--
		// has the transfer finished?
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

func (h *HDMA) AttachVRAM(vramWriteFunc func(uint16, uint8)) {
	h.vRAMWriteFunc = vramWriteFunc
}

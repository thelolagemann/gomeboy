package mmu

import (
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

func (h *HDMA) init() {
	// setup types
	types.RegisterHardware(
		types.HDMA1,
		func(v uint8) {
			h.source = (h.source & 0x00FF) | (uint16(v) << 8)
		},
		types.NoRead,
	)
	types.RegisterHardware(
		types.HDMA2,
		func(v uint8) {
			h.source = (h.source & 0xFF00) | uint16(v&0xF0)
		},
		types.NoRead,
	)
	types.RegisterHardware(
		types.HDMA3,
		func(v uint8) {
			h.destination = (h.destination & 0x00FF) | (uint16(v&0x1F) << 8)
		},
		types.NoRead,
	)
	types.RegisterHardware(
		types.HDMA4,
		func(v uint8) {
			h.destination = (h.destination & 0xFF00) | uint16(v&0xF0)
		},
		types.NoRead,
	)
	types.RegisterHardware(
		types.HDMA5,
		func(v uint8) {
			// is HDMA copying?
			if h.mode == HDMAMode && h.transferring {
				if Mode(v>>7) == GDMAMode {
					// stop the HDMA transfer
					h.transferring = false
				} else {
					// restart the HDMA transfer
					h.mode = v >> 7
					h.blocks = (v & 0x7F) + 1
				}
			} else {
				// start copy
				h.mode = v >> 7
				h.blocks = (v & 0x7F) + 1

				h.transferring = true
			}

			// start GDMA transfer immediately
			if h.mode == GDMAMode {
				h.copying = true
			}
		},
		func() uint8 {
			// is HDMA transferring?
			if h.transferring {
				return h.blocks - 1
			} else {
				return types.Bit7 | h.blocks - 1
			}
		},
	)
}

func NewHDMA(bus IOBus) *HDMA {
	h := &HDMA{
		mode:         GDMAMode,
		transferring: false,
		copying:      false,
		blocks:       0x01,

		bus: bus,
	}
	h.init()
	return h
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

var _ types.Stater = (*HDMA)(nil)

func (h *HDMA) Load(s *types.State) {
	h.transferring = s.ReadBool()
	h.copying = s.ReadBool()
	h.mode = s.Read8()
	h.source = s.Read16()
	h.destination = s.Read16()
	h.blocks = s.Read8()
}

func (h *HDMA) Save(s *types.State) {
	s.WriteBool(h.transferring)
	s.WriteBool(h.copying)
	s.Write8(h.mode)
	s.Write16(h.source)
	s.Write16(h.destination)
	s.Write8(h.blocks)
}

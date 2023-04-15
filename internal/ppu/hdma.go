package ppu

import (
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/scheduler"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

type HDMA struct {
	length uint8

	hdma5 uint8
	// bits 16 - 4 respected only for source
	source uint16
	// bits 12 - 4 respected only for destination
	destination uint16
	complete    bool

	s    *scheduler.Scheduler
	vRAM func(uint16, uint8)
	bus  mmu.IOBus
}

func NewHDMA(bus *mmu.MMU, vRAM func(uint16, uint8), s *scheduler.Scheduler) *HDMA {
	h := &HDMA{
		vRAM: vRAM,
		s:    s,
		bus:  bus,
	}

	types.RegisterHardware(types.HDMA1, func(v uint8) {
		h.source &= 0x00F0
		h.source |= (uint16(v) << 8) & 0xFF00
	}, types.NoRead)
	types.RegisterHardware(types.HDMA2, func(v uint8) {
		h.source &= 0xFF00
		h.source |= uint16(v) & 0x00F0
	}, types.NoRead)
	types.RegisterHardware(types.HDMA3, func(v uint8) {
		h.destination &= 0x00F0
		h.destination |= (uint16(v) << 8) & 0xFF00
	}, types.NoRead)
	types.RegisterHardware(types.HDMA4, func(v uint8) {
		h.destination &= 0xFF00
		h.destination |= uint16(v) & 0x00F0
	}, types.NoRead)
	types.RegisterHardware(types.HDMA5, func(v uint8) {
		// GDMA if bit 7 isn't set
		if v&types.Bit7 == 0 {
			// is there a pending HDMA transfer?
			if h.hdma5&types.Bit7 == 0 {
				// disable HDMA, keeping the length
				h.hdma5 |= types.Bit7
			} else {
				// otherwise, perform the GDMA transfer
				length := ((v & 0x7F) + 1) * 16 // length in bytes
				for i := uint8(0); i < length; i++ {
					h.vRAM(h.destination&0x1FFF, h.bus.Read(h.source))

					h.source++
					h.destination++
				}
				h.hdma5 = 0xFF
			}
		} else {
			// HDMA
			h.hdma5 = v & 0x7F
		}
	}, func() uint8 {
		return 0xFF
		// TODO verify what happens when reading HDMA5
		// other implementations appear to return the
		// value of the HDMA5 register, however this
		// causes several games to perform corrupt
		// HDMA transfers, so more research is needed.
		// that being said, the HDMA/GDMA implementation
		// here is still incomplete, so it's possible
		// that other issues are causing the corruption
	})

	return h
}

// doHDMA is called during the HBlank period to perform the HDMA transfer
func (h *HDMA) doHDMA() {
	// if the HDMA is disabled, return
	if h.hdma5&types.Bit7 != 0 {
		return
	}

	// perform the transfer
	for i := uint8(0); i < 16; i++ {
		h.vRAM(h.destination&0x1FFF, h.bus.Read(h.source))

		h.source++
		h.destination++
	}

	// decrement the length
	h.hdma5--
}

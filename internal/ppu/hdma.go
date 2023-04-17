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
	bus  *mmu.MMU

	hdmaPaused, hdmaComplete, gdmaComplete bool
	hdmaRemaining                          uint8
}

func NewHDMA(bus *mmu.MMU, vRAM func(uint16, uint8), s *scheduler.Scheduler) *HDMA {
	h := &HDMA{
		vRAM: vRAM,
		s:    s,
		bus:  bus,
	}

	types.RegisterHardware(types.HDMA1, func(v uint8) {
		h.source &= 0xF0
		h.source |= uint16(v) << 8
		if h.source >= 0xE000 {
			h.source |= 0xF000
		}
	}, types.NoRead)
	types.RegisterHardware(types.HDMA2, func(v uint8) {
		h.source &= 0xFF00
		h.source |= uint16(v & 0xF0)
	}, types.NoRead)
	types.RegisterHardware(types.HDMA3, func(v uint8) {
		h.destination &= 0x00F0
		h.destination |= (uint16(v) << 8) & 0xFF00
	}, types.NoRead)
	types.RegisterHardware(types.HDMA4, func(v uint8) {
		h.destination &= 0xFF00
		h.destination |= uint16(v & 0xF0)
	}, types.NoRead)
	types.RegisterHardware(types.HDMA5, func(v uint8) {
		// set the new DMA length
		h.length = (v & 0x7F) + 1 // 0x7F = 127 (0x80 = 128)

		// are we starting a HDMA transfer?
		if v&types.Bit7 != 0 {
			h.hdmaRemaining = h.length
			h.hdmaComplete = false
			h.hdmaPaused = false
			h.gdmaComplete = false
		} else {
			if h.hdmaRemaining > 0 {
				// if we're in the middle of a HDMA transfer, pause it
				h.hdmaPaused = true
				h.gdmaComplete = false
			} else {
				// if we're not in the middle of a HDMA transfer, pause the GDMA
				h.newDMA(h.length)
				h.gdmaComplete = true
			}
		}
	}, func() uint8 {
		if !h.bus.IsGBC() {
			return 0xFF
		}
		if h.hdmaComplete || h.gdmaComplete {
			return 0xFF
		} else {
			return h.hdmaRemaining - 1
		}
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

func (h *HDMA) newDMA(length uint8) {
	for i := uint8(0); i < length; i++ {
		for j := uint8(0); j < 16; j++ {
			// tick the scheduler
			if h.s.DoubleSpeed() {
				h.s.Tick(4)
			} else {
				h.s.Tick(2)
			}

			// perform the transfer
			h.vRAM(h.destination&0x1FFF, h.bus.Read(h.source))

			// increment the source and destination
			h.source++
			h.destination++

			// mask the source and destination
			h.source &= 0xFFFF
			h.destination &= 0xFFFF
		}
	}
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

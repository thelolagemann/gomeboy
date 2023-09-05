package ppu

import (
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/ppu/lcd"
	"github.com/thelolagemann/go-gameboy/internal/scheduler"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

type HDMA struct {
	length uint8

	// bits 16 - 4 respected only for source
	source uint16
	// bits 12 - 4 respected only for destination
	destination uint16
	complete    bool

	s   *scheduler.Scheduler
	ppu *PPU
	bus *mmu.MMU

	hdmaPaused, hdmaComplete, gdmaComplete bool
	hdmaRemaining                          uint8
}

func NewHDMA(bus *mmu.MMU, ppu *PPU, s *scheduler.Scheduler) *HDMA {
	h := &HDMA{
		ppu: ppu,
		s:   s,
		bus: bus,
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
		// update the length
		h.length = (v & 0x7F) + 1

		// if bit 7 is set, we are starting a new HDMA transfer
		if v&types.Bit7 != 0 {
			h.hdmaRemaining = h.length // set the remaining length

			// reset the HDMA flags
			h.hdmaComplete = false
			h.hdmaPaused = false
			h.gdmaComplete = false

			// if the LCD is disabled, one HDMA transfer is performed immediately
			// and the rest are performed during the HBlank period
			if !h.ppu.Enabled && h.hdmaRemaining > 0 {
				h.newDMA(1)
				h.hdmaRemaining--
			}

			// if the PPU is already in the HBlank period, then the HDMA would not be
			// performed by the scheduler until the next HBlank period, so we perform
			// the transfer immediately here and decrement the remaining length
			if h.ppu.Enabled && h.ppu.Mode == lcd.HBlank && h.hdmaRemaining > 0 {
				h.newDMA(1)
				h.hdmaRemaining--
			}
		} else {
			// if bit 7 is not set, we are starting a new GDMA transfer
			if h.hdmaRemaining > 0 {
				// if we're in the middle of a HDMA transfer, pause it
				h.hdmaPaused = true
				h.gdmaComplete = false

				h.hdmaRemaining = h.length
			} else {
				// if we're not in the middle of a HDMA transfer, perform a GDMA transfer
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
			v := uint8(0)
			if h.hdmaPaused {
				v |= types.Bit7
			}
			return v | (h.hdmaRemaining-1)&0x7F
		}
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
			h.ppu.writeVRAM(h.destination&0x1FFF, h.bus.Read(h.source))

			// increment the source and destination
			h.source++
			h.destination++

			// mask the source and destination
			h.source &= 0xFFFF
			h.destination &= 0xFFFF
		}
	}
}

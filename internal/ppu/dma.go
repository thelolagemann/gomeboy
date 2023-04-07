package ppu

import (
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/scheduler"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

// DMA (Direct Memory Access) is used to transfer data from
// the CPU's memory to the PPU's OAM (Object Attribute Memory).
// It is used as a fast alternative to the CPU writing to the
// OAM directly.
//
// The DMA transfer is done in 160 cycles, and the CPU is
// halted during this time. The CPU is also halted for 4 cycles
// after the DMA transfer is done.
type DMA struct {
	source      uint16
	destination uint8
	value       uint8
	remaining   uint8 // 40 * 4 = 160

	active     bool
	enabled    bool
	restarting bool

	bus *mmu.MMU
	oam *OAM

	s *scheduler.Scheduler
}

func NewDMA(bus *mmu.MMU, oam *OAM, s *scheduler.Scheduler) *DMA {
	d := &DMA{
		bus: bus,
		oam: oam,
		s:   s,
	}
	s.RegisterEvent(scheduler.DMATransfer, d.doTransfer)
	s.RegisterEvent(scheduler.DMAStartTransfer, d.startTransfer)
	s.RegisterEvent(scheduler.DMAEndTransfer, func() {
		d.active = false
		d.enabled = false
	})

	// setup register
	types.RegisterHardware(
		types.DMA,
		func(v uint8) {
			source := uint16(v) << 8

			// set the value (reading DMA simply returns the last value written)
			// https://github.com/Gekkio/mooneye-test-suite/blob/main/acceptance/oam_dma/reg_read.s
			d.value = v

			// mark DMA as inactive (it will be active in 1-M-cycle)
			d.active = false

			// handle restarting DMA
			d.restarting = d.enabled

			// update new source, destination and remaining bytes
			d.source = source
			d.destination = 0
			d.remaining = 160

			// reschedule any existing DMA
			if d.restarting {
				// d.s.DescheduleEvent(scheduler.DMATransfer)
				d.s.DescheduleEvent(scheduler.DMAStartTransfer)
				d.s.DescheduleEvent(scheduler.DMATransfer)
				d.s.DescheduleEvent(scheduler.DMAEndTransfer)
			}

			// mark DMA as enabled
			d.enabled = true

			// schedule DMA start for 1-M-cycle
			d.s.ScheduleEvent(scheduler.DMAStartTransfer, 8)

			//d.s.ScheduleEvent(scheduler.DMATransfer, 8)

		},
		func() uint8 {
			return d.value
		},
	)

	return d
}

func (d *DMA) startTransfer() {
	d.active = true
	d.doTransfer()
	d.s.ScheduleEvent(scheduler.DMAEndTransfer, 640)
}

func (d *DMA) doTransfer() {
	if d.restarting {
		d.restarting = false

	}
	// where are we transferring from?
	currentSource := d.source
	if currentSource >= 0xFE00 {
		currentSource -= 0x2000
	}

	// transfer a byte from the source to the destination
	d.oam.Write(uint16(d.destination), d.bus.Read(currentSource))

	// increment source and destination
	d.source++
	d.destination++

	// decrement remaining
	d.remaining--

	// if there are still bytes to transfer, schedule another transfer
	if d.remaining > 0 {
		d.s.ScheduleEvent(scheduler.DMATransfer, 4)
	}
}

func (d *DMA) IsTransferring() bool {
	return d.active || d.restarting
}

var _ types.Stater = (*DMA)(nil)

func (d *DMA) Load(s *types.State) {
	d.source = s.Read16()
	d.value = s.Read8()
	d.oam.Load(s)
}

func (d *DMA) Save(s *types.State) {
	s.Write16(d.source)
	s.Write8(d.value)
	d.oam.Save(s)
}

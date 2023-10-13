package ppu

import (
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
)

// DMA (Direct Memory Access) is used to transfer data from
// the CPU's memory to the PPU's OAM (Object Attribute Memory).
//
// The CPU is unable to directly access the PPU's OAM whilst the
// display is being updated; as this is most of the time, DMA
// transfers are used instead.
//
// A DMA transfer copies data from ROM or RAM to the OAM, taking
// 160 M-cycles to complete. The source address is specified by
// the CPU's DMA register.
type DMA struct {
	source      uint16
	destination uint8
	remaining   uint8 // 40 * 4 = 160
	lastByte    uint8

	active     bool
	enabled    bool
	restarting bool

	b   *io.Bus
	oam *OAM

	s *scheduler.Scheduler
}

func NewDMA(b *io.Bus, oam *OAM, s *scheduler.Scheduler) *DMA {
	d := &DMA{
		b:   b,
		oam: oam,
		s:   s,
	}
	s.RegisterEvent(scheduler.DMATransfer, d.doTransfer)
	s.RegisterEvent(scheduler.DMAStartTransfer, d.startTransfer)
	s.RegisterEvent(scheduler.DMAEndTransfer, func() {
		d.active = false
		d.enabled = false
	})

	b.ReserveAddress(types.DMA, func(v byte) byte {
		source := uint16(v) << 8

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

		// return the value written to the DMA register (for the CPU)
		// https://github.com/Gekkio/mooneye-test-suite/blob/main/acceptance/oam_dma/reg_read.s
		return v
	})

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

	d.lastByte = d.b.Read(currentSource)

	// transfer a byte from the source to the destination
	d.oam.Write(uint16(d.destination), d.lastByte)

	//fmt.Printf("writing %x to OAM[%x]\n", d.lastByte, d.destination)
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

// IsConflicting returns true if the given address is conflicting with
// the current bus that the DMA transfer is reading data from.
func (d *DMA) IsConflicting(addr uint16) bool {
	// is source on VRAM bus?
	if d.source >= 0x8000 && d.source <= 0x9FFF {
		return addr >= 0x8000 && addr <= 0x9FFF
	}
	// is source on ROM+WRAM+SRAM bus?
	if d.source <= 0x7FFF || (d.source >= 0xA000 && d.source <= 0xFEFF) {
		return addr <= 0x7FFF || (addr >= 0xA000 && addr <= 0xFEFF)
	}

	return false
}

func (d *DMA) IsTransferring() bool {
	return d.active || d.restarting
}

func (d *DMA) LastByte() uint8 {
	return d.lastByte
}

var _ types.Stater = (*DMA)(nil)

func (d *DMA) Load(s *types.State) {
	d.source = s.Read16()
	d.oam.Load(s)
}

func (d *DMA) Save(s *types.State) {
	s.Write16(d.source)
	d.oam.Save(s)
}

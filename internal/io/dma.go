package io

import "github.com/thelolagemann/gomeboy/internal/scheduler"

// when dma transfer occurs, set conflicting area
// when conflicting - reads return last byte written
// writes are ineffective
// also handle oam/vram locking

// startDMATransfer
// DMA (Direct Memory Access) is used to transfer data from
// the CPU's memory to the PPU's OAM (Object Attribute Memory).
//
// The CPU is unable to directly access the PPU's OAM whilst the
// display is being updated; as this is most of the time, DMA
// transfers are used instead.
//
// A DMA transfer copies data from ROM or RAM to the OAM, taking
// 160 M-cycles to complete. The source address is specified by
// the types.DMA register.
func (b *Bus) startDMATransfer() {
	b.dmaActive = true
	b.doDMATransfer()
	b.s.ScheduleEvent(scheduler.DMAEndTransfer, 640)
}

func (b *Bus) doDMATransfer() {
	// handle restarting latch
	b.dmaRestarting = false

	// copy byte from source to OAM
	b.dmaConflict = b.data[b.dmaSource]
	b.data[b.dmaDestination] = b.dmaConflict

	// increment source and destination
	b.dmaSource++
	b.dmaDestination++

	// are we at the end of the transfer? (dest = 0xFEA0)
	if b.dmaDestination < 0xFEA0 {
		b.s.ScheduleEvent(scheduler.DMATransfer, 4)
	}

	// set OAM changed so PPU knows to update
	b.oamChanged = true
}

func (b *Bus) isDMATransferring() bool {
	return b.dmaActive || b.dmaRestarting
}

func (b *Bus) OAMChanged() bool {
	return b.oamChanged
}

func (b *Bus) OAMCatchup(f func(uint16, uint8)) {
	// write new values to OAM
	for i := 0; i < 160; i++ {
		f(uint16(i), b.data[0xFE00+uint16(i)])
	}

	b.oamChanged = false
}

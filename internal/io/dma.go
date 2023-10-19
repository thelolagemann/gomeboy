package io

import (
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
)

var (
	noConflicts = [16]bool{}
)

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
// writing a value to the types.DMA register, which will then get
// bit shifted by 8 to provide the source address.
//
// E.g. If a write to types.DMA is 0x84, then the source becomes
// 0x8400 (0x84 << 8).
func (b *Bus) startDMATransfer() {
	b.dmaActive = true
	b.doDMATransfer()
	b.s.ScheduleEvent(scheduler.DMAEndTransfer, 640)
}

// doDMATransfer transfers a single byte from the source to the
// PPU's OAM.
// todo conflict
// todo oam changed
// todo increment
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

// isDMATransferring returns true if the CPU is currently
// executing a DMA transfer.
func (b *Bus) isDMATransferring() bool {
	return b.dmaActive || b.dmaRestarting
}

// OAMChanged returns true when the OAM memory is updated.
// This is used by the PPU to determine when to recalculate
// its sprites.
func (b *Bus) OAMChanged() bool {
	return b.oamChanged
}

// OAMCatchup sets the OAMChanged flag to false. This is used
// by the PPU to indicate to the bus that it has caught up to
// the current values in the OAM.
func (b *Bus) OAMCatchup() {
	b.oamChanged = false
}

func (b *Bus) newDMA(length uint8) {
	for i := uint8(0); i < length; i++ {
		for j := uint8(0); j < 16; j++ {
			// tick the scheduler
			if b.s.DoubleSpeed() {
				b.s.Tick(4)
			} else {
				b.s.Tick(2)
			}

			// perform the transfer
			b.Write(b.hdmaDestination|0x8000, b.Get(b.hdmaSource))

			// increment the source and destination
			b.hdmaSource++
			b.hdmaDestination++
		}
	}
}

func (b *Bus) HandleHDMA() {
	// is there any remaining data to transfer and
	// has the DMA not been paused?
	if b.dmaRemaining > 0 && !b.dmaPaused {
		// update HDMA5 register as the next DMA will tick
		b.Set(types.HDMA5, b.Get(types.HDMA5)&0x80|(b.dmaRemaining-1)&0x7f)
		b.newDMA(1)
		b.dmaRemaining--
	} else if !b.dmaPaused {
		b.dmaRemaining = 0
		b.dmaComplete = true
		b.Set(types.HDMA5, 0xFF)
	}
}

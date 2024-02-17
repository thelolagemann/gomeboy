package io

import (
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
)

// startDMATransfer initiates a DMA transfer.
func (b *Bus) startDMATransfer() {
	b.dmaActive = true
	b.dmaRestarting = false
	b.doDMATransfer()
	b.s.ScheduleEvent(scheduler.DMAEndTransfer, 640)
}

// doDMATransfer transfers a single byte from the source to the
// PPU's OAM.
func (b *Bus) doDMATransfer() {
	// copy byte from source to OAM
	b.dmaConflict = b.data[b.dmaSource]

	// set OAM changed so PPU knows to update
	b.oamChanged = true

	b.data[b.dmaDestination] = b.dmaConflict

	// increment source and destination
	b.dmaSource++
	b.dmaDestination++

	// are we at the end of the transfer? (dest = 0xFEA0)
	if b.dmaDestination < 0xFEA0 {
		b.s.ScheduleEvent(scheduler.DMATransfer, 4)
	}
}

func (b *Bus) endDMATransfer() {
	b.dmaActive = false
	b.dmaEnabled = false

	// clear any conflicts
	b.dmaConflicted = 0
	b.dmaConflict = 0xff
}

// isDMATransferring returns true if the CPU is currently
// executing a DMA transfer.
func (b *Bus) isDMATransferring() bool {
	return b.dmaActive || b.dmaRestarting
}

func (b *Bus) OAMChanged() bool {
	return b.oamChanged
}

// OAMCatchup calls f with the OAM memory region
func (b *Bus) OAMCatchup(f func([160]byte)) {
	f([160]byte(b.data[0xfe00 : 0xfe00+160]))
	b.oamChanged = false
}

func (b *Bus) VRAMChanged() bool {
	return b.vramChanged
}

// VRAMCatchup calls f with any pending vRAM changes.
func (b *Bus) VRAMCatchup(f func([]VRAMChange)) {
	f(b.vramChanges)
	b.vramChanges = b.vramChanges[:0]
	b.vramChanged = false
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
			b.Write(b.hdmaDestination&0x1fff|0x8000, b.Get(b.hdmaSource))

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

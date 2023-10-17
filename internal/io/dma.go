package io

import "github.com/thelolagemann/gomeboy/internal/scheduler"

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

func (b *Bus) OAMCatchup() {

	b.oamChanged = false
}

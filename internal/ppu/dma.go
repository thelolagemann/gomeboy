package ppu

import (
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/ram"
	"github.com/thelolagemann/go-gameboy/internal/types/registers"
)

type DMA struct {
	enabled    bool
	restarting bool

	timer  uint
	source uint16
	value  uint8

	bus mmu.IOBus
	oam ram.RAM
}

func (d *DMA) init() {
	// setup register
	registers.RegisterHardware(
		registers.DMA,
		func(v uint8) {
			d.value = v
			d.source = uint16(v) << 8
			d.timer = 0

			d.restarting = d.enabled
			d.enabled = true
		},
		func() uint8 {
			return d.value
		},
	)
}

func NewDMA(bus mmu.IOBus, oam ram.RAM) *DMA {
	d := &DMA{
		bus: bus,
		oam: oam,
	}
	d.init()
	return d
}

func (d *DMA) Tick() {
	if !d.enabled {
		return
	}

	// increment the timer
	d.timer++

	// every 4 ticks, transfer a byte
	if d.timer%4 == 0 {
		d.restarting = false //

		offset := uint16(d.timer-4) >> 2
		currentSource := d.source + (offset)

		// is a DMA trying to read from the OAM?
		if currentSource > 0xE000 {
			// if so, make sure we don't read from the OAM
			// and instead read from the source address - 0x2000
			currentSource &^= 0x2000
		}
		// write directly to OAM to avoid any locking
		d.oam.Write(offset, d.bus.Read(currentSource))

		// 4 ticks per byte (0xFE00-0xFE9F) = 160 bytes = 640 ticks
		if d.timer >= 640 {
			d.enabled = false
			d.timer = 0
		}
	}
}

// HasDoubleSpeed returns true as the DMA controller responds to
// double speed mode.
func (d *DMA) HasDoubleSpeed() bool {
	return true
}

func (d *DMA) IsTransferring() bool {
	return d.timer > 4 || d.restarting
}

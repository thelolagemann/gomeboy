package ppu

import "github.com/thelolagemann/go-gameboy/internal/mmu"

type DMA struct {
	enabled    bool
	restarting bool

	timer  uint16
	source uint16
	value  uint8

	bus mmu.IOBus
}

func NewDMA(bus mmu.IOBus) *DMA {
	return &DMA{
		bus: bus,
	}
}

func (d *DMA) Read(address uint16) uint8 {
	return d.value
}

func (d *DMA) Write(address uint16, value uint8) {
	d.value = value
	d.source = uint16(value) << 8
	d.timer = 0

	d.restarting = d.enabled
	d.enabled = true
}

func (d *DMA) Tick() {
	if !d.enabled {
		return
	}

	// increment the timer
	d.timer++

	// if the timer is done, copy the value
	if d.timer > 4 {
		d.restarting = false

		offset := (d.timer - 4) >> 2
		currentSource := d.source + offset

		// is OAM trying to read from itself? (>= 0xFE00)
		if currentSource >= 0xFE00 {
			// if so, make sure we don't read from the OAM
			// and instead read from the source address
			// minus 0x2000
			currentSource -= 0x2000
		}

		// write to OAM
		d.bus.Write(0xFE00+uint16(offset), d.bus.Read(currentSource))

		if d.timer > 160*4+4 {
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

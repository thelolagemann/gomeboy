package ppu

import (
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

type DMA struct {
	enabled    bool
	restarting bool

	timer  uint16
	source uint16
	value  uint8

	bus mmu.IOBus
	oam *OAM
}

func (d *DMA) init() {
	// setup register
	types.RegisterHardware(
		types.DMA,
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

func NewDMA(bus mmu.IOBus, oam *OAM) *DMA {
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

	// every 4 ticks, transfer a byte to OAM
	// takes 4 ticks to turn on, 640 ticks to transfer
	if d.timer > 4 {
		d.restarting = false
		if d.timer < 644 {
			offset := uint16(d.timer-4) >> 2
			currentSource := d.source + (offset)

			// is a DMA trying to read from the OAM?
			if currentSource >= 0xFE00 {
				// if so, make sure we don't read from the OAM
				// and instead read from the source address - 0x2000
				currentSource -= 0x2000
			}
			// load the value from the source address
			d.oam.Write(offset, d.bus.Read(currentSource))

		}

	}
	if d.timer > 644 {
		d.enabled = false
		d.timer = 0
	}
}

func (d *DMA) IsTransferring() bool {
	return d.timer > 4 || d.restarting
}

var _ types.Stater = (*DMA)(nil)

func (d *DMA) Load(s *types.State) {
	d.enabled = s.ReadBool()
	d.restarting = s.ReadBool()
	d.timer = s.Read16()
	d.source = s.Read16()
	d.value = s.Read8()
	d.oam.Load(s)
}

func (d *DMA) Save(s *types.State) {
	s.WriteBool(d.enabled)
	s.WriteBool(d.restarting)
	s.Write16(d.timer)
	s.Write16(d.source)
	s.Write8(d.value)
	d.oam.Save(s)
}

package ppu

import (
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

type DMA struct {
	Enabled    bool
	restarting bool

	timer  uint16
	source uint16
	value  uint8

	bus *mmu.MMU
	oam *OAM

	cycleFunc func()
}

func (d *DMA) init() {
	// setup register
	types.RegisterHardware(
		types.DMA,
		func(v uint8) {
			d.value = v
			d.source = uint16(v) << 8
			d.timer = 0

			d.restarting = d.Enabled
			d.Enabled = true
			d.cycleFunc()
		},
		func() uint8 {
			return d.value
		},
	)
}

func NewDMA(bus *mmu.MMU, oam *OAM) *DMA {
	d := &DMA{
		bus: bus,
		oam: oam,
	}
	d.init()
	return d
}

// TickM ticks the DMA by 1 M-cycle (4 T-cycles)
func (d *DMA) TickM() {
	d.TickT()
	d.TickT()
	d.TickT()
	d.TickT()
}

// TickT ticks the DMA by 1 T-cycle.
func (d *DMA) TickT() {
	// increment the timer
	d.timer++

	// every 4 ticks, transfer a byte to OAM
	// takes 4 ticks to turn on, 640 ticks to transfer
	if d.timer > 4 {
		d.restarting = false
		if d.timer < 644 {
			offset := (d.timer - 4) >> 2
			currentSource := d.source + (offset)

			// is a DMA trying to read from the OAM?
			if currentSource >= 0xFE00 {
				// if so, make sure we don't read from the OAM
				// and instead read from the source address - 0x2000
				currentSource -= 0x2000
			}
			value := d.bus.Read(currentSource)
			if d.oam.data[offset] != value {
				// load the value from the source address
				d.oam.Write(offset, value)
			}
		}

	}
	if d.timer > 644 {
		d.Enabled = false
		d.timer = 0
		d.cycleFunc()
	}
}

func (d *DMA) IsTransferring() bool {
	return d.timer > 4 || d.restarting
}

var _ types.Stater = (*DMA)(nil)

func (d *DMA) Load(s *types.State) {
	d.Enabled = s.ReadBool()
	d.restarting = s.ReadBool()
	d.timer = s.Read16()
	d.source = s.Read16()
	d.value = s.Read8()
	d.oam.Load(s)
}

func (d *DMA) Save(s *types.State) {
	s.WriteBool(d.Enabled)
	s.WriteBool(d.restarting)
	s.Write16(d.timer)
	s.Write16(d.source)
	s.Write8(d.value)
	d.oam.Save(s)
}

func (d *DMA) AttachRegenerate(cycle func()) {
	d.cycleFunc = cycle
}

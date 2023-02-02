package mmu

import "github.com/thelolagemann/go-gameboy/internal/types/registers"

type WRAM struct {
	bank uint8
	raw  [8][0x1000]uint8
}

func (w *WRAM) init() {
	// setup registers
	registers.RegisterHardware(
		registers.SVBK,
		func(v uint8) {
			v &= 0x07 // only 3 bits are used
			if v == 0 {
				v = 1
			}
			w.bank = v
		}, func() uint8 {
			return w.bank
		},
	)
}

func NewWRAM() *WRAM {
	w := &WRAM{
		bank: 1, // bank 1 is the default as the first bank is fixed
	}
	w.init()
	return w
}

func (w *WRAM) Read(addr uint16) uint8 {
	// are we reading from the fixed bank?
	if addr < 0xD000 {
		return w.raw[0][addr&0xFFF]
	}
	// are we reading from the switchable bank?
	if addr < 0xE000 {
		return w.raw[w.bank][addr&0xFFF]
	}
	// are we reading from the echo? (bank 0)
	if addr < 0xF000 {
		return w.raw[0][addr&0xFFF]
	}
	// are we reading from the echo? (bank 1-7)
	return w.raw[w.bank][addr&0xFFF]
}

func (w *WRAM) Write(addr uint16, v uint8) {
	// are we writing to the fixed bank?
	if addr < 0xD000 {
		w.raw[0][addr&0xFFF] = v
		return
	}
	// are we writing to the switchable bank?
	if addr < 0xE000 {
		w.raw[w.bank][addr&0xFFF] = v
		return
	}
	// are we writing to the echo?
	if addr < 0xF000 {
		w.raw[0][addr&0xFFF] = v
		return
	}
	// are we writing to the echo?
	w.raw[w.bank][addr&0xFFF] = v
}

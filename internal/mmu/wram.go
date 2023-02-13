package mmu

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

type WRAM struct {
	bank uint8
	raw  [8][0x1000]uint8
}

func NewWRAM() *WRAM {
	w := &WRAM{
		bank: 1, // bank 1 is the default as the first bank is fixed
	}

	types.RegisterHardware(
		types.SVBK,
		func(v uint8) {
			fmt.Printf("WRAM bank set to %d\n", v&0x7)
			w.bank = v & 0x07
			if w.bank == 0 {
				w.bank = 1
			}
		}, func() uint8 {
			fmt.Printf("WRAM bank read as %d\n", w.bank)
			return w.bank
		},
	)

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

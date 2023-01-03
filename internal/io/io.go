// Package io provides the input and output of the Game Boy.
package io

import (
	"github.com/sirupsen/logrus"
	"github.com/thelolagemann/go-gameboy/internal/apu"
	"github.com/thelolagemann/go-gameboy/internal/io/timer"
	"github.com/thelolagemann/go-gameboy/internal/joypad"
	"github.com/thelolagemann/go-gameboy/pkg/log"
)

// Bus represents the Bus of the Game Boy.
type Bus interface {
	// Sound returns the Sound.
	Sound() *apu.APU
	// Input returns the Input.
	Input() *joypad.State
	// Interrupts returns the InterruptFlag.
	Interrupts() *Interrupts
	// Serial returns the Serial.
	Serial() *Serial
	// Timer returns the Timer.
	Timer() *timer.Controller

	Video() IOBus

	Log() log.Logger
	AttachMMU(mmu IOBus)

	IOBus
}

type IOBus interface {
	Read(addr uint16) uint8
	Write(addr uint16, value uint8)
}

type io struct {
	sound     *apu.APU
	input     *joypad.State
	interrupt *Interrupts
	serial    *Serial
	timer     *timer.Controller
	log       log.Logger

	video IOBus
	mmu   IOBus
}

func (i *io) AttachMMU(mmu IOBus) {
	i.mmu = mmu
}

func (i *io) Write(addr uint16, value uint8) {
	//TODO implement me
	panic("implement me")
}

func (i *io) Video() IOBus {
	return i.video
}

func (i *io) Sound() *apu.APU {
	return i.sound
}

func (i *io) Input() *joypad.State {
	return i.input
}

func (i *io) Interrupts() *Interrupts {
	return i.interrupt
}

func (i *io) Serial() *Serial {
	return i.serial
}

func (i *io) Timer() *timer.Controller {
	return i.timer
}

// NewIO returns a new Bus.
func NewIO(video IOBus) Bus {
	input := joypad.New()
	interrupt := NewInterrupts()
	serial := NewSerial()
	timerCtl := timer.NewController()
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	bus := &io{
		input:     input,
		interrupt: interrupt,
		serial:    serial,
		timer:     timerCtl,
		log:       logger,
		video:     video,
	}

	return bus
}

func (i *io) Log() log.Logger {
	return i.log
}

// Read reads the value at the given address.
func (i *io) Read(addr uint16) uint8 {

	switch {
	// Joypad
	case addr == 0xFF00:
		return i.input.Read()
	// Serial
	case addr == 0xFF01:
		return i.serial.Read()
	case addr == 0xFF02 || addr == 0xFF03:
		return 0x7e
	// Timer
	case addr >= 0xFF04 && addr <= 0xFF07:
		return i.timer.Read(addr)
	// Interrupts
	case addr == 0xFF0F || addr == 0xFFFF:
		return i.interrupt.Read(addr)
	// Sound
	case addr >= 0xFF10 && addr <= 0xFF3F:
		return i.sound.Read(addr)
	}
	return i.mmu.Read(addr)
}

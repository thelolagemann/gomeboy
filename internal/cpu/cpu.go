package cpu

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/apu"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"github.com/thelolagemann/go-gameboy/internal/serial"
	"github.com/thelolagemann/go-gameboy/internal/timer"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"time"
)

const (
	// ClockSpeed is the clock speed of the CPU.
	ClockSpeed = 4194304
)

type mode = uint8

const (
	// ModeNormal is the normal CPU mode.
	ModeNormal mode = iota
	// ModeHalt is the halt CPU mode.
	ModeHalt
	// ModeStop is the stop CPU mode.
	ModeStop
	// ModeHaltBug is the halt bug CPU mode.
	ModeHaltBug
	// ModeHaltDI is the halt DI CPU mode.
	ModeHaltDI
	// ModeEnableIME is the enable IME CPU mode.
	ModeEnableIME
)

// CPU represents the Gameboy CPU. It is responsible for executing instructions.
type CPU struct {
	// PC is the program counter, it points to the next instruction to be executed.
	PC uint16
	// SP is the stack pointer, it points to the top of the stack.
	SP uint16
	// Registers contains the 8-bit registers, as well as the 16-bit register pairs.
	Registers

	// Speed is the current speed of the CPU.
	Speed float32

	doubleSpeed bool

	mmu *mmu.MMU
	IRQ *interrupts.Service

	Debug           bool
	DebugBreakpoint bool

	// components that need to be ticked
	dma    *ppu.DMA
	timer  *timer.Controller
	ppu    *ppu.PPU
	sound  *apu.APU
	serial *serial.Controller

	currentTick uint8
	mode        mode
}

// NewCPU creates a new CPU instance with the given MMU.
// The MMU is used to read and write to the memory.
func NewCPU(mmu *mmu.MMU, irq *interrupts.Service, dma *ppu.DMA, timer *timer.Controller, ppu *ppu.PPU, sound *apu.APU, serial *serial.Controller) *CPU {
	c := &CPU{
		Registers: Registers{},
		mmu:       mmu,
		Speed:     1,
		IRQ:       irq,
		dma:       dma,
		timer:     timer,
		ppu:       ppu,
		sound:     sound,
		serial:    serial,
	}
	// create register pairs
	c.BC = &RegisterPair{&c.B, &c.C}
	c.DE = &RegisterPair{&c.D, &c.E}
	c.HL = &RegisterPair{&c.H, &c.L}
	c.AF = &RegisterPair{&c.A, &c.F}

	return c
}

// registerIndex returns a Register pointer for the given index.
func (c *CPU) registerIndex(index uint8) *Register {
	switch index {
	case 0:
		return &c.B
	case 1:
		return &c.C
	case 2:
		return &c.D
	case 3:
		return &c.E
	case 4:
		return &c.H
	case 5:
		return &c.L
	case 7:
		return &c.A
	}
	panic(fmt.Sprintf("invalid register index: %d", index))
}

// registerName returns the name of a Register.
func (c *CPU) registerName(reg *Register) string {
	switch reg {
	case &c.A:
		return "A"
	case &c.B:
		return "B"
	case &c.C:
		return "C"
	case &c.D:
		return "D"
	case &c.E:
		return "E"
	case &c.H:
		return "H"
	case &c.L:
		return "L"
	}
	return ""
}

// Step the CPU by one frame and returns the
// number of ticks that have been executed.
func (c *CPU) Step() uint8 {
	// reset tick counter
	c.currentTick = 0

	// should we tick HDMA?
	if c.mmu.HDMA != nil && c.mmu.HDMA.IsCopying() {
		c.hdmaTick4()
		return 0 // TODO determine correct value to return
	}

	reqInt := false
	if c.mode == ModeNormal {
		// execute step normally
		c.runInstruction(c.readInstruction())

		// check for interrupts, in normal mode this requires the IME to be enabled
		reqInt = c.IRQ.IME && c.hasInterrupts()
	} else {
		// execute step based on mode
		switch c.mode {
		case ModeHalt, ModeStop:
			// in stop, halt mode, the CPU ticks 4 times, but does not execute any instructions
			c.tickCycle()

			// check for interrupts, in stop mode the IME is ignored, this
			// is so that the CPU can be woken up by an interrupt while in halt, stop mode
			reqInt = c.hasInterrupts()
		case ModeHaltDI:
			c.tickCycle()

			// check for interrupts
			if c.hasInterrupts() {
				c.mode = ModeNormal
			}
		case ModeEnableIME:
			// Enabling IME, and set mode to normal
			c.IRQ.IME = true
			c.mode = ModeNormal

			// run one instruction
			c.runInstruction(c.readInstruction())

			// check for interrupts
			reqInt = c.IRQ.IME && c.hasInterrupts()
		case ModeHaltBug:
			instr := c.readInstruction()
			c.PC--
			c.runInstruction(instr)
			c.mode = ModeNormal
			reqInt = c.IRQ.IME && c.hasInterrupts()
		}
	}

	// did we get an interrupt?
	if reqInt {
		c.executeInterrupt()
	}

	return c.currentTick
}

// tickDoubleSpeed ticks the CPU components twice as
// fast, if they respond to the double speed flag.
func (c *CPU) tickDoubleSpeed() {
	c.dma.Tick()
	c.timer.Tick()
	c.serial.Tick(c.timer.Div)
}

func (c *CPU) hdmaTick4() {
	if c.doubleSpeed {
		c.tick()
		c.tickDoubleSpeed()
		c.tick()
		c.tickDoubleSpeed()

		c.mmu.HDMA.Tick() // HDMA takes twice as long in double speed mode
	} else {
		c.tick()
		c.tick()
		c.tick()
		c.tick()

		c.mmu.HDMA.Tick()
		c.mmu.HDMA.Tick()
	}
}

func (c *CPU) hasInterrupts() bool {
	return c.IRQ.Enable&c.IRQ.Flag != 0
}

// readInstruction reads the next instruction from memory.
func (c *CPU) readInstruction() uint8 {
	c.tickCycle()
	value := c.mmu.Read(c.PC)
	c.PC++
	return value
}

// readOperand reads the next operand from memory. The same as
// readInstruction, but will allow future optimizations.
func (c *CPU) readOperand() uint8 {
	c.tickCycle()
	value := c.mmu.Read(c.PC)
	c.PC++
	return value
}

func (c *CPU) skipOperand() {
	c.tickCycle()
	c.PC++
}

// readByte reads a byte from memory.
func (c *CPU) readByte(addr uint16) uint8 {
	c.tickCycle()
	return c.mmu.Read(addr)
}

// writeByte writes the given value to the given address.
func (c *CPU) writeByte(addr uint16, val uint8) {
	c.tickCycle()
	c.mmu.Write(addr, val)
}

func (c *CPU) runInstruction(opcode uint8) {
	currentPC := c.PC - 1
	var instruction Instruction
	// do we need to run a CB instruction?
	if opcode == 0xCB {
		// read the next instruction
		cbIns := InstructionSetCB[c.readOperand()]
		instruction = cbIns
	} else {
		// get the instruction
		ins := InstructionSet[opcode]
		instruction = ins
	}

	// execute the instruction
	instruction.fn(c)
	if false {
		fmt.Printf("%s (%d ticks)", instruction.name, c.currentTick)
		fmt.Println()
		fmt.Printf("A: %02x F: %02x B: %02x C: %02x D: %02x E: %02x H: %02x L: %02x SP: %04x PC: %04x\n", c.A, c.F, c.B, c.C, c.D, c.E, c.H, c.L, c.SP, currentPC)
		time.Sleep(20 * time.Millisecond)
	}
	// check for debug
	if c.Debug {
		if instruction.name == "LD B, B" {
			c.DebugBreakpoint = true
		}
	}
}

func (c *CPU) executeInterrupt() {
	// is IME enabled?
	if c.IRQ.IME {
		// save the high byte of the PC
		c.SP--
		c.writeByte(c.SP, uint8(c.PC>>8))

		vector := c.IRQ.Vector()

		// save the low byte of the PC
		c.SP--
		c.writeByte(c.SP, uint8(c.PC&0xFF))

		// jump to the interrupt vector and disable IME
		c.PC = uint16(vector)
		c.IRQ.IME = false

		// tick 12 times
		c.tickCycle()
		c.tickCycle()
		c.tickCycle()
	}

	// set the mode to normal
	c.mode = ModeNormal
}

// tick the various components of the CPU.
func (c *CPU) tick() {
	c.dma.Tick()
	c.timer.Tick()
	c.serial.Tick(c.timer.Div)
	c.ppu.Tick()
	c.sound.Tick()
	c.currentTick++
}

func (c *CPU) tickCycle() {
	if c.doubleSpeed {
		c.tick()
		c.tickDoubleSpeed()
		c.tick()
		c.tickDoubleSpeed()
	} else {
		c.tick()
		c.tick()
		c.tick()
		c.tick()
	}
}

// shouldZeroFlag sets FlagZero if the given value is 0.
func (c *CPU) shouldZeroFlag(value uint8) {
	if value == 0 {
		c.setFlag(FlagZero)
	} else {
		c.clearFlag(FlagZero)
	}
}

var _ types.Stater = (*CPU)(nil)

func (c *CPU) Load(s *types.State) {
	c.A = s.Read8()
	c.F = s.Read8()
	c.B = s.Read8()
	c.C = s.Read8()
	c.D = s.Read8()
	c.E = s.Read8()
	c.H = s.Read8()
	c.L = s.Read8()
	c.SP = s.Read16()
	c.PC = s.Read16()
	c.mode = s.Read8()
	c.doubleSpeed = s.ReadBool()
	c.IRQ.Load(s)
}

func (c *CPU) Save(s *types.State) {
	s.Write8(c.A)
	s.Write8(c.F)
	s.Write8(c.B)
	s.Write8(c.C)
	s.Write8(c.D)
	s.Write8(c.E)
	s.Write8(c.H)
	s.Write8(c.L)
	s.Write16(c.SP)
	s.Write16(c.PC)
	s.Write8(c.mode)
	s.WriteBool(c.doubleSpeed)
	c.IRQ.Save(s)
}

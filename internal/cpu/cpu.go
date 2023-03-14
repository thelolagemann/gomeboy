package cpu

import (
	"github.com/thelolagemann/go-gameboy/internal/apu"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"github.com/thelolagemann/go-gameboy/internal/serial"
	"github.com/thelolagemann/go-gameboy/internal/timer"
	"github.com/thelolagemann/go-gameboy/internal/types"
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

	doubleSpeed bool

	mmu *mmu.MMU
	IRQ *interrupts.Service

	Debug           bool
	DebugBreakpoint bool

	currentTick uint8
	mode        mode

	registerSlice [8]*uint8
	tickCycle     func()

	instructions   [256]func(cpu *CPU)
	instructionsCB [256]func(cpu *CPU)
}

// NewCPU creates a new CPU instance with the given MMU.
// The MMU is used to read and write to the memory.
func NewCPU(mmu *mmu.MMU, irq *interrupts.Service, dma *ppu.DMA, timerCtl *timer.Controller, ppu *ppu.PPU, sound *apu.APU, serialCtl *serial.Controller) *CPU {
	c := &CPU{
		Registers: Registers{},
		mmu:       mmu,
		IRQ:       irq,
	}
	// create register pairs
	c.BC = &RegisterPair{&c.B, &c.C}
	c.DE = &RegisterPair{&c.D, &c.E}
	c.HL = &RegisterPair{&c.H, &c.L}
	c.AF = &RegisterPair{&c.A, &c.F}

	// create tick fn
	c.tickCycle = func() {
		dma.TickM()
		timerCtl.TickM()
		serialCtl.TickM(timerCtl.Div)

		var count int
		if c.doubleSpeed {
			count = 2
		} else {
			count = 4
		}
		ppu.Tick(count)
		sound.Tick(count)
		c.currentTick += 4
	}

	var n uint8
	c.registerSlice = [8]*uint8{&c.B, &c.C, &c.D, &c.E, &c.H, &c.L, &n, &c.A}

	// embed the instruction set
	c.instructions = [256]func(*CPU){}
	c.instructionsCB = [256]func(*CPU){}
	for i := 0; i < 256; i++ {
		c.instructions[i] = InstructionSet[i].fn
		c.instructionsCB[i] = InstructionSetCB[i].fn
	}
	return c
}

// registerIndex returns a Register pointer for the given index.
func (c *CPU) registerIndex(index uint8) Register {
	return *c.registerSlice[index]
}

// registerPointer returns a Register pointer for the given index.
func (c *CPU) registerPointer(index uint8) *Register {
	return c.registerSlice[index]
}

// Step the CPU by one frame and returns the
// number of ticks that have been executed.
func (c *CPU) Step() uint8 {
	// reset tick counter
	c.currentTick = 0

	// should we tick HDMA?
	if c.mmu.HDMA != nil && c.mmu.HDMA.IsCopying() {
		c.hdmaTick4()
		return c.currentTick
	}

	reqInt := false

	// execute step based on mode
	switch c.mode {
	case ModeNormal:
		// execute step normally
		c.runInstruction(c.readInstruction())

		// check for interrupts, in normal mode this requires the IME to be enabled
		reqInt = c.IRQ.IME && c.IRQ.HasInterrupts()
	case ModeHalt, ModeStop:
		// in stop, halt mode, the CPU ticks 4 times, but does not execute any instructions
		c.tickCycle()

		// check for interrupts, in stop mode the IME is ignored, this
		// is so that the CPU can be woken up by an interrupt while in halt, stop mode
		reqInt = c.IRQ.HasInterrupts()
	case ModeHaltDI:
		c.tickCycle()

		// check for interrupts
		if c.IRQ.HasInterrupts() {
			c.mode = ModeNormal
		}
	case ModeEnableIME:
		// Enabling IME, and set mode to normal
		c.IRQ.IME = true
		c.mode = ModeNormal

		// run one instruction
		c.runInstruction(c.readInstruction())

		// check for interrupts
		reqInt = c.IRQ.IME && c.IRQ.HasInterrupts()
	case ModeHaltBug:
		instr := c.readInstruction()
		c.PC--
		c.runInstruction(instr)
		c.mode = ModeNormal
		reqInt = c.IRQ.IME && c.IRQ.HasInterrupts()
	}

	// did we get an interrupt?
	if reqInt {
		c.executeInterrupt()
	}

	return c.currentTick
}

func (c *CPU) hdmaTick4() {
	if c.doubleSpeed {
		c.tickCycle()

		c.mmu.HDMA.Tick() // HDMA takes twice as long in double speed mode
	} else {
		c.tickCycle()

		c.mmu.HDMA.Tick()
		c.mmu.HDMA.Tick()
	}
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
	// do we need to run a CB instruction?
	if opcode == 0xCB {
		// read the next instruction
		c.instructionsCB[c.readOperand()](c)
	} else {
		// get the instruction
		c.instructions[opcode](c)
	}

	// have we hit a breakpoint? (only if debug is enabled) LD B, B
	if c.Debug && opcode == 0x40 {
		c.DebugBreakpoint = true

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
		c.PC = vector
		c.IRQ.IME = false

		// tick 12 times
		c.tickCycle()
		c.tickCycle()
		c.tickCycle()
	}

	// set the mode to normal
	c.mode = ModeNormal
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

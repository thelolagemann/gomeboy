package cpu

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
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

	// Speed is the current speed of the CPU.
	Speed float32

	mmu     *mmu.MMU
	stopped bool
	Halted  bool
	irq     *interrupts.Service

	Debug           bool
	DebugBreakpoint bool

	peripherals []types.Peripheral

	currentTick uint16
	mode        mode
}

// PopStack pops a 16-bit value from the stack.
func (c *CPU) PopStack() uint16 {
	value := c.mmu.Read16(c.SP)
	c.SP += 2
	return value
}

// PushStack pushes a 16-bit value to the stack.
func (c *CPU) PushStack(pc uint16) {
	c.SP -= 2
	c.mmu.Write16(c.SP, pc)
}

// NewCPU creates a new CPU instance with the given MMU.
// The MMU is used to read and write to the memory.
func NewCPU(mmu *mmu.MMU, irq *interrupts.Service, peripherals ...types.Peripheral) *CPU {
	c := &CPU{
		PC: 0,
		SP: 0,
		Registers: Registers{
			A: 0x11,
			B: 0x01,
			C: 0x00,
			D: 0xFF,
			E: 0x08,
			F: 0x00,
			H: 0xB0,
			L: 0x7C,
		},
		mmu:         mmu,
		Speed:       1,
		stopped:     false,
		Halted:      false,
		irq:         irq,
		peripherals: peripherals,
	}
	// create register pairs
	c.BC = &RegisterPair{&c.B, &c.C}
	c.DE = &RegisterPair{&c.D, &c.E}
	c.HL = &RegisterPair{&c.H, &c.L}
	c.AF = &RegisterPair{&c.A, &c.F}

	// generate instructions
	c.generateBitInstructions()
	c.generateLoadRegisterToRegisterInstructions()
	c.generateLogicInstructions()
	c.generateRSTInstructions()

	if len(InstructionSet) != 256 || len(InstructionSetCB) != 256 {
		panic("invalid instruction set")
	}

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

// registerMap maps a Register name to a Register pointer.
func (c *CPU) registerMap(name string) *Register {
	switch name {
	case "A":
		return &c.A
	case "B":
		return &c.B
	case "C":
		return &c.C
	case "D":
		return &c.D
	case "E":
		return &c.E
	case "F":
		return &c.F
	case "H":
		return &c.H
	case "L":
		return &c.L
	}
	return nil
}

// registerPairMap maps a RegisterPair name to a RegisterPair pointer.
func (c *CPU) registerPairMap(name string) *RegisterPair {
	switch name {
	case "AF":
		return c.AF
	case "BC":
		return c.BC
	case "DE":
		return c.DE
	case "HL":
		return c.HL
	}
	return nil
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
func (c *CPU) Step() uint16 {
	// TODO handle CGB HDMA
	reqInt := false

	// execute step based on mode
	switch c.mode {
	case ModeNormal:
		// execute step normally
		c.runInstruction(c.readInstruction())

		// check for interrupts, in normal mode this requires the IME to be enabled
		reqInt = c.irq.IME && c.hasInterrupts()
	case ModeHalt, ModeStop:
		// in stop mode, the CPU ticks 4 times, but does not execute any instructions
		c.ticks(4)

		// check for interrupts, in stop mode the IME is ignored
		reqInt = c.hasInterrupts()
	case ModeHaltBug:
		// TODO implement halt bug
		fmt.Println("waiting for halt bug")
		panic("halt bug")
	case ModeHaltDI:
		c.ticks(4)

		// check for interrupts
		if c.hasInterrupts() {
			c.mode = ModeNormal
		}
	case ModeEnableIME:
		// Enabling IME, and set mode to normal
		c.irq.IME = true
		c.mode = ModeNormal

		// run one instruction
		c.runInstruction(c.readInstruction())

		// check for interrupts
		reqInt = c.irq.IME && c.hasInterrupts()
	}

	// did we get an interrupt?
	if reqInt {
		c.executeInterrupt()
	}
	ticks := c.currentTick
	c.currentTick = 0
	return ticks
}

func (c *CPU) hasInterrupts() bool {
	return c.irq.Enable.Read()&c.irq.Flag.Read()&0x1F != 0
}

// readInstruction reads the next instruction from memory.
func (c *CPU) readInstruction() uint8 {
	c.ticks(4)
	value := c.mmu.Read(c.PC)
	c.PC++
	return value
}

// readOperand reads the next operand from memory. The same as
// readInstruction, but will allow future optimizations.
func (c *CPU) readOperand() uint8 {
	c.ticks(4)
	value := c.mmu.Read(c.PC)
	c.PC++
	// fmt.Println("readOperand", value)
	return value
}

func (c *CPU) skipOperand() {
	c.ticks(4)
	c.PC++
}

// readByte reads a byte from memory.
func (c *CPU) readByte(addr uint16) uint8 {
	c.ticks(4)
	return c.mmu.Read(addr)
}

// writeByte writes the given value to the given address.
func (c *CPU) writeByte(addr uint16, val uint8) {
	c.ticks(4)
	c.mmu.Write(addr, val)
}

func (c *CPU) runInstruction(opcode uint8) {
	var instruction Instructor
	// do we need to run a CB instruction?
	if opcode == 0xCB {
		// read the next instruction
		cbIns, ok := InstructionSetCB[c.readInstruction()]
		if !ok {
			panic(fmt.Sprintf("invalid CB instruction: %x", opcode))
		}

		instruction = cbIns.Instruction()
	} else {
		// get the instruction
		ins, ok := InstructionSet[opcode]
		if !ok {
			panic(fmt.Sprintf("invalid instruction: %x", opcode))
		}
		instruction = ins
	}

	// execute the instruction
	instruction.Execute(c)

	// check for debug
	if c.Debug {
		if instruction.Name() == "LD B, B" {
			c.DebugBreakpoint = true
		}
	}
}

func (c *CPU) executeInterrupt() {
	// is IME enabled?
	if c.irq.IME {
		// save the high byte of the PC
		c.SP--
		c.writeByte(c.SP, uint8(c.PC>>8))

		vector := c.irq.Vector()

		// save the low byte of the PC
		c.SP--
		c.writeByte(c.SP, uint8(c.PC&0xFF))

		// jump to the interrupt vector and disable IME
		c.PC = uint16(vector)
		c.irq.IME = false

		// tick 12 times
		c.ticks(12)
	}

	// set the mode to normal
	c.mode = ModeNormal
}

// tick the various components of the CPU.
func (c *CPU) tick() {
	//c.dma.Tick()
	for _, p := range c.peripherals {
		p.Tick()
	}
	c.currentTick++
}

func (c *CPU) ticks(n uint) {
	for i := uint(0); i < n; i++ {
		c.tick()
	}
}

// rst resets the CPU.
func (c *CPU) rst(v uint8) {
	c.push(uint8(c.PC>>8), uint8(c.PC&0xFF))
	c.PC = uint16(v)
}

// setFlags sets the given flags to the given values.
func (c *CPU) setFlags(flags ...Flag) {
	for _, flag := range flags {
		c.setFlag(flag)
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

// pop pops a value from the stack.
func (c *CPU) pop() uint8 {
	value := c.mmu.Read(c.SP)
	c.SP++
	return value
}

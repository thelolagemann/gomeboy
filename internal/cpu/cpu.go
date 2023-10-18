package cpu

import (
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
)

// CPU represents the Gameboy CPU. It is responsible for executing instructions.
type CPU struct {
	// PC is the program counter, it points to the next instruction to be executed.
	PC uint16
	// SP is the stack pointer, it points to the top of the stack.
	SP uint16
	// Registers contains the 8-bit registers, as well as the 16-bit register pairs.
	types.Registers
	Debug           bool
	DebugBreakpoint bool

	ime bool

	doubleSpeed bool

	b *io.Bus

	instructions   [256]func(cpu *CPU)
	instructionsCB [256]func(cpu *CPU)

	hasFrame bool
	s        *scheduler.Scheduler
	model    types.Model
	ppu      *ppu.PPU
	stopped  bool
}

func (c *CPU) SetModel(model types.Model) {
	c.model = model
}

// NewCPU creates a new CPU instance with the given MMU.
// The MMU is used to read and write to the memory.
func NewCPU(b *io.Bus, sched *scheduler.Scheduler, ppu *ppu.PPU) *CPU {
	c := &CPU{
		Registers: types.Registers{},
		b:         b,
		s:         sched,
		ppu:       ppu,
	}

	// create register pairs
	c.BC = &types.RegisterPair{High: &c.B, Low: &c.C}
	c.DE = &types.RegisterPair{High: &c.D, Low: &c.E}
	c.HL = &types.RegisterPair{High: &c.H, Low: &c.L}
	c.AF = &types.RegisterPair{High: &c.A, Low: &c.F}

	// embed the instruction set
	c.instructions = [256]func(*CPU){}
	c.instructionsCB = [256]func(*CPU){}
	for i := 0; i < 256; i++ {
		c.instructions[i] = InstructionSet[i].fn
		c.instructionsCB[i] = InstructionSetCB[i].fn
	}

	sched.RegisterEvent(scheduler.EIPending, func() {
		c.ime = true
	})
	sched.RegisterEvent(scheduler.EIHaltDelay, func() {
		c.ime = true

		c.PC--
	})

	return c
}

// Boot emulates the boot process by setting the initial
// register values and HW registers of the provided model.
func (c *CPU) Boot(m types.Model) {
	// PC, SP is the same across all models
	c.PC = 0x100
	c.SP = 0xFFFE

	// get the CPU registers
	startingRegs := m.Registers()
	for i, reg := range []*uint8{&c.A, &c.F, &c.B, &c.C, &c.D, &c.E, &c.H, &c.L} {
		*reg = startingRegs[i]
	}

	// get the IO registers
	//ioRegs := m.IO()
	/*
		for i := 0xFF08; i < 0xFF80; i++ {
			if reg, ok := ioRegs[types.HardwareAddress(i)]; ok {
				switch reg.(type) {
				case uint8:

					c.b.Set(uint16(i), reg.(uint8))
				case uint16:
					// TODO handle uint16 values
				}
			} else {
				c.b.Set(uint16(i), 0xFF)
			}
		}*/
}

// skipHALT invokes the scheduler to "skip" until the next
// event triggering an interrupt occurs. This is used when
// the CPU is in HALT mode and the IME is enabled.
func (c *CPU) skipHALT() {
	for !c.b.HasInterrupts() {
		c.s.Skip()
	}
}

// Frame steps the CPU until the next frame is rzeady.
func (c *CPU) Frame() {
	for !c.hasFrame && !c.DebugBreakpoint {
		instr := c.readOperand()

		// execute the instruction
		c.instructions[instr](c)

		// did we get an interrupt?
		if c.ime && c.b.HasInterrupts() {
			c.executeInterrupt()
		}

	}
	c.hasFrame = false
}

// readOperand reads the next operand from memory.
func (c *CPU) readOperand() uint8 {
	value := c.b.ClockedRead(c.PC)
	c.PC++
	return value
}

func (c *CPU) skipOperand() {
	c.s.Tick(4)
	c.PC++
}

func (c *CPU) executeInterrupt() {
	// disable interrupts
	c.ime = false

	// 8 cycles
	c.s.Tick(4)
	c.s.Tick(4)

	// save PC to stack
	c.SP--
	c.b.ClockedWrite(c.SP, uint8(c.PC>>8))

	irq := c.b.Get(types.IE) // IRQ check saved for later

	c.SP--
	c.b.ClockedWrite(c.SP, uint8(c.PC&0xFF))

	// get vector from IRQ
	c.PC = c.b.IRQVector(irq)

	// final 4 cycles
	c.s.Tick(4)
}

// clearFlag clears the given flag in the F register,
// leaving all other flags unchanged. If the flag
// is already cleared, this function does nothing. To
// set a flag, use setFlag.
func (c *CPU) clearFlag(flag types.Flag) {
	c.F &^= flag
}

// setFlags sets all the flags in the F register,
// as specified by the given arguments.
func (c *CPU) setFlags(Z bool, N bool, H bool, C bool) {
	v := uint8(0)
	if Z {
		v |= types.FlagZero
	}
	if N {
		v |= types.FlagSubtract
	}
	if H {
		v |= types.FlagHalfCarry
	}
	if C {
		v |= types.FlagCarry
	}
	c.F = v
}

// isFlagSet returns true if the given flag is set,
// false otherwise.
func (c *CPU) isFlagSet(flag types.Flag) bool {
	return c.F&flag == flag
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
	c.doubleSpeed = s.ReadBool()
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
	s.WriteBool(c.doubleSpeed)
}

func (c *CPU) HasFrame() {
	c.hasFrame = true
}

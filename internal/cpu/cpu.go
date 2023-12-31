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
	Registers
	Debug           bool
	DebugBreakpoint bool

	shouldInt bool
	hasFrame  bool

	doubleSpeed bool

	b *io.Bus

	instructions [256]func(cpu *CPU)

	s            *scheduler.Scheduler
	ppu          *ppu.PPU
	skippingHalt bool
}

// NewCPU creates a new CPU instance with the given MMU.
// The MMU is used to read and write to the memory.
func NewCPU(b *io.Bus, sched *scheduler.Scheduler, ppu *ppu.PPU) *CPU {
	c := &CPU{
		Registers: Registers{},
		b:         b,
		s:         sched,
		ppu:       ppu,
	}

	// create register pairs
	c.BC = &RegisterPair{High: &c.B, Low: &c.C}
	c.DE = &RegisterPair{High: &c.D, Low: &c.E}
	c.HL = &RegisterPair{High: &c.H, Low: &c.L}
	c.AF = &RegisterPair{High: &c.A, Low: &c.F}

	// embed the instruction set
	c.instructions = [256]func(*CPU){}
	for i := 0; i < 256; i++ {
		c.instructions[i] = InstructionSet[i].fn
	}

	b.ReserveAddress(0xFF7D, func(b byte) byte {
		c.shouldInt = true
		c.hasFrame = true
		return 0xff
	})
	b.Set(0xff7d, 0xff)
	b.ReserveAddress(0xFF7E, func(b byte) byte {
		c.shouldInt = true

		return 0xff
	})
	b.Set(0xff7e, 0xff)

	sched.RegisterEvent(scheduler.EIPending, func() {
		c.b.EnableInterrupts()
	})
	sched.RegisterEvent(scheduler.EIHaltDelay, func() {
		c.b.EnableInterrupts()

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
	for !c.hasFrame && !c.b.HasInterrupts() {
		c.s.Skip()
	}

	// if we came out of the halt skip because a frame was rendered
	// then we need to indicate to the cpu that we should latch back
	// onto halt skipping on the next frame
	if c.hasFrame && !c.b.HasInterrupts() {
		c.skippingHalt = true
	}
}

// Frame steps the CPU until the next frame is ready.
func (c *CPU) Frame() {
	// check to see if we should skip the next frame
	if c.skippingHalt {
		c.skippingHalt = false
		c.skipHALT()
	}

	// shouldInt is triggered on 3 conditions
	// 1. IME = 1 && types.IE & types.IF &0x1f != 0
	// 2. DebugBreakpoint = true
	// 3. hasFrame = true
step:
	for ; !c.shouldInt; c.instructions[c.readOperand()](c) {
	}

	// check to see if shouldInt was triggered by an interrupt
	if c.b.CanInterrupt() {
		c.executeInterrupt()

		// if no other conditions prevent us from stepping, then go back to stepping
		if !c.shouldInt {
			goto step
		}
	}

	// if we have handled the interrupt and shouldInt is still true
	// then we have either rendered a frame, or hit a debug breakpoint
	c.hasFrame = false
	c.shouldInt = c.DebugBreakpoint
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

	c.b.DisableInterrupts()
	c.shouldInt = c.hasFrame || c.DebugBreakpoint
}

// Register represents a GB Register which is used to hold an 8-bit value.
// The CPU has 8 registers: A, B, C, D, E, H, L, and F. The F register is
// special in that it is used to hold the flags.
type Register = uint8

// RegisterPair represents a pair of GB Registers which is used to hold a 16-bit
// value. The CPU has 4 register pairs: AF, BC, DE, and HL.
type RegisterPair struct {
	High *Register
	Low  *Register
}

// Uint16 returns the value of the RegisterPair as an uint16.
func (r *RegisterPair) Uint16() uint16 {
	return uint16(*r.High)<<8 | uint16(*r.Low)
}

// SetUint16 sets the value of the RegisterPair to the given value.
func (r *RegisterPair) SetUint16(value uint16) {
	*r.High = uint8(value >> 8)
	*r.Low = uint8(value)
}

// Registers represents the GB CPU registers.
type Registers struct {
	A Register
	B Register
	C Register
	D Register
	E Register
	F Register
	H Register
	L Register

	BC *RegisterPair
	DE *RegisterPair
	HL *RegisterPair
	AF *RegisterPair
}

// flag represents a flag in the F register, which is
// used to hold the status of various mathematical
// operations.
//
// On the official hardware, the F register is 8 bits
// wide, but only the upper 4 bits are used. The lower
// 4 bits are always 0.
//
// The upper 4 bits are laid out as follows:
//
//	Bit 7 - (Z) FlagZero
//	Bit 6 - (N) FlagSubtract
//	Bit 5 - (H) FlagHalfCarry
//	Bit 4 - (C) FlagCarry
type flag = uint8

const (
	// flagZero is set when the result of an operation is 0.
	//
	// Examples:
	//  SUB A, B; A = 0x00, B = 0x00 -> FlagZero is set
	//  SUB A, B; A = 0x02, B = 0x01 -> FlagZero is not set
	//  DEC A; A = 0x01 -> FlagZero is set
	//  DEC A; A = 0x00 -> FlagZero is not set
	//  INC A; A = 0x00 -> FlagZero is not set
	//  INC A; A = 0xFF -> FlagZero is set
	flagZero = types.Bit7
	// flagSubtract is set when an operation performs a subtraction.
	//
	// Examples:
	//  SUB A, B; A = 0x00, B = 0x00 -> FlagSubtract is set
	//  SUB A, B; A = 0x02, B = 0x01 -> FlagSubtract is set
	//  ADD A, B; A = 0x00, B = 0x00 -> FlagSubtract is not set
	//  ADD A, B; A = 0x02, B = 0x01 -> FlagSubtract is not set
	//  DEC A; A = 0x01 -> FlagSubtract is set
	//  DEC A; A = 0x00 -> FlagSubtract is set
	//  INC A; A = 0x00 -> FlagSubtract is not set
	//  INC A; A = 0xFF -> FlagSubtract is not set
	flagSubtract = types.Bit6
	// flagHalfCarry is set when there is a carry from the lower nibble to
	// the upper nibble, or with 16-bit operations, when there is a carry
	// from the lower byte to the upper byte.
	//
	// Examples:
	//   ADD A, B; A = 0x0F, B = 0x01 -> FlagHalfCarry is set
	//   ADD A, B; A = 0x04, B = 0x01 -> FlagHalfCarry is not set
	//   ADD HL, BC; HL = 0x00FF, BC = 0x0001 -> FlagHalfCarry is set
	//   ADD HL, BC; HL = 0x000F, BC = 0x0001 -> FlagHalfCarry is not set
	flagHalfCarry = types.Bit5
	// flagCarry is set when there is a mathematical operation that has a
	// result that is too large to fit in the destination register.
	//
	// Examples:
	//   ADD A, B; A = 0xFF, B = 0x01 -> FlagCarry is set
	//   ADD A, B; A = 0x04, B = 0x01 -> FlagCarry is not set
	//   ADD HL, BC; HL = 0xFFFF, BC = 0x0001 -> FlagCarry is set
	//   ADD HL, BC; HL = 0x00FF, BC = 0x0001 -> FlagCarry is not set
	flagCarry = types.Bit4
)

// clearFlag clears the given flag in the F register,
// leaving all other flags unchanged.
func (c *CPU) clearFlag(flag flag) {
	c.F &^= flag
}

// setFlags sets all the flags in the F register,
// as specified by the given arguments.
func (c *CPU) setFlags(Z bool, N bool, H bool, C bool) {
	v := uint8(0)
	if Z {
		v |= flagZero
	}
	if N {
		v |= flagSubtract
	}
	if H {
		v |= flagHalfCarry
	}
	if C {
		v |= flagCarry
	}
	c.F = v
}

// isFlagSet returns true if the given flag is set,
// false otherwise.
func (c *CPU) isFlagSet(flag flag) bool {
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

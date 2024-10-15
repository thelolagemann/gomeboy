package cpu

import (
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
)

// CPU represents the Game Boy's 8-bit CPU (sm83).
type CPU struct {
	PC, SP uint16 // (P)rogram (C)ounter, (S)tack (P)ointer
	Registers

	Debug, DebugBreakpoint            bool
	DoubleSpeed, Halted, skippingHalt bool
	hasInt, hasFrame                  bool

	registerPointers [8]*uint8

	b *io.Bus
	s *scheduler.Scheduler
}

func NewCPU(b *io.Bus, sched *scheduler.Scheduler) *CPU {
	c := &CPU{
		b: b,
		s: sched,
	}

	c.BC = RegisterPair{&c.B, &c.C}
	c.DE = RegisterPair{&c.D, &c.E}
	c.HL = RegisterPair{&c.H, &c.L}
	c.AF = RegisterPair{&c.A, &c.F}
	var hl uint8
	c.registerPointers = [8]*uint8{&c.B, &c.C, &c.D, &c.E, &c.H, &c.L, &hl, &c.A}

	b.InterruptCallback = func(v uint8) {
		if v&io.VBlankINT > 0 {
			c.hasFrame = true
		}
		c.hasInt = true
	}

	sched.RegisterEvent(scheduler.EIPending, c.b.EnableInterrupts)
	sched.RegisterEvent(scheduler.EIHaltDelay, func() { c.b.EnableInterrupts(); c.PC-- })

	return c
}

// Boot emulates the boot process by setting the Registers to the starting value provided
// by the types.Model.
func (c *CPU) Boot(m types.Model) {
	// PC, SP is the same across all models
	c.PC = 0x100
	c.SP = 0xFFFE

	// get the CPU registers
	startingRegs := types.ModelRegisters[m]
	for i, reg := range []*uint8{&c.A, &c.F, &c.B, &c.C, &c.D, &c.E, &c.H, &c.L} {
		*reg = startingRegs[i]
	}
}

// Frame steps the CPU until the next frame is ready.
func (c *CPU) Frame() {
	// check to see if we should resume halt skipping
	if c.skippingHalt {
		c.skippingHalt = false
		c.skipHALT()
	}

	// hasInt is triggered on 3 conditions
	// 1. IME = 1 && types.IE & types.IF &0x1f != 0
	// 2. Debug && DebugBreakpoint = true
	// 3. hasFrame = true
step:
	for ; !c.hasInt; InstructionSet[c.readOperand()].fn(c) {
	}

	// check to see if hasInt was triggered by an interrupt
	if c.b.CanInterrupt() {
		c.s.Tick(4)
		c.s.Tick(4)

		// save PC to stack
		c.SP--
		c.b.ClockedWrite(c.SP, uint8(c.PC>>8))

		irq := c.b.Get(types.IE) // IRQ check saved for later

		c.SP--
		c.b.ClockedWrite(c.SP, uint8(c.PC&0xFF))

		c.PC = c.b.IRQVector(irq) // get irq vector
		c.s.Tick(4)

		c.b.DisableInterrupts()

		// if no other conditions prevent us from stepping, then go back to stepping
		if !(c.hasFrame || (c.Debug && c.DebugBreakpoint)) {
			c.hasInt = false
			goto step
		}
	}

	// if we have handled the interrupt and hasInt is still true
	// then we have either rendered a frame, or hit a debug breakpoint
	c.hasFrame = false
	c.hasInt = c.Debug && c.DebugBreakpoint
}

// skipHALT invokes the scheduler to "skip" until the next
// event triggering an interrupt occurs. This is used when
// the CPU is in HALT mode and the IME is enabled.
func (c *CPU) skipHALT() {
	c.Halted = true
	for !c.hasFrame && !c.b.HasInterrupts() {
		c.s.Skip()
	}

	// if we came out of the halt skip because a frame was rendered
	// but there are no pending interrupts, then we need to indicate
	// to the cpu that we should latch back onto halt skipping on the
	// next frame
	if c.hasFrame && !c.b.HasInterrupts() {
		c.skippingHalt = true
	} else {
		c.Halted = false
	}
}

func (c *CPU) readOperand() (v uint8) { v = c.b.ClockedRead(c.PC); c.PC++; return } // read next operand from memory
func (c *CPU) skipOperand()           { c.s.Tick(4); c.PC++ }                       // skip the next operand from memory

// Register represents a Register used to hold an 8-bit value.
type Register = uint8

// RegisterPair is a pair of Registers used to address them as a 16-bit value.
type RegisterPair [2]*Register

func (r RegisterPair) Uint16() uint16         { return uint16(*r[0])<<8 | uint16(*r[1]) }         // get 16-bit value
func (r RegisterPair) SetUint16(value uint16) { *r[0] = uint8(value >> 8); *r[1] = uint8(value) } // set 16-bit value

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

	BC RegisterPair
	DE RegisterPair
	HL RegisterPair
	AF RegisterPair
}

// flag represents a flag in the F register, which is
// used to hold the status of various mathematical
// operations.
//
// The lower 4 bits are disconnected and always read 0.
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

// setFlags sets all the flags in the F register,
// as specified by the given arguments.
func (c *CPU) setFlags(Z, N, H, C bool) {
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

func (c *CPU) isFlagSet(flag flag) bool { return c.F&flag == flag } // is the given flag set?
func (c *CPU) clearFlag(flag flag)      { c.F &^= flag }            // clear the given flag

// addSPSigned adds the signed value of the next
// operand to the SP register, and returns the
// result.
func (c *CPU) addSPSigned() uint16 {
	value := c.readOperand()
	result := uint16(int32(c.SP) + int32(int8(value)))

	tmpVal := c.SP ^ uint16(int8(value)) ^ result

	c.setFlags(false, false, tmpVal&0x10 == 0x10, tmpVal&0x100 == 0x100)

	c.s.Tick(4)
	return result
}

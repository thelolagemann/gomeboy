package cpu

import (
	"github.com/thelolagemann/gomeboy/internal/interrupts"
	"github.com/thelolagemann/gomeboy/internal/mmu"
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

	mmu *mmu.MMU
	irq *interrupts.Service

	instructions   [256]func(cpu *CPU)
	instructionsCB [256]func(cpu *CPU)

	cartFixedBank [0x4000]byte
	io            *[65536]*types.Address

	isGBC    bool
	isMBC1   bool
	hasFrame bool
	s        *scheduler.Scheduler
	model    types.Model
	ppu      *ppu.PPU
	stopped  bool
}

func (c *CPU) SetModel(model types.Model) {
	c.model = model
}

func (c *CPU) AttachIO(io *[65536]*types.Address) {
	c.io = io
}

// NewCPU creates a new CPU instance with the given MMU.
// The MMU is used to read and write to the memory.
func NewCPU(mmu *mmu.MMU, irq *interrupts.Service, sched *scheduler.Scheduler, ppu *ppu.PPU) *CPU {
	c := &CPU{
		Registers: types.Registers{},
		mmu:       mmu,
		irq:       irq,
		isMBC1:    mmu.IsMBC1,
		isGBC:     mmu.IsGBC(),
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

	// read the fixed bank of the cartridge
	for i := uint16(0); i < 0x4000; i++ {
		c.cartFixedBank[i] = mmu.Cart.Read(i)
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

// skipHALT invokes the scheduler to "skip" until the next
// event triggering an interrupt occurs. This is used when
// the CPU is in HALT mode and the IME is enabled.
func (c *CPU) skipHALT() {
	for !c.irq.HasInterrupts() {
		c.s.Skip()
	}
}

// Frame steps the CPU until the next frame is ready.
func (c *CPU) Frame() {
	for !c.hasFrame && !c.DebugBreakpoint {
		// execute the instruction
		c.instructions[c.readOperand()](c)

		// did we get an interrupt?
		if c.ime && c.irq.Enable&c.irq.Flag != 0 {
			c.executeInterrupt()
		}

	}
	c.hasFrame = false
}

// readOperand reads the next operand from memory.
func (c *CPU) readOperand() uint8 {
	value := c.readByte(c.PC)
	c.PC++
	return value
}

func (c *CPU) skipOperand() {
	c.s.Tick(4)
	c.PC++
}

// readByte reads a byte from memory.
func (c *CPU) readByte(addr uint16) uint8 {
	c.s.Tick(4)

	switch {
	case c.ppu.DMA.IsTransferring() && c.ppu.DMA.IsConflicting(addr):
		return c.ppu.DMA.LastByte()
	case c.mmu.BootROM != nil && !c.mmu.IsBootROMDone():
		if addr < 0x100 {
			return c.mmu.BootROM.Read(addr)
		}
		if addr >= 0x200 && addr < 0x900 {
			return c.mmu.BootROM.Read(addr)
		}
	case addr < 0x4000 && !c.isMBC1:
		// can we avoid the call to mmu? MBC1 is special
		return c.cartFixedBank[addr]
	case addr >= 0xFF00 && addr <= 0xFF7F:
		// are we trying to read an IO register?
		return c.io[addr].Read(addr)
	}

	return c.mmu.Read(addr)
}

// writeByte writes the given value to the given address.
func (c *CPU) writeByte(addr uint16, val uint8) {
	c.s.Tick(4)

	if c.ppu.DMA.IsTransferring() && c.ppu.DMA.IsConflicting(addr) {
		// TODO ^^ this is incorrect but enough to pass most anti-drm checks
		return
	}
	c.mmu.Write(addr, val)
}

func (c *CPU) executeInterrupt() {
	// disable interrupts
	c.ime = false

	// 8 cycles
	c.s.Tick(4)
	c.s.Tick(4)

	// save PC to stack
	c.SP--
	c.writeByte(c.SP, uint8(c.PC>>8))

	irq := c.irq.Enable // IRQ check saved for later

	c.SP--
	c.writeByte(c.SP, uint8(c.PC&0xFF))

	// get vector from IRQ
	c.PC = c.irq.Vector(irq)

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
	c.irq.Load(s)
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
	c.irq.Save(s)
}

func (c *CPU) HasFrame() {
	c.hasFrame = true
}

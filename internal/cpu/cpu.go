package cpu

import (
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"github.com/thelolagemann/go-gameboy/internal/ppu/lcd"
	"github.com/thelolagemann/go-gameboy/internal/scheduler"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"sort"
	"time"
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

	ime bool

	doubleSpeed bool

	mmu *mmu.MMU
	irq *interrupts.Service

	registerSlice [8]*uint8

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
		Registers: Registers{},
		mmu:       mmu,
		irq:       irq,
		isMBC1:    mmu.IsMBC1,
		isGBC:     mmu.IsGBC(),
		s:         sched,
		ppu:       ppu,
	}

	// create register pairs
	c.BC = &RegisterPair{&c.B, &c.C}
	c.DE = &RegisterPair{&c.D, &c.E}
	c.HL = &RegisterPair{&c.H, &c.L}
	c.AF = &RegisterPair{&c.A, &c.F}

	var n uint8
	c.registerSlice = [8]*uint8{&c.B, &c.C, &c.D, &c.E, &c.H, &c.L, &n, &c.A}

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

// doHALTBug is called when the CPU is in HALT mode and
// the IME is disabled. It will execute the next instruction
// and then return to the HALT instruction.
func (c *CPU) doHALTBug() {
	// read the next instruction
	instr := c.readOpcode()

	// decrement the PC to execute the instruction again
	c.PC--

	// execute the instruction
	c.instructions[instr](c)

	// did we get an interrupt?
	if c.irq.Flag&c.irq.Enable != 0 {
		c.executeInterrupt()
	}
}

// registerIndex returns a Register pointer for the given index.
func (c *CPU) registerIndex(index uint8) Register {
	return *c.registerSlice[index]
}

// registerPointer returns a Register pointer for the given index.
func (c *CPU) registerPointer(index uint8) *Register {
	return c.registerSlice[index]
}

func (c *CPU) handleOAMCorruption(pos uint16) {
	if c.model == types.CGBABC || c.model == types.CGB0 {
		return // no corruption on CGB
	}
	if pos >= 0xFE00 && pos < 0xFEFF {
		if (c.ppu.Mode == lcd.OAM ||
			c.s.Until(scheduler.PPUContinueOAMSearch) == 4) &&
			c.s.Until(scheduler.PPUEndOAMSearch) != 8 {
			// TODO
			// get the current cycle of mode 2 that the PPU is in
			// the oam is split into 20 rows of 8 bytes each, with
			// each row taking 1 M-cycle to read
			// so we need to figure out which row we're in
			// and then perform the oam corruption
			c.ppu.WriteCorruptionOAM()
		}
	}
}

// Frame steps the CPU until the next frame is ready.
func (c *CPU) Frame() {
	for !c.hasFrame && !c.DebugBreakpoint {
		// execute the instruction
		c.instructions[c.readOpcode()](c)

		// did we get an interrupt?
		if c.ime && c.irq.Enable&c.irq.Flag != 0 {
			c.executeInterrupt()
		}

	}
	c.hasFrame = false
}

// readOpcode reads the next instruction from memory.
func (c *CPU) readOpcode() uint8 {
	value := c.readByte(c.PC)
	c.PC++
	return value
}

// readOperand reads the next operand from memory. The same as
// readOpcode, but will allow future optimizations.
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
	c.mmu.Write(addr, val)
}

// LogUsedInstructions sorts the used instructions by the number of times they have
// been executed, and logs them in a human-readable format, in descending order.
func (c *CPU) LogUsedInstructions() {
	// c.mmu.Log.Infof("Instruction\t Count\t Time")
	type instructionResult struct {
		instruction string
		count       uint64
		time        time.Duration
	}
	results := make([]instructionResult, 512)
	/*for i := 0; i < 256; i++ {
		if c.usedInstructions[i] > 10 {
			results = append(results, instructionResult{
				instruction: InstructionSet[i].name,
				count:       c.usedInstructions[i],
				time:        c.usedInstructionsTime[i],
			})
		}
		if c.usedInstructions[i+256] > 10 {
			results = append(results, instructionResult{
				instruction: InstructionSetCB[i].name,
				count:       c.usedInstructions[i+256],
				time:        c.usedInstructionsTime[i+256],
			})
		}
	}*/

	// sort the instructions by time taken
	sort.Slice(results, func(i, j int) bool {
		return results[i].time > results[j].time
	})

	for i := 0; i < 256; i++ {
		if results[i].count > 10 {
			c.mmu.Log.Infof("%16s\t %d\t %s", results[i].instruction, results[i].count, results[i].time.String())
		}
		if results[i+256].count > 10 {
			c.mmu.Log.Infof("%16s\t %d\t %s", results[i].instruction, results[i].count, results[i].time.String())
		}
	}
}

func (c *CPU) executeInterrupt() {
	if c.ime {
		// save the high byte of the PC
		c.SP--
		c.writeByte(c.SP, uint8(c.PC>>8))

		vector := c.irq.Vector()

		// gameshark is applied on vblank interrupt
		if vector == 0x40 {
			// handle game shark TODO (emulate CPU time stolen by GameShark)
			if c.mmu.GameShark != nil {
				for _, code := range c.mmu.GameShark.Codes {
					if code.Enabled {
						c.mmu.Write(code.Address, code.NewData)
					}
				}
			}
		}

		// save the low byte of the PC
		c.SP--
		c.writeByte(c.SP, uint8(c.PC&0xFF))

		// jump to the interrupt vector and disable IME
		c.PC = vector
		c.ime = false

		// tick 12 times
		c.s.Tick(4)
		c.s.Tick(4)
		c.s.Tick(4)
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

package cpu

import (
	"github.com/thelolagemann/go-gameboy/internal/apu"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"github.com/thelolagemann/go-gameboy/internal/scheduler"
	"github.com/thelolagemann/go-gameboy/internal/serial"
	"github.com/thelolagemann/go-gameboy/internal/timer"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"sort"
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
	Debug           bool
	DebugBreakpoint bool

	ime      bool
	sysClock uint16

	doubleSpeed  bool
	perTickCount uint16

	mmu *mmu.MMU
	irq *interrupts.Service

	tickFuncs [16]func()
	tickFunc  func()

	mode mode

	registerSlice [8]*uint8

	instructions   [256]func(cpu *CPU)
	instructionsCB [256]func(cpu *CPU)

	cartFixedBank [0x4000]byte
	isGBC         bool
	isMBC1        bool
	hasFrame      bool
	sound         *apu.APU
	scheduler     *scheduler.Scheduler
}

func shouldTickPPU(number uint16) bool {
	return true
	switch {
	case number == 4:
		return true
	case number == 80:
		return true
	case number == 84:
		return true
	case number >= 168 && number <= 180:
		return true
	case number >= 192 && number <= 200:
		return true
	case number == 456:
		return true
	default:
		// fmt.Println("fallthrough", number)
		return false
	}
	// 2   = 0b0000_0010
	// 42  = 0b0010_1010
	// 84  = 0b0101_0100
	// 86  = 0b0101_0110
	// 88  = 0b0101_1000
	// 90  = 0b0101_1010
	// 96  = 0b0110_0000
	// 98  = 0b0110_0010
	// 100 = 0b0110_0100
	// 228 = 0b1110_0100
}

// NewCPU creates a new CPU instance with the given MMU.
// The MMU is used to read and write to the memory.
func NewCPU(mmu *mmu.MMU, irq *interrupts.Service, timerCtl *timer.Controller, video *ppu.PPU, sound *apu.APU, serialCtl *serial.Controller, sched *scheduler.Scheduler) *CPU {
	c := &CPU{
		Registers:    Registers{},
		mmu:          mmu,
		irq:          irq,
		perTickCount: 4,
		sysClock:     0xABCC,
		isMBC1:       mmu.IsMBC1,
		isGBC:        mmu.IsGBC(),
		sound:        sound,
		scheduler:    sched,
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

	types.RegisterHardware(
		types.DIV,
		func(v uint8) {
			c.sysClock = 0 // any write to DIV resets it
		},
		func() uint8 {
			// return bits 6-13 of divider register
			return uint8(c.sysClock >> 8) // TODO actually return bits 6-13
		},
		types.WithSet(func(v interface{}) {
			c.sysClock = v.(uint16)
		}),
	)

	timerT := func() {
		timerCtl.TickM(c.sysClock)
	}
	serialT := func() {
		serialCtl.TickM(c.sysClock)
	}
	timerSerial := func() {
		timerCtl.TickM(c.sysClock)
		serialCtl.TickM(c.sysClock)
	}
	videoT := func() {
		video.Dots += c.perTickCount
		video.CurrentCycle += uint64(c.perTickCount)

		if shouldTickPPU(video.Dots) {
			video.Tick()
		}
	}
	serialVideo := func() {
		serialCtl.TickM(c.sysClock)

		video.Dots += c.perTickCount
		video.CurrentCycle += uint64(c.perTickCount)

		if shouldTickPPU(video.Dots) {
			video.Tick()
		}
	}
	timerVideo := func() {
		timerCtl.TickM(c.sysClock)

		video.Dots += c.perTickCount
		video.CurrentCycle += uint64(c.perTickCount)

		if shouldTickPPU(video.Dots) {
			video.Tick()
		}
	}
	timerSerialVideo := func() {
		timerCtl.TickM(c.sysClock)
		serialCtl.TickM(c.sysClock)

		video.Dots += c.perTickCount
		video.CurrentCycle += uint64(c.perTickCount)

		if shouldTickPPU(video.Dots) {
			video.Tick()
		}
	}
	dma := func() {
		video.DMA.TickM()
	}
	timerDma := func() {
		video.DMA.TickM()
		timerCtl.TickM(c.sysClock)

	}
	serialDma := func() {
		video.DMA.TickM()

		serialCtl.TickM(c.sysClock)
	}
	timerSerialDma := func() {
		video.DMA.TickM()

		timerCtl.TickM(c.sysClock)

		serialCtl.TickM(c.sysClock)
	}
	videoDma := func() {
		video.DMA.TickM()

		video.Dots += c.perTickCount
		video.CurrentCycle += uint64(c.perTickCount)

		if shouldTickPPU(video.Dots) {
			video.Tick()
		}
	}
	timerDmaVideo := func() {
		video.DMA.TickM()
		timerCtl.TickM(c.sysClock)

		video.Dots += c.perTickCount
		video.CurrentCycle += uint64(c.perTickCount)

		if shouldTickPPU(video.Dots) {
			video.Tick()
		}
	}
	serialDmaVideo := func() {
		video.DMA.TickM()

		serialCtl.TickM(c.sysClock)

		video.Dots += c.perTickCount
		video.CurrentCycle += uint64(c.perTickCount)

		if shouldTickPPU(video.Dots) {
			video.Tick()
		}
	}
	timerSerialDmaVideo := func() {
		video.DMA.TickM()

		timerCtl.TickM(c.sysClock)
		serialCtl.TickM(c.sysClock)

		video.Dots += c.perTickCount
		video.CurrentCycle += uint64(c.perTickCount)

		if shouldTickPPU(video.Dots) {
			video.Tick()
		}
	}

	c.tickFuncs = [16]func(){
		0b0000_0000: func() {},
		0b0000_0001: timerT,
		0b0000_0010: serialT,
		0b0000_0011: timerSerial,
		0b0000_0100: videoT,
		0b0000_0101: timerVideo,
		0b0000_0110: serialVideo,
		0b0000_0111: timerSerialVideo,
		0b0000_1000: dma,
		0b0000_1001: timerDma,
		0b0000_1010: serialDma,
		0b0000_1011: timerSerialDma,
		0b0000_1100: videoDma,
		0b0000_1101: timerDmaVideo,
		0b0000_1110: serialDmaVideo,
		0b0000_1111: timerSerialDmaVideo,
	}
	c.tickFunc = c.tickFuncs[0]

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

func (c *CPU) tickCycle() {
	// tick the components
	c.tickFunc()

	// tick the internal clock
	c.sysClock += 4

	// tick the sound
	c.sound.TickM()

	// tick the scheduler
	if c.doubleSpeed {
		c.scheduler.Tick(2)
	} else {
		c.scheduler.Tick(4)
	}

	// handle any scheduled events
	for {
		// check if we have a scheduled event at this cycle
		if c.scheduler.Next() > c.scheduler.Cycle() {
			break
		}

		// execute the event
		c.scheduler.DoEvent()
	}
}

func (c *CPU) Frame() {
	for !c.hasFrame {
		if c.isGBC && c.mmu.HDMA.Copying {
			c.hdmaTick4()
			continue
		}
		if c.mode == ModeNormal {
			c.step()
		} else {
			c.stepSpecial()
		}
	}
	c.hasFrame = false
}

// step the CPU by one frame and returns the
// number of ticks that have been executed.
func (c *CPU) step() {
	// execute step normally
	c.instructions[c.readInstruction()](c)

	// did we get an interrupt?
	if c.ime && c.irq.Flag&c.irq.Enable != 0 {
		c.executeInterrupt()
	}
}

func (c *CPU) stepSpecial() {
	reqInt := false
	delayHalt := false
	// execute step based on mode
	switch c.mode {
	case ModeEnableIME:
		// Enabling IME, and set mode to normal
		c.ime = true
		c.mode = ModeNormal

		// read the next instruction
		instr := c.readInstruction()

		// handle ei_delay_halt (see https://github.com/LIJI32/SameSuite/blob/master/interrupt/ei_delay_halt.asm)
		if instr == 0x76 {
			// if an EI instruction is directly succeeded by a HALT instruction,
			// and there is a pending interrupt, the interrupt will be serviced
			// first, before the interrupt returns control to the HALT instruction,
			// effectively delaying the execution of HALT by one instruction.
			if c.irq.HasInterrupts() {
				delayHalt = true
			}
		}

		// execute the instruction
		c.instructions[instr](c)

		// check for interrupts
		reqInt = c.ime && c.irq.Flag&c.irq.Enable > 0
	case ModeHalt, ModeStop:
		// in stop, halt mode, the CPU ticks 4 times, but does not execute any instructions
		c.tickCycle()

		// check for interrupts, in stop mode the IME is ignored, this
		// is so that the CPU can be woken up by an interrupt while in halt, stop mode
		reqInt = c.irq.HasInterrupts()
	case ModeHaltBug:
		instr := c.readInstruction()
		c.PC--
		c.instructions[instr](c)
		c.mode = ModeNormal
		reqInt = c.ime && c.irq.Flag&c.irq.Enable > 0
	case ModeHaltDI:
		c.tickCycle()

		// check for interrupts
		if c.irq.HasInterrupts() {
			c.mode = ModeNormal
		}
	}

	if delayHalt {
		c.PC--
	}
	// did we get an interrupt?
	if reqInt {
		c.executeInterrupt()
	}
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
	value := c.readByte(c.PC)
	c.PC++
	return value
}

// readOperand reads the next operand from memory. The same as
// readInstruction, but will allow future optimizations.
func (c *CPU) readOperand() uint8 {
	value := c.readByte(c.PC)
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
	if c.mmu.BootROM != nil && !c.mmu.IsBootROMDone() {
		if addr < 0x100 {
			return c.mmu.BootROM.Read(addr)
		}
		if addr >= 0x200 && addr < 0x900 {
			return c.mmu.BootROM.Read(addr)
		}
	}
	if addr < 0x4000 && !c.isMBC1 {
		return c.cartFixedBank[addr]
	}

	return c.mmu.Read(addr)
}

// writeByte writes the given value to the given address.
func (c *CPU) writeByte(addr uint16, val uint8) {
	c.tickCycle()
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

		// save the low byte of the PC
		c.SP--
		c.writeByte(c.SP, uint8(c.PC&0xFF))

		// jump to the interrupt vector and disable IME
		c.PC = vector
		c.ime = false

		// tick 12 times
		c.tickCycle()
		c.tickCycle()
		c.tickCycle()
	}

	// set the mode to normal
	c.mode = ModeNormal
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
	s.Write8(c.mode)
	s.WriteBool(c.doubleSpeed)
	c.irq.Save(s)
}

func (c *CPU) SetTickKey(key uint8) {
	c.tickFunc = c.tickFuncs[key]
}

func (c *CPU) HasFrame() {
	c.hasFrame = true
}

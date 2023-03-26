package cpu

import (
	"github.com/thelolagemann/go-gameboy/internal/apu"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
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
	perTickCount uint8

	mmu *mmu.MMU
	irq *interrupts.Service

	tickFuncs [16]func()
	tickFunc  func()

	mode mode

	registerSlice [8]*uint8

	instructions   [256]func(cpu *CPU)
	instructionsCB [256]func(cpu *CPU)

	cartFixedBank [0x4000]byte
	isMBC1        bool
	hasFrame      bool
}

func shouldTickPPU(number uint16) bool {
	return true
	switch number {
	case 4, 80, 84, 168, 172, 176, 180, 196, 200, 204, 456:
		return true
	default:
		return false
	}
}

// NewCPU creates a new CPU instance with the given MMU.
// The MMU is used to read and write to the memory.
func NewCPU(mmu *mmu.MMU, irq *interrupts.Service, timerCtl *timer.Controller, video *ppu.PPU, sound *apu.APU, serialCtl *serial.Controller) *CPU {
	c := &CPU{
		Registers:    Registers{},
		mmu:          mmu,
		irq:          irq,
		perTickCount: 4,
		sysClock:     0xABCC,
		isMBC1:       mmu.IsMBC1,
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
		video.Dots += uint16(c.perTickCount)
		if shouldTickPPU(video.Dots) {
			c.hasFrame = video.Tick()
		}
	}
	serialVideo := func() {
		serialCtl.TickM(c.sysClock)

		video.Dots += uint16(c.perTickCount)
		if shouldTickPPU(video.Dots) {
			c.hasFrame = video.Tick()
		}
	}
	timerVideo := func() {
		timerCtl.TickM(c.sysClock)

		video.Dots += uint16(c.perTickCount)
		if shouldTickPPU(video.Dots) {
			c.hasFrame = video.Tick()
		}
	}
	timerSerialVideo := func() {
		timerCtl.TickM(c.sysClock)
		serialCtl.TickM(c.sysClock)

		video.Dots += uint16(c.perTickCount)
		if shouldTickPPU(video.Dots) {
			c.hasFrame = video.Tick()
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

		video.Dots += uint16(c.perTickCount)
		if shouldTickPPU(video.Dots) {
			c.hasFrame = video.Tick()
		}
	}
	timerDmaVideo := func() {
		video.DMA.TickM()
		timerCtl.TickM(c.sysClock)

		video.Dots += uint16(c.perTickCount)
		if shouldTickPPU(video.Dots) {
			c.hasFrame = video.Tick()
		}
	}
	serialDmaVideo := func() {
		video.DMA.TickM()

		serialCtl.TickM(c.sysClock)

		video.Dots += uint16(c.perTickCount)
		if shouldTickPPU(video.Dots) {
			c.hasFrame = video.Tick()
		}
	}
	timerSerialDmaVideo := func() {
		video.DMA.TickM()

		timerCtl.TickM(c.sysClock)
		serialCtl.TickM(c.sysClock)

		video.Dots += uint16(c.perTickCount)
		if shouldTickPPU(video.Dots) {
			c.hasFrame = video.Tick()
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
	c.tickFunc()
	c.sysClock += 4
}

func (c *CPU) Frame() {

	for !c.hasFrame {
		if c.mmu.HDMA != nil && c.mmu.HDMA.Copying {
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
	// execute step based on mode
	switch c.mode {
	case ModeEnableIME:
		// Enabling IME, and set mode to normal
		c.ime = true
		c.mode = ModeNormal

		// run one instruction
		c.instructions[c.readInstruction()](c)

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

	// is it possible to read from the cpu and avoid the call to the mmu?
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

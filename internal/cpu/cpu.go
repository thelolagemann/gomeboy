package cpu

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/thelolagemann/go-gameboy/pkg/utils"
)

const (
	// ClockSpeed is the clock speed of the CPU.
	ClockSpeed = 4194304
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

	peripherals []types.Component
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
func NewCPU(mmu *mmu.MMU, irq *interrupts.Service) *CPU {
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
			L: 0x7c,
		},
		mmu:     mmu,
		Speed:   1,
		stopped: false,
		Halted:  false,
		irq:     irq,
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

// Step executes the next instruction in the CPU and
// returns the number of cycles it took to execute.
func (c *CPU) Step() uint {
	// advance peripherals
	for _, p := range c.peripherals {
		p.Step() // 4
	}

	var cycles uint
	var cyclesCPU uint

	if c.Halted {
		cyclesCPU = 1
	} else {
		// fetch opcode
		opcode := c.fetch()
		var instruction Instructor

		// if 16-bit instruction
		if opcode == 0xCB {
			opcode = c.fetch()
			instruction = InstructionSetCB[opcode].Instruction()
			if instruction.Name() == "" {
				panic(fmt.Sprintf("instruction not found: 0xCB%02X", opcode))
			}
		} else {
			instruction = InstructionSet[opcode]
		}

		if instruction == nil {
			panic(fmt.Sprintf("instruction not found: 0x%02X", opcode))
		}

		// get operands
		operands := make([]uint8, instruction.Length()-1)
		for i := uint8(0); i < instruction.Length()-1; i++ {
			operands[i] = c.fetch()
		}
		if instruction.Name() != "NOP" && c.PC > 0x0010 {
			/*time.Sleep(100 * time.Millisecond)
			if len(operands) == 1 {
				c.mmu.Log.Debugf("cpu\t 0x%04X: %s 0x%02X", c.PC-uint16(instruction.Length()), instruction.Name(), operands[0])
			} else if len(operands) == 2 {
				c.mmu.Log.Debugf("cpu\t 0x%04X: %s 0x%02X%02X", c.PC-uint16(instruction.Length()), instruction.Name(), operands[1], operands[0])
			} else {
				c.mmu.Log.Debugf("cpu\t 0x%04X: %s", c.PC-uint16(instruction.Length()), instruction.Name())
			}
			c.mmu.Log.Debugf("reg\t A: %v, B: %v, C: %v, D: %v, E: %v, F: %v, H: %v, L: %v, SP: %v, PC: %v, opcode: 0x%02X", c.A, c.B, c.C, c.D, c.E, c.F, c.H, c.L, c.SP, c.PC, opcode)*/
		}
		instruction.Execute(c, operands)
		cyclesCPU = uint(instruction.Cycles())

		// handle debug
		if c.Debug {
			if instruction.Name() == "LD B, B" {
				c.DebugBreakpoint = true
			}
		}
	}
	cycles = cyclesCPU
	cycles += c.DoInterrupts()

	return cycles
}

// DoInterrupts handles all the interrupts.
func (c *CPU) DoInterrupts() uint {
	// check if interrupts are enabling (EI is delayed by 1 instruction)
	if c.irq.Enabling {
		c.irq.Enabling = false
		c.irq.IME = true
		return 0
	}

	// if not halted and IME is disabled, return
	if !c.Halted && !c.irq.IME {
		return 0
	}

	for i := uint8(0); i < 5; i++ {
		if utils.Test(c.irq.Flag, i) && utils.Test(c.irq.Enable, i) {
			if c.serviceInterrupt(i) {
				return 5
			}
		}
	}

	return 0
}

// serviceInterrupt handles the given interrupt.
func (c *CPU) serviceInterrupt(interrupt uint8) bool {
	// if halted without IME enabled, just clear the halt flag
	// do not jump or reset IF
	if c.Halted && !c.irq.IME {
		c.Halted = false
		return false
	}
	c.irq.IME = false
	c.Halted = false
	c.irq.Flag = utils.Reset(c.irq.Flag, interrupt)

	// save the current execution address by pushing it to the stack
	c.PushStack(c.PC)

	// jump to the interrupt handler
	switch interrupt {
	case 0:
		c.PC = 0x0040
	case 1:
		c.PC = 0x0048
	case 2:
		c.PC = 0x0050
	case 3:
		c.PC = 0x0058
	case 4:
		c.PC = 0x0060
	default:
		panic("illegal interrupt")
	}

	return true
}

// fetch returns the next byte in memory and increments the PC.
func (c *CPU) fetch() uint8 {
	opcode := c.mmu.Read(c.PC)
	c.PC++
	return opcode
}

// rst resets the CPU.
func (c *CPU) rst(v uint8) {
	c.push(uint8(c.PC >> 8))
	c.push(uint8(c.PC & 0xFF))
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

// push pushes a value to the stack.
func (c *CPU) push(value uint8) {
	c.SP--
	c.mmu.Write(c.SP, value)
}

// pop pops a value from the stack.
func (c *CPU) pop() uint8 {
	value := c.mmu.Read(c.SP)
	c.SP++
	return value
}

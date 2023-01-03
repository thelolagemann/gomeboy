package cpu

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
)

type CPU struct {
	PC uint16
	SP uint16
	Registers
	DebugPause bool

	mmu *mmu.MMU

	stopped           bool
	halted            bool
	interruptsEnabled bool

	usedPC map[uint16]bool
}

func (c *CPU) PopStack() uint16 {
	value := c.mmu.Read16(c.SP)
	c.SP += 2
	return value
}

func (c *CPU) PushStack(pc uint16) {
	c.SP -= 2
	c.mmu.Write16(c.SP, pc)
}

// NewCPU creates a new CPU instance with the given MMU.
// The MMU is used to read and write to the memory.
func NewCPU(mmu *mmu.MMU) *CPU {
	c := &CPU{
		PC: 0,
		SP: 0,
		Registers: Registers{
			A: 0x00,
			B: 0x00,
			C: 0x00,
			D: 0xFF,
			E: 0x56,
			F: 0x80,
			H: 0x00,
			L: 0x00,
		},
		mmu:     mmu,
		stopped: false,
		halted:  false,
		usedPC:  map[uint16]bool{},
	}
	c.BC = &RegisterPair{&c.B, &c.C}
	c.DE = &RegisterPair{&c.D, &c.E}
	c.HL = &RegisterPair{&c.H, &c.L}
	c.AF = &RegisterPair{&c.A, &c.F}
	c.generateCBInstructionSet()
	return c
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
func (c *CPU) Step() uint8 {
	// handle interrupts
	if c.halted {
		return 0x01
	}

	// fetch opcode
	opcode := c.fetch()
	var instruction Instruction

	// if 16-bit instruction
	if opcode == 0xCB {
		opcode = c.fetch()
		instruction = InstructionSetCB[opcode]
		if instruction.Name == "" {
			panic(fmt.Sprintf("instruction not found: 0xCB%02X", opcode))
		}
	} else {
		instruction = InstructionSet[opcode]
	}

	if instruction.Name == "" {
		panic(fmt.Sprintf("instruction not found: 0x%02X", opcode))
	}

	// get operands
	operands := make([]uint8, instruction.Length-1)
	for i := uint8(0); i < instruction.Length-1; i++ {
		operands[i] = c.fetch()
	}
	if instruction.Name != "NOP" {
		/* time.Sleep(1 * time.Millisecond)
		if len(operands) == 1 {
			c.mmu.Bus.Log().Debugf("cpu\t 0x%04X: %s 0x%02X", c.PC-uint16(instruction.Length), instruction.Name, operands[0])
		} else if len(operands) == 2 {
			c.mmu.Bus.Log().Debugf("cpu\t 0x%04X: %s 0x%02X%02X", c.PC-uint16(instruction.Length), instruction.Name, operands[1], operands[0])
		} else {
			c.mmu.Bus.Log().Debugf("cpu\t 0x%04X: %s", c.PC-uint16(instruction.Length), instruction.Name)
		}
		c.mmu.Bus.Log().Debugf("reg\t A: %v, B: %v, C: %v, D: %v, E: %v, F: %v, H: %v, L: %v, SP: %v, PC: %v, opcode: 0x%02X", c.A, c.B, c.C, c.D, c.E, c.F, c.H, c.L, c.SP, c.PC, opcode)*/
	}
	// execute instruction
	instruction.Execute(c, operands)
	if opcode == 0x40 {
		c.DebugPause = true
	}
	c.usedPC[c.PC] = true
	return instruction.Cycles
}

func (c *CPU) fetch() uint8 {
	opcode := c.mmu.Read(c.PC)
	c.PC++
	return opcode
}

// halt the CPU until an interrupt occurs. The CPU will
// not execute any instructions until then.
//
//	HALT
func (c *CPU) halt() {
	c.halted = true
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

func (c *CPU) push16(value uint16) {
	c.push(uint8(value >> 8))
	c.push(uint8(value & 0xFF))
}

// pop pops a value from the stack.
func (c *CPU) pop() uint8 {
	value := c.mmu.Read(c.SP)
	c.SP++
	return value
}

func (c *CPU) pop16() uint16 {
	low := c.pop()
	high := c.pop()
	return uint16(high)<<8 | uint16(low)
}

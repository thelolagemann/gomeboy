package cpu

import (
	"fmt"
)

type Instruction struct {
	name string
	fn   func(*CPU)
}

// Execute executes the instruction
func (i Instruction) Execute(cpu *CPU) {
	i.fn(cpu)
}

// Name returns the name of the instruction
func (i Instruction) Name() string {
	return i.name
}

// DefineInstruction is similar to NewInstruction, but it defines the instruction in
// the InstructionSet, with the provided opcode
func DefineInstruction(opcode uint8, name string, fn func(*CPU)) {
	instruction := Instruction{
		name: name,
		fn:   fn,
	}

	InstructionSet[opcode] = instruction
}

// Instructor is an interface that can be implemented by an instruction
type Instructor interface {
	Execute(cpu *CPU)

	// Name returns the name of the instruction
	Name() string
}

func init() {
	DefineInstruction(0x00, "NOP", func(c *CPU) {})
	DefineInstruction(0x10, "STOP", func(c *CPU) { c.mode = ModeStop })
	DefineInstruction(0x27, "DAA", func(cpu *CPU) {
		if !cpu.isFlagSet(FlagSubtract) {
			if cpu.isFlagSet(FlagCarry) || cpu.A > 0x99 {
				cpu.A += 0x60
				cpu.setFlag(FlagCarry)
			}
			if cpu.isFlagSet(FlagHalfCarry) || cpu.A&0xF > 0x9 {
				cpu.A += 0x06
				cpu.clearFlag(FlagHalfCarry)
			}
		} else if cpu.isFlagSet(FlagCarry) && cpu.isFlagSet(FlagHalfCarry) {
			cpu.A += 0x9a
			cpu.clearFlag(FlagHalfCarry)
		} else if cpu.isFlagSet(FlagCarry) {
			cpu.A += 0xa0
		} else if cpu.isFlagSet(FlagHalfCarry) {
			cpu.A += 0xfa
			cpu.clearFlag(FlagHalfCarry)
		}
		cpu.shouldZeroFlag(cpu.A)
	})
	DefineInstruction(0x2F, "CPL", func(cpu *CPU) {
		cpu.A = 0xFF ^ cpu.A
		cpu.setFlag(FlagSubtract)
		cpu.setFlag(FlagHalfCarry)
	})
	DefineInstruction(0x37, "SCF", func(cpu *CPU) {
		cpu.setFlag(FlagCarry)
		cpu.clearFlag(FlagSubtract)
		cpu.clearFlag(FlagHalfCarry)
	})
	DefineInstruction(0x3F, "CCF", func(cpu *CPU) {
		if cpu.isFlagSet(FlagCarry) {
			cpu.clearFlag(FlagCarry)
		} else {
			cpu.setFlag(FlagCarry)
		}
		cpu.clearFlag(FlagSubtract)
		cpu.clearFlag(FlagHalfCarry)
	})
	DefineInstruction(0x76, "HALT", func(c *CPU) {
		if c.irq.IME {
			c.mode = ModeHalt
		} else {
			if c.irq.Flag&c.irq.Enable != 0 {
				c.mode = ModeHaltBug
			} else {
				c.mode = ModeHaltDI
			}
		}
	})
	DefineInstruction(0xF3, "DI", func(c *CPU) { c.irq.IME = false })
	DefineInstruction(0xFB, "EI", func(c *CPU) { c.mode = ModeEnableIME })

	for _, opcode := range disallowedOpcodes {
		DefineInstruction(opcode, "disallowed", disallowedOpcode)
	}
}

var disallowedOpcodes = []uint8{
	0xCB, 0xD3, 0xDB, 0xDD, 0xE3, 0xE4, 0xEB, 0xEC, 0xED, 0xF4, 0xFC, 0xFD,
}

var InstructionSet = map[uint8]Instructor{}

func disallowedOpcode(cpu *CPU) {
	panic(fmt.Sprintf("disallowed opcode %X", cpu.mmu.Read(cpu.PC)))
}

// FlagInstruction represents an instruction that
// affects a flag in the CPU's flags register
type FlagInstruction struct {
	name   string
	length uint8
	cycles uint8

	// Flags is an InstructionFlags value that represents
	// the flags that are affected by the instruction
	Flags InstructionFlags

	// fn is the function that executes the instruction
	fn func(cpu *CPU, operands []byte)

	// fnResult is the function that executes the instruction
	// and returns the result of the operation
	fnResult func(cpu *CPU, operands []byte) uint8
}

func (fi FlagInstruction) Name() string {
	return fi.name
}

func (fi FlagInstruction) Length() uint8 {
	return fi.length
}

func (fi FlagInstruction) Cycles() uint8 {
	return fi.cycles
}

// FlagInstructionOpt is a function that configures a FlagInstruction
type FlagInstructionOpt func(*FlagInstruction)

// SetFlags sets the flags that are set by the instruction
func SetFlags(flags ...Flag) FlagInstructionOpt {
	return func(fi *FlagInstruction) {
		fi.Flags.Set = flags
	}
}

// ClearFlags sets the flags that are cleared by the instruction
func ClearFlags(flags ...Flag) FlagInstructionOpt {
	return func(fi *FlagInstruction) {
		fi.Flags.Reset = flags
	}
}

// OperationFlags configures the flags that are affected by the instruction
func OperationFlags(fn func(*CPU, []byte) uint8, flags ...Flag) FlagInstructionOpt {
	return func(fi *FlagInstruction) {
		fi.fnResult = fn
		fi.Flags.Operation = flags
	}
}

// NewFlagInstruction creates a new FlagInstruction
func NewFlagInstruction(name string, length uint8, cycles uint8, fn func(*CPU, []byte), opts ...FlagInstructionOpt) FlagInstruction {
	fi := FlagInstruction{
		name:   name,
		length: length,
		cycles: cycles,
		fn:     fn,
	}
	for _, opt := range opts {
		opt(&fi)
	}
	return fi
}

// Execute executes the instruction
func (fi FlagInstruction) Execute(cpu *CPU, operands []byte) {
	// configure the flags
	cpu.setFlags(fi.Flags.Set...)
	cpu.clearFlags(fi.Flags.Reset...)

	// determine if the instruction is a result instruction
	if fi.fnResult != nil {
		result := fi.fnResult(cpu, operands)

		// loop through the flags that are affected by the operation
		for _, flag := range fi.Flags.Operation {
			switch flag {
			case FlagZero:
				if result == 0 {
					cpu.setFlag(FlagZero)
				} else {
					cpu.clearFlag(FlagZero)
				}
			case FlagSubtract:
				cpu.setFlag(FlagSubtract)
			case FlagHalfCarry:
				if (cpu.Registers.A&0x0F)+(operands[0]&0x0F) > 0x0F {
					cpu.setFlag(FlagHalfCarry)
				} else {
					cpu.clearFlag(FlagHalfCarry)
				}
			case FlagCarry:
				if (cpu.Registers.A&0xFF)+(operands[0]&0xFF) > 0xFF {
					cpu.setFlag(FlagCarry)
				} else {
					cpu.clearFlag(FlagCarry)
				}
			}
		}
	} else {
		fi.fn(cpu, operands)
	}
}

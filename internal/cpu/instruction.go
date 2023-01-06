package cpu

import (
	"fmt"
)

type Instruction struct {
	name   string
	length uint8
	cycles uint8
	fn     func(cpu *CPU, operands []byte)
}

// Execute executes the instruction
func (i Instruction) Execute(cpu *CPU, operands []byte) {
	i.fn(cpu, operands)
}

// Name returns the name of the instruction
func (i Instruction) Name() string {
	return i.name
}

// Length returns the length of the instruction
func (i Instruction) Length() uint8 {
	return i.length
}

// Cycles returns the number of cycles the instruction takes
func (i Instruction) Cycles() uint8 {
	return i.cycles
}

// NewInstruction creates a new Instruction
func NewInstruction(name string, length uint8, cycles uint8, fn func(*CPU, []byte)) Instruction {
	return Instruction{
		name:   name,
		length: length,
		cycles: cycles,
		fn:     fn,
	}
}

// Instructor is an interface that can be implemented by an instruction
type Instructor interface {
	Execute(cpu *CPU, operands []byte)

	// Name returns the name of the instruction
	Name() string
	// Length returns the length of the instruction
	Length() uint8
	// Cycles returns the number of cycles the instruction takes
	Cycles() uint8
}

var InstructionSet = map[uint8]Instructor{
	// perform no operation
	0x00: NewInstruction("NOP", 1, 1, func(cpu *CPU, operands []uint8) {}),
	// stop the CPU until an interrupt occurs
	0x10: NewInstruction("STOP", 2, 1, func(cpu *CPU, operands []byte) { cpu.stopped = true }),
	// advances PC by 1, rotates Register A left
	0x27: NewInstruction("DAA", 1, 1, func(cpu *CPU, operands []byte) {
		a := uint16(cpu.A)
		if cpu.isFlagSet(FlagSubtract) {
			if cpu.isFlagSet(FlagHalfCarry) || a&0x0F > 9 {
				a += 0x06
			}
			if cpu.isFlagSet(FlagCarry) || a > 0x9F {
				a += 0x60
			}
		} else {
			if cpu.isFlagSet(FlagHalfCarry) {
				a = (a - 0x06) & 0xFF
			}
			if cpu.isFlagSet(FlagCarry) {
				a -= 0x60
			}
		}
		cpu.clearFlag(FlagHalfCarry)
		if a&0x100 == 0x100 {
			cpu.setFlag(FlagCarry)
		}
		a &= 0xFF
		cpu.shouldZeroFlag(uint8(a))
		cpu.A = uint8(a)
	}),
	// the contents of Register A are complemented (i.e. flip all bits)
	0x2F: NewInstruction("CPL", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.Registers.A = ^cpu.Registers.A
		cpu.setFlag(FlagSubtract)
		cpu.setFlag(FlagHalfCarry)
	}),
	// advances PC by 1, sets carry flag
	0x37: NewInstruction("SCF", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.setFlag(FlagCarry)
		cpu.clearFlag(FlagSubtract)
		cpu.clearFlag(FlagHalfCarry)
	}),
	// flips the carry flag
	0x3F: NewInstruction("CCF", 1, 1, func(cpu *CPU, operands []byte) {
		if cpu.isFlagSet(FlagCarry) {
			cpu.clearFlag(FlagCarry)
		} else {
			cpu.setFlag(FlagCarry)
		}
		cpu.clearFlag(FlagSubtract)
		cpu.clearFlag(FlagHalfCarry)
	}),
	// halt the CPU and LCD until an interrupt occurs
	0x76: NewInstruction("HALT", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.halt()
	}),
	// disable interrupts after the next instruction is executed
	0xF3: NewInstruction("DI", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.irq.IME = false
	}),
	// set the interrupt master enable flag and enable maskable interrupts
	0xFB: NewInstruction("EI", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.irq.IME = true
	}),
	// disallowed opcodes
	0xCB: NewInstruction("", 0, 0, disallowedOpcode),
	0xD3: NewInstruction("", 0, 0, disallowedOpcode),
	0xDB: NewInstruction("", 0, 0, disallowedOpcode),
	0xDD: NewInstruction("", 0, 0, disallowedOpcode),
	0xE3: NewInstruction("", 0, 0, disallowedOpcode),
	0xE4: NewInstruction("", 0, 0, disallowedOpcode),
	0xEB: NewInstruction("", 0, 0, disallowedOpcode),
	0xEC: NewInstruction("", 0, 0, disallowedOpcode),
	0xED: NewInstruction("", 0, 0, disallowedOpcode),
	0xF4: NewInstruction("", 0, 0, disallowedOpcode),
	0xFC: NewInstruction("", 0, 0, disallowedOpcode),
	0xFD: NewInstruction("", 0, 0, disallowedOpcode),
}

func disallowedOpcode(cpu *CPU, operands []byte) {
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

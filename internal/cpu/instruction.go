package cpu

import (
	"fmt"
)

type Instruction struct {
	name string
	fn   func(*CPU)
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

func DefineInstructionCB(opcode uint8, name string, fn func(*CPU)) {
	instruction := Instruction{
		name: name,
		fn:   fn,
	}

	InstructionSetCB[opcode] = instruction
}

func init() {
	DefineInstruction(0x00, "NOP", func(c *CPU) {})
	DefineInstruction(0x10, "STOP", func(c *CPU) {
		if c.mmu.IsGBC() {
			if c.mmu.Key()&0b0000_0001 == 1 {
				c.mmu.Log.Debugf("CGB STOP, key: %08b", c.mmu.Key())
				c.doubleSpeed = !c.doubleSpeed

				if c.mmu.Key()&0b1000_0000 == 1 {
					c.mmu.SetKey(0)
				} else {
					c.mmu.SetKey(0b1000_0000)
				}
			}

		} else {
			c.mode = ModeStop
		}
	})
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

var InstructionSet [256]Instruction

func disallowedOpcode(cpu *CPU) {
	panic(fmt.Sprintf("disallowed opcode %X", cpu.mmu.Read(cpu.PC)))
}

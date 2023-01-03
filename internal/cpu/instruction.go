package cpu

import (
	"encoding/binary"
	"fmt"
	"strconv"
)

type Instruction struct {
	Name    string
	Length  uint8
	Cycles  uint8
	Execute func(cpu *CPU, operands []byte)
}

var InstructionSet = [0x100]Instruction{
	// advances PC by 1, performs no other operations
	0x00: {"NOP", 1, 1, func(cpu *CPU, operands []uint8) {}},
	// load the 2 byte immediate value into Register pair BC
	0x01: {"LD BC, d16", 3, 3, func(cpu *CPU, operands []uint8) {
		cpu.loadRegister16(cpu.Registers.BC, binary.LittleEndian.Uint16(operands))
	}},
	// store the contents of Register A in the memory location specified by Register pair BC
	0x02: {"LD (BC), A", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToMemory(&cpu.Registers.A, cpu.Registers.BC.Uint16())
	}},
	// increment Register pair BC
	0x03: {"INC BC", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.incrementNN(cpu.Registers.BC)
	}},
	// increment Register B
	0x04: {"INC B", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.incrementN(&cpu.Registers.B)
	}},
	// decrement Register B
	0x05: {"DEC B", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.decrementN(&cpu.Registers.B)
	}},
	// loads the 1 byte immediate value into Register B
	0x06: {"LD B, d8", 2, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegister8(&cpu.Registers.B, operands[0])
	}},
	// rotates Register A left through the carry flag
	0x07: {"RLCA", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.rotateLeftAccumulator()
	}},
	// load SP into the address specified by the 2 byte immediate value
	0x08: {"LD (a16), SP", 3, 5, func(cpu *CPU, operands []byte) {
		cpu.mmu.Write16(binary.LittleEndian.Uint16(operands), cpu.SP)
	}},
	// add Register pair BC to Register pair HL
	0x09: {"ADD HL, BC", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.addHL(cpu.Registers.BC)
	}},
	// loads the contents of the memory location specified by Register pair BC into Register A
	0x0A: {"LD A, (BC)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadMemoryToRegister(&cpu.Registers.A, cpu.Registers.BC.Uint16())
	}},
	// decrement Register pair BC
	0x0B: {"DEC BC", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.decrementNN(cpu.Registers.BC)
	}},
	// increment Register C
	0x0C: {"INC C", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.incrementN(&cpu.Registers.C)
	}},
	// decrement Register C
	0x0D: {"DEC C", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.decrementN(&cpu.Registers.C)
	}},
	// loads the 1 byte immediate value into Register C
	0x0E: {"LD C, d8", 2, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegister8(&cpu.Registers.C, operands[0])
	}},
	// rotates Register A right through the carry flag
	0x0F: {"RRCA", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.rotateRightAccumulator()
	}},
	// advances PC by 2, STOPs the CPU and LCD until button pressed
	0x10: {"STOP 0", 2, 1, func(cpu *CPU, operands []byte) {
		cpu.stopped = true
	}},
	// loads the 2 byte immediate value into Register pair DE
	0x11: {"LD DE, d16", 3, 3, func(cpu *CPU, operands []byte) {
		cpu.loadRegister16(cpu.Registers.DE, binary.LittleEndian.Uint16(operands))
	}},
	// store the contents of Register A in the memory location specified by Register pair DE
	0x12: {"LD (DE), A", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToMemory(&cpu.Registers.A, cpu.Registers.DE.Uint16())
	}},
	// increment Register pair DE
	0x13: {"INC DE", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.incrementNN(cpu.Registers.DE)
	}},
	// increment Register D
	0x14: {"INC D", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.incrementN(&cpu.Registers.D)
	}},
	// decrement Register D
	0x15: {"DEC D", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.decrementN(&cpu.Registers.D)
	}},
	// loads the 1 byte immediate value into Register D
	0x16: {"LD D, d8", 2, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegister8(&cpu.Registers.D, operands[0])
	}},
	// rotates Register A left
	0x17: {"RLA", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.rotateLeftAccumulatorThroughCarry()
	}},
	// jumps s8 steps relative to the current PC
	0x18: {"JR r8", 2, 3, func(cpu *CPU, operands []byte) {
		cpu.jumpRelative(operands[0])
	}},
	// add Register pair DE to Register pair HL
	0x19: {"ADD HL, DE", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.addHL(cpu.Registers.DE)
	}},
	// loads the contents of the memory location specified by Register pair DE into Register A
	0x1A: {"LD A, (DE)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadMemoryToRegister(&cpu.Registers.A, cpu.Registers.DE.Uint16())
	}},
	// decrement Register pair DE
	0x1B: {"DEC DE", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.decrementNN(cpu.Registers.DE)
	}},
	// increment Register E
	0x1C: {"INC E", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.incrementN(&cpu.Registers.E)
	}},
	// decrement Register E
	0x1D: {"DEC E", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.decrementN(&cpu.Registers.E)
	}},
	// loads the 1 byte immediate value into Register E
	0x1E: {"LD E, d8", 2, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegister8(&cpu.Registers.E, operands[0])
	}},
	// rotates Register A right
	0x1F: {"RRA", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.rotateRightAccumulatorThroughCarry()
	}},
	// jumps to the relative address specified by the 1 byte immediate value
	0x20: {"JR NZ, r8", 2, 3, func(cpu *CPU, operands []byte) {
		cpu.jumpRelativeConditional(!cpu.isFlagSet(FlagZero), operands[0])
	}},
	// loads the 2 byte immediate value into Register pair HL
	0x21: {"LD HL, d16", 3, 3, func(cpu *CPU, operands []byte) {
		cpu.loadRegister16(cpu.Registers.HL, binary.LittleEndian.Uint16(operands))
	}},
	// store the contents of Register A in the memory location specified by Register pair HL and increment HL
	0x22: {"LD (HL+), A", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToMemory(&cpu.Registers.A, cpu.Registers.HL.Uint16())
		cpu.incrementNN(cpu.Registers.HL)
	}},
	// increment Register pair HL
	0x23: {"INC HL", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.incrementNN(cpu.Registers.HL)
	}},
	// increment Register H
	0x24: {"INC H", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.incrementN(&cpu.Registers.H)
	}},
	// decrement Register H
	0x25: {"DEC H", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.decrementN(&cpu.Registers.H)
	}},
	// loads the 1 byte immediate value into Register H
	0x26: {"LD H, d8", 2, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegister8(&cpu.Registers.H, operands[0])
	}},
	// advances PC by 1, rotates Register A left
	0x27: {"DAA", 1, 1, func(cpu *CPU, operands []byte) {
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
	}},
	// jumps to the address specified by the 1 byte immediate value the Z flag is set
	0x28: {"JR Z, r8", 2, 3, func(cpu *CPU, operands []byte) {
		cpu.jumpRelativeConditional(cpu.isFlagSet(FlagZero), operands[0])
	}},
	// add Register pair HL to Register pair HL
	0x29: {"ADD HL, HL", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.addHL(cpu.Registers.HL)
	}},
	// loads the contents of the memory location specified by Register pair HL into Register A and increment HL
	0x2A: {"LD A, (HL+)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadMemoryToRegister(&cpu.Registers.A, cpu.Registers.HL.Uint16())
		cpu.incrementNN(cpu.Registers.HL)
	}},
	// decrement Register pair HL
	0x2B: {"DEC HL", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.decrementNN(cpu.Registers.HL)
	}},
	// increment Register L
	0x2C: {"INC L", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.incrementN(&cpu.Registers.L)
	}},
	// decrement Register L
	0x2D: {"DEC L", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.decrementN(&cpu.Registers.L)
	}},
	// loads the 1 byte immediate value into Register L
	0x2E: {"LD L, d8", 2, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegister8(&cpu.Registers.L, operands[0])
	}},
	// the contents of Register A are complemented (i.e. flip all bits)
	0x2F: {"CPL", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.Registers.A = ^cpu.Registers.A
		cpu.setFlag(FlagSubtract)
		cpu.setFlag(FlagHalfCarry)
	}},
	// jumps to the address specified by the 1 byte immediate value if the C flag is not set
	0x30: {"JR NC, r8", 2, 3, func(cpu *CPU, operands []byte) {
		cpu.jumpRelativeConditional(!cpu.isFlagSet(FlagCarry), operands[0])
	}},
	// loads the 2 byte immediate value into Register pair SP
	0x31: {"LD SP, d16", 3, 3, func(cpu *CPU, operands []byte) {
		cpu.SP = binary.LittleEndian.Uint16(operands)
	}},
	// loads the contents of Register A into the memory location specified by Register pair HL and decrement HL
	0x32: {"LD (HL-), A", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToMemory(&cpu.Registers.A, cpu.Registers.HL.Uint16())
		cpu.decrementNN(cpu.Registers.HL)
	}},
	// increment Register pair SP
	0x33: {"INC SP", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.SP++
	}},
	// increment the contents of the memory location specified by Register pair HL
	0x34: {"INC (HL)", 1, 3, func(cpu *CPU, operands []byte) {
		cpu.mmu.Write(cpu.Registers.HL.Uint16(), cpu.increment(cpu.mmu.Read(cpu.Registers.HL.Uint16())))
	}},
	// decrement the contents of the memory location specified by Register pair HL
	0x35: {"DEC (HL)", 1, 3, func(cpu *CPU, operands []byte) {
		cpu.mmu.Write(cpu.Registers.HL.Uint16(), cpu.decrement(cpu.mmu.Read(cpu.Registers.HL.Uint16())))
	}},
	// loads the 1 byte immediate value into the memory location specified by Register pair HL
	0x36: {"LD (HL), d8", 2, 3, func(cpu *CPU, operands []byte) {
		cpu.mmu.Write(cpu.Registers.HL.Uint16(), operands[0])
	}},
	// advances PC by 1, sets carry flag
	0x37: {"SCF", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.setFlag(FlagCarry)
		cpu.clearFlag(FlagSubtract)
		cpu.clearFlag(FlagHalfCarry)
	}},
	// jumps to the address specified by the 1 byte immediate value if the C flag is set
	0x38: {"JR C, r8", 2, 3, func(cpu *CPU, operands []byte) {
		cpu.jumpRelativeConditional(cpu.isFlagSet(FlagCarry), operands[0])
	}},
	// add Register pair SP to Register pair HL
	0x39: {"ADD HL, SP", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.HL.SetUint16(cpu.addUint16(cpu.HL.Uint16(), cpu.SP))
	}},
	// loads the contents of the memory location specified by Register pair HL into Register A and decrement HL
	0x3A: {"LD A, (HL-)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadMemoryToRegister(&cpu.Registers.A, cpu.Registers.HL.Uint16())
		cpu.decrementNN(cpu.Registers.HL)
	}},
	// decrement Register pair SP
	0x3B: {"DEC SP", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.SP--
	}},
	// increments Register A
	0x3C: {"INC A", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.incrementN(&cpu.Registers.A)
	}},
	// decrements Register A
	0x3D: {"DEC A", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.decrementN(&cpu.Registers.A)
	}},
	// loads the 1 byte immediate value into Register A
	0x3E: {"LD A, d8", 2, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegister8(&cpu.Registers.A, operands[0])
	}},
	// flips the carry flag
	0x3F: {"CCF", 1, 1, func(cpu *CPU, operands []byte) {
		if cpu.isFlagSet(FlagCarry) {
			cpu.clearFlag(FlagCarry)
		} else {
			cpu.setFlag(FlagCarry)
		}
		cpu.clearFlag(FlagSubtract)
		cpu.clearFlag(FlagHalfCarry)
	}},
	// loads the contents of Register B into Register B
	0x40: {"LD B, B", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.B, &cpu.Registers.B)
	}},
	// loads the contents of Register C into Register B
	0x41: {"LD B, C", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.B, &cpu.Registers.C)
	}},
	// loads the contents of Register D into Register B
	0x42: {"LD B, D", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.B, &cpu.Registers.D)
	}},
	// loads the contents of Register E into Register B
	0x43: {"LD B, E", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.B, &cpu.Registers.E)
	}},
	// loads the contents of Register H into Register B
	0x44: {"LD B, H", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.B, &cpu.Registers.H)
	}},
	// loads the contents of Register L into Register B
	0x45: {"LD B, L", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.B, &cpu.Registers.L)
	}},
	// loads the contents of the memory location specified by Register pair HL into Register B
	0x46: {"LD B, (HL)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadMemoryToRegister(&cpu.Registers.B, cpu.Registers.HL.Uint16())
	}},
	// loads the contents of Register A into Register B
	0x47: {"LD B, A", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.B, &cpu.Registers.A)
	}},
	// loads the contents of Register B into Register C
	0x48: {"LD C, B", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.C, &cpu.Registers.B)
	}},
	// loads the contents of Register C into Register C
	0x49: {"LD C, C", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.C, &cpu.Registers.C)
	}},
	// loads the contents of Register D into Register C
	0x4A: {"LD C, D", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.C, &cpu.Registers.D)
	}},
	// loads the contents of Register E into Register C
	0x4B: {"LD C, E", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.C, &cpu.Registers.E)
	}},
	// loads the contents of Register H into Register C
	0x4C: {"LD C, H", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.C, &cpu.Registers.H)
	}},
	// loads the contents of Register L into Register C
	0x4D: {"LD C, L", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.C, &cpu.Registers.L)
	}},
	// loads the contents of the memory location specified by Register pair HL into Register C
	0x4E: {"LD C, (HL)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadMemoryToRegister(&cpu.Registers.C, cpu.Registers.HL.Uint16())
	}},
	// loads the contents of Register A into Register C
	0x4F: {"LD C, A", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.C, &cpu.Registers.A)
	}},
	// loads the contents of Register B into Register D
	0x50: {"LD D, B", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.D, &cpu.Registers.B)
	}},
	// loads the contents of Register C into Register D
	0x51: {"LD D, C", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.D, &cpu.Registers.C)
	}},
	// loads the contents of Register D into Register D
	0x52: {"LD D, D", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.D, &cpu.Registers.D)
	}},
	// loads the contents of Register E into Register D
	0x53: {"LD D, E", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.D, &cpu.Registers.E)
	}},
	// loads the contents of Register H into Register D
	0x54: {"LD D, H", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.D, &cpu.Registers.H)
	}},
	// loads the contents of Register L into Register D
	0x55: {"LD D, L", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.D, &cpu.Registers.L)
	}},
	// loads the contents of the memory location specified by Register pair HL into Register D
	0x56: {"LD D, (HL)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadMemoryToRegister(&cpu.Registers.D, cpu.Registers.HL.Uint16())
	}},
	// loads the contents of Register A into Register D
	0x57: {"LD D, A", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.D, &cpu.Registers.A)
	}},
	// loads the contents of Register B into Register E
	0x58: {"LD E, B", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.E, &cpu.Registers.B)
	}},
	// loads the contents of Register C into Register E
	0x59: {"LD E, C", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.E, &cpu.Registers.C)
	}},
	// loads the contents of Register D into Register E
	0x5A: {"LD E, D", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.E, &cpu.Registers.D)
	}},
	// loads the contents of Register E into Register E
	0x5B: {"LD E, E", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.E, &cpu.Registers.E)
	}},
	// loads the contents of Register H into Register E
	0x5C: {"LD E, H", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.E, &cpu.Registers.H)
	}},
	// loads the contents of Register L into Register E
	0x5D: {"LD E, L", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.E, &cpu.Registers.L)
	}},
	// loads the contents of the memory location specified by Register pair HL into Register E
	0x5E: {"LD E, (HL)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadMemoryToRegister(&cpu.Registers.E, cpu.Registers.HL.Uint16())
	}},
	// loads the contents of Register A into Register E
	0x5F: {"LD E, A", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.E, &cpu.Registers.A)
	}},
	// loads the contents of Register B into Register H
	0x60: {"LD H, B", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.H, &cpu.Registers.B)
	}},
	// loads the contents of Register C into Register H
	0x61: {"LD H, C", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.H, &cpu.Registers.C)
	}},
	// loads the contents of Register D into Register H
	0x62: {"LD H, D", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.H, &cpu.Registers.D)
	}},
	// loads the contents of Register E into Register H
	0x63: {"LD H, E", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.H, &cpu.Registers.E)
	}},
	// loads the contents of Register H into Register H
	0x64: {"LD H, H", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.H, &cpu.Registers.H)
	}},
	// loads the contents of Register L into Register H
	0x65: {"LD H, L", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.H, &cpu.Registers.L)
	}},
	// loads the contents of the memory location specified by Register pair HL into Register H
	0x66: {"LD H, (HL)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadMemoryToRegister(&cpu.Registers.H, cpu.Registers.HL.Uint16())
	}},
	// loads the contents of Register A into Register H
	0x67: {"LD H, A", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.H, &cpu.Registers.A)
	}},
	// loads the contents of Register B into Register L
	0x68: {"LD L, B", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.L, &cpu.Registers.B)
	}},
	// loads the contents of Register C into Register L
	0x69: {"LD L, C", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.L, &cpu.Registers.C)
	}},
	// loads the contents of Register D into Register L
	0x6A: {"LD L, D", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.L, &cpu.Registers.D)
	}},
	// loads the contents of Register E into Register L
	0x6B: {"LD L, E", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.L, &cpu.Registers.E)
	}},
	// loads the contents of Register H into Register L
	0x6C: {"LD L, H", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.L, &cpu.Registers.H)
	}},
	// loads the contents of Register L into Register L
	0x6D: {"LD L, L", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.L, &cpu.Registers.L)
	}},
	// loads the contents of the memory location specified by Register pair HL into Register L
	0x6E: {"LD L, (HL)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadMemoryToRegister(&cpu.Registers.L, cpu.Registers.HL.Uint16())
	}},
	// loads the contents of Register A into Register L
	0x6F: {"LD L, A", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.L, &cpu.Registers.A)
	}},
	// loads the contents of Register B into the memory location specified by Register pair HL
	0x70: {"LD (HL), B", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToMemory(&cpu.Registers.B, cpu.Registers.HL.Uint16())
	}},
	// loads the contents of Register C into the memory location specified by Register pair HL
	0x71: {"LD (HL), C", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToMemory(&cpu.Registers.C, cpu.Registers.HL.Uint16())
	}},
	// loads the contents of Register D into the memory location specified by Register pair HL
	0x72: {"LD (HL), D", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToMemory(&cpu.Registers.D, cpu.Registers.HL.Uint16())
	}},
	// loads the contents of Register E into the memory location specified by Register pair HL
	0x73: {"LD (HL), E", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToMemory(&cpu.Registers.E, cpu.Registers.HL.Uint16())
	}},
	// loads the contents of Register H into the memory location specified by Register pair HL
	0x74: {"LD (HL), H", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToMemory(&cpu.Registers.H, cpu.Registers.HL.Uint16())
	}},
	// loads the contents of Register L into the memory location specified by Register pair HL
	0x75: {"LD (HL), L", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToMemory(&cpu.Registers.L, cpu.Registers.HL.Uint16())
	}},
	// halt the CPU and LCD until an interrupt occurs
	0x76: {"HALT", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.halt()
	}},
	// loads the contents of Register A into the memory location specified by Register pair HL
	0x77: {"LD (HL), A", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToMemory(&cpu.Registers.A, cpu.Registers.HL.Uint16())
	}},
	// loads the contents of Register B into Register A
	0x78: {"LD A, B", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.A, &cpu.Registers.B)
	}},
	// loads the contents of Register C into Register A
	0x79: {"LD A, C", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.A, &cpu.Registers.C)
	}},
	// loads the contents of Register D into Register A
	0x7A: {"LD A, D", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.A, &cpu.Registers.D)
	}},
	// loads the contents of Register E into Register A
	0x7B: {"LD A, E", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.A, &cpu.Registers.E)
	}},
	// loads the contents of Register H into Register A
	0x7C: {"LD A, H", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.A, &cpu.Registers.H)
	}},
	// loads the contents of Register L into Register A
	0x7D: {"LD A, L", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.A, &cpu.Registers.L)
	}},
	// loads the contents of the memory location specified by Register pair HL into Register A
	0x7E: {"LD A, (HL)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadMemoryToRegister(&cpu.Registers.A, cpu.Registers.HL.Uint16())
	}},
	// loads the contents of Register A into Register A
	0x7F: {"LD A, A", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.loadRegisterToRegister(&cpu.Registers.A, &cpu.Registers.A)
	}},
	// adds the contents of Register A and Register B and stores the result in Register A
	0x80: {"ADD A, B", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.addN(cpu.Registers.B)
	}},
	// adds the contents of Register A and Register C and stores the result in Register A
	0x81: {"ADD A, C", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.addN(cpu.Registers.C)
	}},
	// adds the contents of Register A and Register D and stores the result in Register A
	0x82: {"ADD A, D", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.addN(cpu.Registers.D)
	}},
	// adds the contents of Register A and Register E and stores the result in Register A
	0x83: {"ADD A, E", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.addN(cpu.Registers.E)
	}},
	// adds the contents of Register A and Register H and stores the result in Register A
	0x84: {"ADD A, H", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.addN(cpu.Registers.H)
	}},
	// adds the contents of Register A and Register L and stores the result in Register A
	0x85: {"ADD A, L", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.addN(cpu.Registers.L)
	}},
	// adds the contents of Register A and the memory location specified by Register pair HL and stores the result in Register A
	0x86: {"ADD A, (HL)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.addN(cpu.mmu.Read(cpu.Registers.HL.Uint16()))
	}},
	// adds the contents of Register A and Register A and stores the result in Register A
	0x87: {"ADD A, A", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.addN(cpu.Registers.A)
	}},
	// adds the contents of Register A and Register B and stores the result in Register A and sets the carry flag if there is a carry
	0x88: {"ADC A, B", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.addNCarry(cpu.Registers.B)
	}},
	// adds the contents of Register A and Register C and stores the result in Register A and sets the carry flag if there is a carry
	0x89: {"ADC A, C", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.addNCarry(cpu.Registers.C)
	}},
	// adds the contents of Register A and Register D and stores the result in Register A and sets the carry flag if there is a carry
	0x8A: {"ADC A, D", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.addNCarry(cpu.Registers.D)
	}},
	// adds the contents of Register A and Register E and stores the result in Register A and sets the carry flag if there is a carry
	0x8B: {"ADC A, E", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.addNCarry(cpu.Registers.E)
	}},
	// adds the contents of Register A and Register H and stores the result in Register A and sets the carry flag if there is a carry
	0x8C: {"ADC A, H", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.addNCarry(cpu.Registers.H)
	}},
	// adds the contents of Register A and Register L and stores the result in Register A and sets the carry flag if there is a carry
	0x8D: {"ADC A, L", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.addNCarry(cpu.Registers.L)
	}},
	// adds the contents of Register A and the memory location specified by Register pair HL and stores the result in Register A and sets the carry flag if there is a carry
	0x8E: {"ADC A, (HL)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.addNCarry(cpu.mmu.Read(cpu.Registers.HL.Uint16()))
	}},
	// adds the contents of Register A and Register A and stores the result in Register A and sets the carry flag if there is a carry
	0x8F: {"ADC A, A", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.addNCarry(cpu.Registers.A)
	}},
	// subtracts the contents of Register B from Register A and stores the result in Register A
	0x90: {"SUB B", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.subtractN(cpu.Registers.B)
	}},
	// subtracts the contents of Register C from Register A and stores the result in Register A
	0x91: {"SUB C", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.subtractN(cpu.Registers.C)
	}},
	// subtracts the contents of Register D from Register A and stores the result in Register A
	0x92: {"SUB D", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.subtractN(cpu.Registers.D)
	}},
	// subtracts the contents of Register E from Register A and stores the result in Register A
	0x93: {"SUB E", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.subtractN(cpu.Registers.E)
	}},
	// subtracts the contents of Register H from Register A and stores the result in Register A
	0x94: {"SUB H", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.subtractN(cpu.Registers.H)
	}},
	// subtracts the contents of Register L from Register A and stores the result in Register A
	0x95: {"SUB L", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.subtractN(cpu.Registers.L)
	}},
	// subtracts the contents of the memory location specified by Register pair HL from Register A and stores the result in Register A
	0x96: {"SUB (HL)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.subtractN(cpu.mmu.Read(cpu.Registers.HL.Uint16()))
	}},
	// subtracts the contents of Register A from Register A and stores the result in Register A
	0x97: {"SUB A", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.subtractN(cpu.Registers.A)
	}},
	// subtracts the contents of Register B from Register A and stores the result in Register A and sets the carry flag if there is a carry
	0x98: {"SBC A, B", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.subtractNCarry(cpu.Registers.B)
	}},
	// subtracts the contents of Register C from Register A and stores the result in Register A and sets the carry flag if there is a carry
	0x99: {"SBC A, C", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.subtractNCarry(cpu.Registers.C)
	}},
	// subtracts the contents of Register D from Register A and stores the result in Register A and sets the carry flag if there is a carry
	0x9A: {"SBC A, D", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.subtractNCarry(cpu.Registers.D)
	}},
	// subtracts the contents of Register E from Register A and stores the result in Register A and sets the carry flag if there is a carry
	0x9B: {"SBC A, E", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.subtractNCarry(cpu.Registers.E)
	}},
	// subtracts the contents of Register H from Register A and stores the result in Register A and sets the carry flag if there is a carry
	0x9C: {"SBC A, H", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.subtractNCarry(cpu.Registers.H)
	}},
	// subtracts the contents of Register L from Register A and stores the result in Register A and sets the carry flag if there is a carry
	0x9D: {"SBC A, L", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.subtractNCarry(cpu.Registers.L)
	}},
	// subtracts the contents of the memory location specified by Register pair HL from Register A and stores the result in Register A and sets the carry flag if there is a carry
	0x9E: {"SBC A, (HL)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.subtractNCarry(cpu.mmu.Read(cpu.Registers.HL.Uint16()))
	}},
	// subtracts the contents of Register A from Register A and stores the result in Register A and sets the carry flag if there is a carry
	0x9F: {"SBC A, A", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.subtractNCarry(cpu.Registers.A)
	}},
	// performs a bitwise AND operation on the contents of Register A and Register B and stores the result in Register A
	0xA0: {"AND B", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.andRegister(&cpu.Registers.B)
	}},
	// performs a bitwise AND operation on the contents of Register A and Register C and stores the result in Register A
	0xA1: {"AND C", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.andRegister(&cpu.Registers.C)
	}},
	// performs a bitwise AND operation on the contents of Register A and Register D and stores the result in Register A
	0xA2: {"AND D", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.andRegister(&cpu.Registers.D)
	}},
	// performs a bitwise AND operation on the contents of Register A and Register E and stores the result in Register A
	0xA3: {"AND E", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.andRegister(&cpu.Registers.E)
	}},
	// performs a bitwise AND operation on the contents of Register A and Register H and stores the result in Register A
	0xA4: {"AND H", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.andRegister(&cpu.Registers.H)
	}},
	// performs a bitwise AND operation on the contents of Register A and Register L and stores the result in Register A
	0xA5: {"AND L", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.andRegister(&cpu.Registers.L)
	}},
	// performs a bitwise AND operation on the contents of Register A and the memory location specified by Register pair HL and stores the result in Register A
	0xA6: {"AND (HL)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.Registers.A = Register(cpu.and(uint8(cpu.Registers.A), cpu.mmu.Read(cpu.Registers.HL.Uint16())))
	}},
	// performs a bitwise AND operation on the contents of Register A and Register A and stores the result in Register A
	0xA7: {"AND A", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.andRegister(&cpu.Registers.A)
	}},
	// performs a bitwise XOR operation on the contents of Register A and Register B and stores the result in Register A
	0xA8: {"XOR B", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.xorRegister(&cpu.Registers.B)
	}},
	// performs a bitwise XOR operation on the contents of Register A and Register C and stores the result in Register A
	0xA9: {"XOR C", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.xorRegister(&cpu.Registers.C)
	}},
	// performs a bitwise XOR operation on the contents of Register A and Register D and stores the result in Register A
	0xAA: {"XOR D", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.xorRegister(&cpu.Registers.D)
	}},
	// performs a bitwise XOR operation on the contents of Register A and Register E and stores the result in Register A
	0xAB: {"XOR E", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.xorRegister(&cpu.Registers.E)
	}},
	// performs a bitwise XOR operation on the contents of Register A and Register H and stores the result in Register A
	0xAC: {"XOR H", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.xorRegister(&cpu.Registers.H)
	}},
	// performs a bitwise XOR operation on the contents of Register A and Register L and stores the result in Register A
	0xAD: {"XOR L", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.xorRegister(&cpu.Registers.L)
	}},
	// performs a bitwise XOR operation on the contents of Register A and the memory location specified by Register pair HL and stores the result in Register A
	0xAE: {"XOR (HL)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.Registers.A = Register(cpu.xor(uint8(cpu.Registers.A), cpu.mmu.Read(cpu.Registers.HL.Uint16())))
	}},
	// performs a bitwise XOR operation on the contents of Register A and Register A and stores the result in Register A
	0xAF: {"XOR A", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.xorRegister(&cpu.Registers.A)
	}},
	// performs a bitwise OR operation on the contents of Register A and Register B and stores the result in Register A
	0xB0: {"OR B", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.orRegister(&cpu.Registers.B)
	}},
	// performs a bitwise OR operation on the contents of Register A and Register C and stores the result in Register A
	0xB1: {"OR C", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.orRegister(&cpu.Registers.C)
	}},
	// performs a bitwise OR operation on the contents of Register A and Register D and stores the result in Register A
	0xB2: {"OR D", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.orRegister(&cpu.Registers.D)
	}},
	// performs a bitwise OR operation on the contents of Register A and Register E and stores the result in Register A
	0xB3: {"OR E", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.orRegister(&cpu.Registers.E)
	}},
	// performs a bitwise OR operation on the contents of Register A and Register H and stores the result in Register A
	0xB4: {"OR H", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.orRegister(&cpu.Registers.H)
	}},
	// performs a bitwise OR operation on the contents of Register A and Register L and stores the result in Register A
	0xB5: {"OR L", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.orRegister(&cpu.Registers.L)
	}},
	// performs a bitwise OR operation on the contents of Register A and the memory location specified by Register pair HL and stores the result in Register A
	0xB6: {"OR (HL)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.Registers.A = Register(cpu.or(uint8(cpu.Registers.A), cpu.mmu.Read(cpu.Registers.HL.Uint16())))
	}},
	// performs a bitwise OR operation on the contents of Register A and Register A and stores the result in Register A
	0xB7: {"OR A", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.orRegister(&cpu.Registers.A)
	}},
	// compares the contents of Register A and Register B and sets the zero flag if they are equal
	0xB8: {"CP B", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.compareRegister(&cpu.Registers.B)
	}},
	// compares the contents of Register A and Register C and sets the zero flag if they are equal
	0xB9: {"CP C", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.compareRegister(&cpu.Registers.C)
	}},
	// compares the contents of Register A and Register D and sets the zero flag if they are equal
	0xBA: {"CP D", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.compareRegister(&cpu.Registers.D)
	}},
	// compares the contents of Register A and Register E and sets the zero flag if they are equal
	0xBB: {"CP E", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.compareRegister(&cpu.Registers.E)
	}},
	// compares the contents of Register A and Register H and sets the zero flag if they are equal
	0xBC: {"CP H", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.compareRegister(&cpu.Registers.H)
	}},
	// compares the contents of Register A and Register L and sets the zero flag if they are equal
	0xBD: {"CP L", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.compareRegister(&cpu.Registers.L)
	}},
	// compares the contents of Register A and the memory location specified by Register pair HL and sets the zero flag if they are equal
	0xBE: {"CP (HL)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.compare(cpu.mmu.Read(cpu.Registers.HL.Uint16()))
	}},
	// compares the contents of Register A and Register A and sets the zero flag if they are equal
	0xBF: {"CP A", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.compareRegister(&cpu.Registers.A)
	}},
	// return from subroutine if the zero flag is not set
	0xC0: {"RET NZ", 1, 5, func(cpu *CPU, operands []byte) {
		cpu.retConditional(!cpu.isFlagSet(FlagZero))
	}},
	// pop the contents from the memory stack and store them in Register pair BC
	0xC1: {"POP BC", 1, 3, func(cpu *CPU, operands []byte) {
		cpu.Registers.BC.SetUint16(cpu.pop16())
	}},
	// jump to the address specified by the immediate 16-bit operand if the zero flag is not set
	0xC2: {"JP NZ, a16", 3, 3, func(cpu *CPU, operands []byte) {
		cpu.jumpAbsoluteConditional(!cpu.isFlagSet(FlagZero), binary.LittleEndian.Uint16(operands))
	}},
	// jump to the address specified by the immediate 16-bit operand
	0xC3: {"JP a16", 3, 4, func(cpu *CPU, operands []byte) {
		cpu.jumpAbsolute(binary.LittleEndian.Uint16(operands))
	}},
	// call the address specified by the immediate 16-bit operand if the zero flag is not set
	0xC4: {"CALL NZ, a16", 3, 3, func(cpu *CPU, operands []byte) {
		cpu.callConditional(!cpu.isFlagSet(FlagZero), binary.LittleEndian.Uint16(operands))
	}},
	// push the contents of Register pair BC onto the memory stack
	0xC5: {"PUSH BC", 1, 4, func(cpu *CPU, operands []byte) {
		cpu.push16(cpu.Registers.BC.Uint16())
	}},
	// add the immediate 8-bit operand to the contents of Register A and store the result in Register A
	0xC6: {"ADD A, d8", 2, 2, func(cpu *CPU, operands []byte) {
		cpu.addN(operands[0])
	}},
	// push the current contents of the program counter onto the memory stack (the contents of the lower-order byte of PC are pushed onto the stack first,
	// followed by the contents of the higher-order byte of PC). The contents of SP are decremented by 1 and the contents of the higher-order byte of PC are
	// stored in the address specified by the new SP value. The contents of SP are decremented by 1 and the contents of the lower-order byte of PC are stored
	// in the address specified by the new SP value (the stack pointer is decremented by 2 in total during this operation). The next instruction is fetched
	// from the address specified by the new contents of the program counter (as usual)
	0xC7: {"RST 0", 1, 4, func(cpu *CPU, operands []byte) {
		cpu.rst(0x00)
	}},
	// control is returned to the source of the call instruction by popping from the stack the program counter value that was pushed onto the stack by
	// the call instruction. The contents of the address specified by the stack pointer are loaded in the lower-order byte of PC and the contents of SP
	// are incremented by 1. The contents of the address specified by the new SP value are then loaded in the higher-order byte of PC and the contents of
	// SP are incremented by 1 (the stack pointer is incremented by 2 in total during this operation) The next instruction is fetched from the address
	// specified by the new contents of the program counter (as usual)
	0xC8: {"RET Z", 1, 5, func(cpu *CPU, operands []byte) {
		cpu.retConditional(cpu.isFlagSet(FlagZero))
	}},
	// return
	0xC9: {"RET", 1, 4, func(cpu *CPU, operands []byte) {
		cpu.ret()
	}},
	// jump to the address specified by the immediate 16-bit operand if the zero flag is set
	0xCA: {"JP Z, a16", 3, 3, func(cpu *CPU, operands []byte) {
		cpu.jumpAbsoluteConditional(cpu.isFlagSet(FlagZero), binary.LittleEndian.Uint16(operands))
	}},
	// call the address specified by the immediate 16-bit operand
	0xCC: {"CALL Z, a16", 3, 3, func(cpu *CPU, operands []byte) {
		cpu.callConditional(cpu.isFlagSet(FlagZero), binary.LittleEndian.Uint16(operands))
	}},
	// call the address specified by the immediate 16-bit operand
	0xCD: {"CALL a16", 3, 6, func(cpu *CPU, operands []byte) {
		cpu.call(binary.LittleEndian.Uint16(operands))
	}},
	// add the immediate 8-bit operand to the contents of Register A and store the result in Register A
	0xCE: {"ADC A, d8", 2, 2, func(cpu *CPU, operands []byte) {
		cpu.addNCarry(operands[0])
	}},
	// push the current contents of the program counter onto the memory stack (the contents of the lower-order byte of PC are pushed onto the stack first,
	// followed by the contents of the higher-order byte of PC). The contents of SP are decremented by 1 and the contents of the higher-order byte of PC are
	// stored in the address specified by the new SP value. The contents of SP are decremented by 1 and the contents of the lower-order byte of PC are stored
	// in the address specified by the new SP value (the stack pointer is decremented by 2 in total during this operation). The next instruction is fetched
	// from the address specified by the new contents of the program counter (as usual)
	0xCF: {"RST 1", 1, 4, func(cpu *CPU, operands []byte) {
		cpu.rst(0x08)
	}},
	// if the carry flag is not set, the next instruction is fetched from the address specified by the immediate 8-bit operand. Otherwise, the next
	// instruction is fetched from the address specified by the program counter (as usual)
	0xD0: {"RET NC", 1, 5, func(cpu *CPU, operands []byte) {
		cpu.retConditional(!cpu.isFlagSet(FlagCarry))
	}},
	// pop the contents of the memory stack into Register pair DE
	0xD1: {"POP DE", 1, 3, func(cpu *CPU, operands []byte) {
		cpu.DE.SetUint16(cpu.pop16())
	}},
	// jump to the address specified by the immediate 16-bit operand if the carry flag is not set
	0xD2: {"JP NC, a16", 3, 3, func(cpu *CPU, operands []byte) {
		cpu.jumpAbsoluteConditional(!cpu.isFlagSet(FlagCarry), binary.LittleEndian.Uint16(operands))
	}},
	// call the address specified by the immediate 16-bit operand if the carry flag is not set
	0xD4: {"CALL NC, a16", 3, 3, func(cpu *CPU, operands []byte) {
		cpu.callConditional(!cpu.isFlagSet(FlagCarry), binary.LittleEndian.Uint16(operands))
	}},
	// push the contents of Register pair DE onto the memory stack
	0xD5: {"PUSH DE", 1, 4, func(cpu *CPU, operands []byte) {
		cpu.push(uint8(cpu.Registers.D))
		cpu.push(uint8(cpu.Registers.E))
	}},
	// subtract the immediate 8-bit operand from the contents of Register A and store the result in Register A
	0xD6: {"SUB d8", 2, 2, func(cpu *CPU, operands []byte) {
		cpu.subtractN(operands[0])
	}},
	// push the current contents of the program counter onto the memory stack (the contents of the lower-order byte of PC are pushed onto the stack first,
	// followed by the contents of the higher-order byte of PC). The contents of SP are decremented by 1 and the contents of the higher-order byte of PC are
	// stored in the address specified by the new SP value. The contents of SP are decremented by 1 and the contents of the lower-order byte of PC are stored
	// in the address specified by the new SP value (the stack pointer is decremented by 2 in total during this operation). The next instruction is fetched
	// from the address specified by the new contents of the program counter (as usual)
	0xD7: {"RST 2", 1, 4, func(cpu *CPU, operands []byte) {
		cpu.rst(0x10)
	}},
	// if the carry flag is set, the next instruction is fetched from the address specified by the immediate 8-bit operand. Otherwise, the next
	// instruction is fetched from the address specified by the program counter (as usual)
	0xD8: {"RET C", 1, 5, func(cpu *CPU, operands []byte) {
		cpu.retConditional(cpu.isFlagSet(FlagCarry))
	}},
	// pop the contents of the memory stack into Register pair PC, and jump to that address
	0xD9: {"RETI", 1, 4, func(cpu *CPU, operands []byte) {
		cpu.retInterrupt()
	}},
	// jump to the address specified by the immediate 16-bit operand if the carry flag is set
	0xDA: {"JP C, a16", 3, 3, func(cpu *CPU, operands []byte) {
		cpu.jumpAbsoluteConditional(cpu.isFlagSet(FlagCarry), binary.LittleEndian.Uint16(operands))
	}},
	// call the address specified by the immediate 16-bit operand if the carry flag is set
	0xDC: {"CALL C, a16", 3, 3, func(cpu *CPU, operands []byte) {
		cpu.callConditional(cpu.isFlagSet(FlagCarry), binary.LittleEndian.Uint16(operands))
	}},
	// subtract the immediate 8-bit operand from the contents of Register A and store the result in Register A
	0xDE: {"SBC A, d8", 2, 2, func(cpu *CPU, operands []byte) {
		cpu.subtractNCarry(operands[0])
	}},
	// push the current contents of the program counter onto the memory stack (the contents of the lower-order byte of PC are pushed onto the stack first,
	// followed by the contents of the higher-order byte of PC). The contents of SP are decremented by 1 and the contents of the higher-order byte of PC are
	// stored in the address specified by the new SP value. The contents of SP are decremented by 1 and the contents of the lower-order byte of PC are stored
	// in the address specified by the new SP value (the stack pointer is decremented by 2 in total during this operation). The next instruction is fetched
	// from the address specified by the new contents of the program counter (as usual)
	0xDF: {"RST 3", 1, 4, func(cpu *CPU, operands []byte) {
		cpu.rst(0x18)
	}},
	// load the contents of Register A into the address specified by the immediate 8-bit operand
	0xE0: {"LD (a8), A", 2, 3, func(cpu *CPU, operands []byte) {
		cpu.mmu.Write(0xFF00+uint16(operands[0]), cpu.Registers.A)
	}},
	// pop the contents of the memory stack into Register pair HL
	0xE1: {"POP HL", 1, 3, func(cpu *CPU, operands []byte) {
		cpu.HL.SetUint16(cpu.pop16())
	}},
	// load the contents of Register A in the internal RAM, port Register, or mode Register at the address in the range 0xFF00 to 0xFFFF specified by the
	// Register C
	// 0xFF00-0xFF7F: Port/Mode registers, control registers, sound Register
	// 0xFF80-0xFFFE: Internal RAM
	// 0xFFFF: Interrupt Enable Register
	0xE2: {"LD (C), A", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.mmu.Write(0xFF00+uint16(cpu.Registers.C), cpu.Registers.A)
	}},
	// push the contents of Register pair HL onto the memory stack
	0xE5: {"PUSH HL", 1, 4, func(cpu *CPU, operands []byte) {
		cpu.push(uint8(cpu.Registers.H))
		cpu.push(uint8(cpu.Registers.L))
	}},
	// add the immediate 8-bit operand to the contents of Register A and store the result in Register A
	0xE6: {"AND d8", 2, 2, func(cpu *CPU, operands []byte) {
		cpu.and(cpu.A, operands[0])
	}},
	// push the current contents of the program counter onto the memory stack (the contents of the lower-order byte of PC are pushed onto the stack first,
	// followed by the contents of the higher-order byte of PC). The contents of SP are decremented by 1 and the contents of the higher-order byte of PC are
	// stored in the address specified by the new SP value. The contents of SP are decremented by 1 and the contents of the lower-order byte of PC are stored
	// in the address specified by the new SP value (the stack pointer is decremented by 2 in total during this operation). The next instruction is fetched
	// from the address specified by the new contents of the program counter (as usual)
	0xE7: {"RST 4", 1, 4, func(cpu *CPU, operands []byte) {
		cpu.rst(0x20)
	}},
	// adds the contents of the 8-bit signed immediate operand and the stack pointer and stores the result in the stack pointer
	0xE8: {"ADD SP, r8", 2, 4, func(cpu *CPU, operands []byte) {
		var computed uint16
		if operands[0] > 127 {
			computed = cpu.SP - uint16(-operands[0])
		} else {
			computed = cpu.SP + uint16(operands[0])
		}
		cpu.SP = computed

		carry := cpu.SP ^ uint16(operands[0]) ^ ((cpu.SP + uint16(operands[0])) & 0xFFFF)

		if (carry & 0x100) == 0x100 {
			cpu.setFlag(FlagCarry)
		} else {
			cpu.clearFlag(FlagCarry)
		}

		if (carry & 0x10) == 0x10 {
			cpu.setFlag(FlagHalfCarry)
		} else {
			cpu.clearFlag(FlagHalfCarry)
		}

		cpu.clearFlag(FlagZero)
		cpu.clearFlag(FlagSubtract)
	}},
	// jump to the address specified by the HL Register pair
	0xE9: {"JP (HL)", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.jumpAbsolute(cpu.HL.Uint16())
	}},
	// load the contents of Register A in the internal RAM, port Register, or mode Register at the address in the range 0xFF00 to 0xFFFF specified by the
	// immediate 8-bit operand
	// 0xFF00-0xFF7F: Port/Mode registers, control registers, sound Register
	// 0xFF80-0xFFFE: Internal RAM
	// 0xFFFF: Interrupt Enable Register
	0xEA: {"LD (a16), A", 3, 4, func(cpu *CPU, operands []byte) {
		cpu.mmu.Write(uint16(operands[0])<<8|uint16(operands[1]), cpu.Registers.A)
	}},
	// xor the immediate 8-bit operand with the contents of Register A and store the result in Register A
	0xEE: {"XOR d8", 2, 2, func(cpu *CPU, operands []byte) {
		cpu.Registers.A = Register(cpu.xor(uint8(cpu.Registers.A), operands[0]))
	}},
	// push the current contents of the program counter onto the memory stack (the contents of the lower-order byte of PC are pushed onto the stack first,
	// followed by the contents of the higher-order byte of PC). The contents of SP are decremented by 1 and the contents of the higher-order byte of PC are
	// stored in the address specified by the new SP value. The contents of SP are decremented by 1 and the contents of the lower-order byte of PC are stored
	// in the address specified by the new SP value (the stack pointer is decremented by 2 in total during this operation). The next instruction is fetched
	// from the address specified by the new contents of the program counter (as usual)
	0xEF: {"RST 5", 1, 4, func(cpu *CPU, operands []byte) {
		cpu.rst(0x28)
	}},
	// load into Register "A" the contents of the internal RAM, port Register, or mode Register at the address in the range 0xFF00 to 0xFFFF specified by the
	// immediate 8-bit operand.
	// 0xFF00-0xFF7F: Port/Mode registers, control registers, sound Register
	// 0xFF80-0xFFFE: Internal RAM
	// 0xFFFF: Interrupt Enable Register
	0xF0: {"LD A, (a8)", 2, 3, func(cpu *CPU, operands []byte) {
		cpu.loadMemoryToRegister(&cpu.Registers.A, 0xFF00+uint16(operands[0]))
	}},
	// pop the contents of the memory stack into Register pair AF
	0xF1: {"POP AF", 1, 3, func(cpu *CPU, operands []byte) {
		cpu.AF.SetUint16(cpu.pop16())
	}},
	// load into Register "A" the contents of the internal RAM, port Register, or mode Register at the address in the range 0xFF00 to 0xFFFF specified by the
	// Register C
	// 0xFF00-0xFF7F: Port/Mode registers, control registers, sound Register
	// 0xFF80-0xFFFE: Internal RAM
	// 0xFFFF: Interrupt Enable Register
	0xF2: {"LD A, (C)", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.loadMemoryToRegister(&cpu.Registers.A, 0xFF00+uint16(cpu.Registers.C))
	}},
	// disable interrupts after the next instruction is executed
	0xF3: {"DI", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.mmu.Bus.Interrupts().IME = false
	}},
	// push the contents of Register pair AF onto the memory stack
	0xF5: {"PUSH AF", 1, 4, func(cpu *CPU, operands []byte) {
		cpu.push16(cpu.AF.Uint16())
	}},
	// or the immediate 8-bit operand with the contents of Register A and store the result in Register A
	0xF6: {"OR d8", 2, 2, func(cpu *CPU, operands []byte) {
		cpu.or(cpu.Registers.A, operands[0])
	}},
	// push the current contents of the program counter onto the memory stack (the contents of the lower-order byte of PC are pushed onto the stack first,
	// followed by the contents of the higher-order byte of PC). The contents of SP are decremented by 1 and the contents of the higher-order byte of PC are
	// stored in the address specified by the new SP value. The contents of SP are decremented by 1 and the contents of the lower-order byte of PC are stored
	// in the address specified by the new SP value (the stack pointer is decremented by 2 in total during this operation). The next instruction is fetched
	// from the address specified by the new contents of the program counter (as usual)
	0xF7: {"RST 6", 1, 4, func(cpu *CPU, operands []byte) {
		cpu.rst(0x30)
	}},
	// add the 8-bit signed operand to the stack pointer and store the result in the Register pair HL
	0xF8: {"LD HL, SP+r8", 2, 3, func(cpu *CPU, operands []byte) {
		if int8(operands[0]) < 0 {
			cpu.HL.SetUint16(cpu.SP - uint16(-int8(operands[0])))
		} else {
			cpu.HL.SetUint16(cpu.SP + uint16(int8(operands[0])))
		}
		// check if the carry flag should be set
		if (cpu.SP&0x0F)+(uint16(operands[0])&0x0F) > 0x0F {
			cpu.setFlag(FlagCarry)
		} else {
			cpu.clearFlag(FlagCarry)
		}
		// check if the half carry flag should be set
		if (cpu.SP&0xFF)+(uint16(operands[0])&0xFF) > 0xFF {
			cpu.setFlag(FlagHalfCarry)
		} else {
			cpu.clearFlag(FlagHalfCarry)
		}
		cpu.clearFlag(FlagSubtract)
		cpu.clearFlag(FlagZero)
	}},
	// load the contents of Register pair HL into the stack pointer
	0xF9: {"LD SP, HL", 1, 2, func(cpu *CPU, operands []byte) {
		cpu.SP = cpu.HL.Uint16()
	}},
	// load into Register "A" the contents of the memory address specified by the immediate 16-bit operand
	0xFA: {"LD A, (a16)", 3, 4, func(cpu *CPU, operands []byte) {
		cpu.loadMemoryToRegister(&cpu.Registers.A, binary.LittleEndian.Uint16(operands))
	}},
	// set the interrupt master enable flag and enable maskable interrupts
	0xFB: {"EI", 1, 1, func(cpu *CPU, operands []byte) {
		cpu.mmu.Bus.Interrupts().IME = true
	}},
	// compare the contents of Register A with the immediate 8-bit operand by subtracting the operand from the contents of Register A and setting the zero
	// flag is they are equal. The contents of Register A are not changed
	0xFE: {"CP d8", 2, 2, func(cpu *CPU, operands []byte) {
		cpu.compare(operands[0])
	}},
	// push the current contents of the program counter onto the memory stack (the contents of the lower-order byte of PC are pushed onto the stack first,
	// followed by the contents of the higher-order byte of PC). The contents of SP are decremented by 1 and the contents of the higher-order byte of PC are
	// stored in the address specified by the new SP value. The contents of SP are decremented by 1 and the contents of the lower-order byte of PC are stored
	// in the address specified by the new SP value (the stack pointer is decremented by 2 in total during this operation). The next instruction is fetched
	// from the address specified by the new contents of the program counter (as usual)
	0xFF: {"RST 7", 1, 4, func(cpu *CPU, operands []byte) {
		cpu.rst(0x38)
	}},
	// disallowed opcodes
	0xCB: {"", 0, 0, disallowedOpcode},
	0xD3: {"", 0, 0, disallowedOpcode},
	0xDB: {"", 0, 0, disallowedOpcode},
	0xDD: {"", 0, 0, disallowedOpcode},
	0xE3: {"", 0, 0, disallowedOpcode},
	0xE4: {"", 0, 0, disallowedOpcode},
	0xEB: {"", 0, 0, disallowedOpcode},
	0xEC: {"", 0, 0, disallowedOpcode},
	0xED: {"", 0, 0, disallowedOpcode},
	0xF4: {"", 0, 0, disallowedOpcode},
	0xFC: {"", 0, 0, disallowedOpcode},
	0xFD: {"", 0, 0, disallowedOpcode},
}

var InstructionSetCB = map[uint8]Instruction{
	// (HL) CB Instructions need to be manually implemented
	0x76: {"BIT 6 (HL)", 2, 4, func(cpu *CPU, operands []byte) {
		cpu.testBit(cpu.mmu.Read(cpu.HL.Uint16()), 6)
	}},
	0x7E: {"BIT 7 (HL)", 2, 4, func(cpu *CPU, operands []byte) {
		cpu.testBit(cpu.mmu.Read(cpu.HL.Uint16()), 7)
	}},
	0xBE: {"RES 7 (HL)", 2, 4, func(cpu *CPU, operands []byte) {
		cpu.mmu.Write(cpu.HL.Uint16(), cpu.clearBit(cpu.mmu.Read(cpu.HL.Uint16()), 7))
	}},
	0xD6: {"SET 2 (HL)", 2, 4, func(cpu *CPU, operands []byte) {
		cpu.mmu.Write(cpu.HL.Uint16(), cpu.setBit(cpu.mmu.Read(cpu.HL.Uint16()), 2))
	}},
	0xDE: {"SET 3 (HL)", 2, 4, func(cpu *CPU, operands []byte) {
		cpu.mmu.Write(cpu.HL.Uint16(), cpu.setBit(cpu.mmu.Read(cpu.HL.Uint16()), 3))
	}},
	0xE6: {"SET 4 (HL)", 2, 4, func(cpu *CPU, operands []byte) {
		cpu.mmu.Write(cpu.HL.Uint16(), cpu.setBit(cpu.mmu.Read(cpu.HL.Uint16()), 4))
	}},
	0xEE: {"SET 5 (HL)", 2, 4, func(cpu *CPU, operands []byte) {
		cpu.mmu.Write(cpu.HL.Uint16(), cpu.setBit(cpu.mmu.Read(cpu.HL.Uint16()), 5))
	}},
	0xF6: {"SET 6 (HL)", 2, 4, func(cpu *CPU, operands []byte) {
		cpu.mmu.Write(cpu.HL.Uint16(), cpu.setBit(cpu.mmu.Read(cpu.HL.Uint16()), 6))
	}},
	0xFE: {"SET 7 (HL)", 2, 4, func(cpu *CPU, operands []byte) {
		cpu.mmu.Write(cpu.HL.Uint16(), cpu.setBit(cpu.mmu.Read(cpu.HL.Uint16()), 7))
	}},
}

func (c *CPU) generateCBInstructionSet() {
	for i := uint8(0); true; i++ {
		// get current reg pointer
		var reg *Register
		switch i & 0x07 {
		case 0:
			reg = &c.Registers.B
		case 1:
			reg = &c.Registers.C
		case 2:
			reg = &c.Registers.D
		case 3:
			reg = &c.Registers.E
		case 4:
			reg = &c.Registers.H
		case 5:
			reg = &c.Registers.L
		case 6:
			// handle (HL) instructions
			switch i >> 3 {
			case 0:
				InstructionSetCB[i] = Instruction{"RLC (HL)", 2, 4, func(cpu *CPU, operands []byte) {
					cpu.mmu.Write(cpu.HL.Uint16(), cpu.rotateLeftThroughCarry(cpu.mmu.Read(cpu.HL.Uint16())))
				}}
			case 1:
				InstructionSetCB[i] = Instruction{"RRC (HL)", 2, 4, func(cpu *CPU, operands []byte) {
					cpu.mmu.Write(cpu.HL.Uint16(), cpu.rotateRightThroughCarry(cpu.mmu.Read(cpu.HL.Uint16())))
				}}
			case 2:
				InstructionSetCB[i] = Instruction{"RL (HL)", 2, 4, func(cpu *CPU, operands []byte) {
					cpu.mmu.Write(cpu.HL.Uint16(), cpu.rotateLeft(cpu.mmu.Read(cpu.HL.Uint16())))
				}}
			case 3:
				InstructionSetCB[i] = Instruction{"RR (HL)", 2, 4, func(cpu *CPU, operands []byte) {
					cpu.mmu.Write(cpu.HL.Uint16(), cpu.rotateRight(cpu.mmu.Read(cpu.HL.Uint16())))
				}}
			case 4:
				InstructionSetCB[i] = Instruction{"SLA (HL)", 2, 4, func(cpu *CPU, operands []byte) {
					cpu.mmu.Write(cpu.HL.Uint16(), cpu.shiftLeftIntoCarry(cpu.mmu.Read(cpu.HL.Uint16())))
				}}
			case 5:
				InstructionSetCB[i] = Instruction{"SRA (HL)", 2, 4, func(cpu *CPU, operands []byte) {
					cpu.mmu.Write(cpu.HL.Uint16(), cpu.shiftRightIntoCarry(cpu.mmu.Read(cpu.HL.Uint16())))
				}}
			case 6:
				InstructionSetCB[i] = Instruction{"SWAP (HL)", 2, 4, func(cpu *CPU, operands []byte) {
					cpu.mmu.Write(cpu.HL.Uint16(), cpu.swapByte(cpu.mmu.Read(cpu.HL.Uint16())))
				}}
			case 7:
				InstructionSetCB[i] = Instruction{"SRL (HL)", 2, 4, func(cpu *CPU, operands []byte) {
					cpu.mmu.Write(cpu.HL.Uint16(), cpu.shiftRightLogical(cpu.mmu.Read(cpu.HL.Uint16())))
				}}
			}
			continue
		case 7:
			reg = &c.Registers.A
		}

		// rotate left carry
		if i <= 0x07 {
			InstructionSetCB[i] = Instruction{
				"RLC " + c.registerName(reg),
				1,
				2,
				func(cpu *CPU, operands []byte) {
					*reg = cpu.rotateLeft(*reg)
				},
			}
		}
		// rotate right carry
		if i >= 0x08 && i <= 0x0F {
			InstructionSetCB[i] = Instruction{
				"RRC " + c.registerName(reg),
				1,
				2,
				func(cpu *CPU, operands []byte) {
					*reg = cpu.rotateRight(*reg)
				},
			}
		}

		// rotate left
		if i >= 0x10 && i <= 0x17 {
			InstructionSetCB[i] = Instruction{
				"RL " + c.registerName(reg),
				1,
				2,
				func(cpu *CPU, operands []byte) {
					*reg = cpu.rotateLeftThroughCarry(*reg)
				},
			}
		}

		// rotate right
		if i >= 0x18 && i <= 0x1F {
			InstructionSetCB[i] = Instruction{
				"RR " + c.registerName(reg),
				1,
				2,
				func(cpu *CPU, operands []byte) {
					*reg = cpu.rotateRightThroughCarry(*reg)
				},
			}
		}

		// shift left arithmetic
		if i >= 0x20 && i <= 0x27 {
			InstructionSetCB[i] = Instruction{
				"SLA " + c.registerName(reg),
				1,
				2,
				func(cpu *CPU, operands []byte) {
					*reg = cpu.shiftLeftIntoCarry(*reg)
				},
			}
		}

		// shift right arithmetic
		if i >= 0x28 && i <= 0x2F {
			InstructionSetCB[i] = Instruction{
				"SRA " + c.registerName(reg),
				1,
				2,
				func(cpu *CPU, operands []byte) {
					*reg = cpu.shiftRightIntoCarry(*reg)
				},
			}
		}

		// swap upper and lower nibbles
		if i >= 0x30 && i <= 0x37 {
			InstructionSetCB[i] = Instruction{
				"SWAP " + c.registerName(reg),
				1,
				2,
				func(cpu *CPU, operands []byte) {
					cpu.swap(reg)
				},
			}
		}

		// shift right logical
		if i >= 0x38 && i <= 0x3F {
			InstructionSetCB[i] = Instruction{
				"SRL " + c.registerName(reg),
				1,
				2,
				func(cpu *CPU, operands []byte) {
					*reg = cpu.shiftRightLogical(*reg)
				},
			}
		}

		// test bits
		if i >= 0x40 && i <= 0x7F {
			InstructionSetCB[i] = Instruction{
				"BIT " + strconv.Itoa(int((i&0x38)>>3)) + ", " + c.registerName(reg),
				1,
				2,
				func(cpu *CPU, operands []byte) {
					cpu.testBit(*reg, (i&0x38)>>3)
				},
			}
		}

		// reset bits
		if i >= 0x80 && i <= 0xBF {
			InstructionSetCB[i] = Instruction{
				"RES " + strconv.Itoa(int((i&0x38)>>3)) + ", " + c.registerName(reg),
				1,
				2,
				func(cpu *CPU, operands []byte) {
					*reg = cpu.clearBit(*reg, (i&0x38)>>3)
				},
			}
		}

		// set bits
		if i >= 0xC0 {
			InstructionSetCB[i] = Instruction{
				"SET " + strconv.Itoa(int((i&0x38)>>3)) + ", " + c.registerName(reg),
				1,
				2,
				func(cpu *CPU, operands []byte) {
					*reg = cpu.setBit(*reg, (i&0x38)>>3)
				},
			}
		}

		if i == 255 {
			break
		}
	}
}

func disallowedOpcode(cpu *CPU, operands []byte) {
	panic(fmt.Sprintf("disallowed opcode %X", cpu.mmu.Read(cpu.PC)))
}

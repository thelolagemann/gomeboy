package cpu

import (
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/utils"
)

// Instruction represents a single instruction of the
// CPU.
type Instruction struct {
	name string     // name of the instruction
	fn   func(*CPU) // fn called when executing the instruction
}

// disallowedOpcode creates an instruction that will panic if executed.
func disallowedOpcode(opcode uint8) Instruction {
	return Instruction{
		name: fmt.Sprintf("disallowed opcode %X", opcode),
		fn: func(cpu *CPU) {
			//panic(fmt.Sprintf("disallowed opcode %X at %04x", opcode, cpu.PC))
		},
	}
}

// InstructionSet holds the first 256 instructions.
var InstructionSet = [256]Instruction{
	0x00: {
		"NOP",
		func(c *CPU) {},
	},
	0x01: {
		"LD BC,d16",
		func(c *CPU) {
			c.loadRegister16(c.BC)
		},
	},
	0x02: {
		"LD (BC), A",
		func(c *CPU) {
			c.loadRegisterToMemory(c.A, c.BC.Uint16())
		},
	},
	0x03: {
		"INC BC",
		func(c *CPU) {
			c.incrementNN(c.BC)
		},
	},
	0x04: {
		"INC B",
		func(c *CPU) {
			c.B = c.increment(c.B)
		},
	},
	0x05: {
		"DEC B",
		func(c *CPU) {
			c.B = c.decrement(c.B)
		},
	},
	0x06: {
		"LD B, d8",
		func(c *CPU) {
			c.loadRegister8(&c.B)
		},
	},
	0x07: {
		"RLCA",
		func(c *CPU) {
			c.rotateLeftCarryAccumulator()
		},
	},
	0x08: {
		"LD (a16), SP",
		func(c *CPU) {
			low := c.readOperand()
			high := c.readOperand()

			address := uint16(high)<<8 | uint16(low)
			c.b.ClockedWrite(address, uint8(c.SP&0xFF))
			c.b.ClockedWrite(address+1, uint8(c.SP>>8))
		},
	},
	0x09: {
		"ADD HL, BC",
		func(c *CPU) {
			c.addHLRR(c.BC)
		},
	},
	0x0A: {
		"LD A, (BC)",
		func(c *CPU) {
			c.loadMemoryToRegister(&c.A, c.BC.Uint16())
		},
	},
	0x0B: {
		"DEC BC",
		func(c *CPU) {
			c.decrementNN(c.BC)
		},
	},
	0x0C: {
		"INC C",
		func(c *CPU) {
			c.C = c.increment(c.C)
		},
	},
	0x0D: {
		"DEC C",
		func(c *CPU) {
			c.C = c.decrement(c.C)
		},
	},
	0x0E: {
		"LD C, d8",
		func(c *CPU) {
			c.loadRegister8(&c.C)
		},
	},
	0x0F: {
		"RRCA",
		func(c *CPU) {
			c.rotateRightAccumulator()
		},
	},
	0x10: {
		"STOP",
		func(c *CPU) {
			// reset div clock
			c.s.SysClockReset()

			// if there's no pending interrupt then STOP becomes a 2-byte opcode
			if !c.b.HasInterrupts() {
				c.PC++
			}

			// are we in gbc mode (STOP is alternatively used for speed-switching)
			if c.b.Model() == types.CGB0 || c.b.Model() == types.CGBABC &&
				c.b.Get(types.KEY1)&types.Bit0 == types.Bit0 {
				// TODO unimplemented
				c.doubleSpeed = !c.doubleSpeed
				c.s.ChangeSpeed(c.doubleSpeed)

				if c.doubleSpeed {
					c.b.SetBit(types.KEY1, types.Bit7)
				} else {
					c.b.ClearBit(types.KEY1, types.Bit7)
				}

				// clear armed bit
				c.b.ClearBit(types.KEY1, types.Bit0)
			}
		},
	},
	0x11: {
		"LD DE, d16",
		func(c *CPU) {
			c.loadRegister16(c.DE)
		},
	},
	0x12: {
		"LD (DE), A",
		func(c *CPU) {
			c.loadRegisterToMemory(c.A, c.DE.Uint16())
		},
	},
	0x13: {
		"INC DE",
		func(c *CPU) {
			c.incrementNN(c.DE)
		},
	},
	0x14: {
		"INC D",
		func(c *CPU) {
			c.D = c.increment(c.D)
		},
	},
	0x15: {
		"DEC D",
		func(c *CPU) {
			c.D = c.decrement(c.D)
		},
	},
	0x16: {
		"LD D, d8",
		func(c *CPU) {
			c.loadRegister8(&c.D)
		},
	},
	0x17: {
		"RLA",
		func(c *CPU) {
			c.rotateLeftAccumulatorThroughCarry()
		},
	},
	0x18: {
		"JR r8",
		func(c *CPU) {
			c.jumpRelative(true)
		},
	},
	0x19: {
		"ADD HL, DE",
		func(c *CPU) {
			c.addHLRR(c.DE)
		},
	},
	0x1A: {
		"LD A, (DE)",
		func(c *CPU) {
			c.loadMemoryToRegister(&c.A, c.DE.Uint16())
		},
	},
	0x1B: {
		"DEC DE",
		func(c *CPU) {
			c.decrementNN(c.DE)
		},
	},
	0x1C: {
		"INC E",
		func(c *CPU) {
			c.E = c.increment(c.E)
		},
	},
	0x1D: {
		"DEC E",
		func(c *CPU) {
			c.E = c.decrement(c.E)
		},
	},
	0x1E: {
		"LD E, d8",
		func(c *CPU) {
			c.loadRegister8(&c.E)
		},
	},
	0x1F: {
		"RRA",
		func(c *CPU) {
			c.rotateRightAccumulatorThroughCarry()
		},
	},
	0x20: {
		"JR NZ, r8",
		func(c *CPU) {
			c.jumpRelative(!c.isFlagSet(flagZero))
		},
	},
	0x21: {
		"LD HL, d16",
		func(c *CPU) {
			c.loadRegister16(c.HL)
		},
	},
	0x22: {
		"LD (HL+), A",
		func(c *CPU) {
			c.loadRegisterToMemory(c.A, c.HL.Uint16())
			c.HL.SetUint16(c.HL.Uint16() + 1)
		},
	},
	0x23: {
		"INC HL",
		func(c *CPU) {
			c.incrementNN(c.HL)
		},
	},
	0x24: {
		"INC H",
		func(c *CPU) {
			c.H = c.increment(c.H)
		},
	},
	0x25: {
		"DEC H",
		func(c *CPU) {
			c.H = c.decrement(c.H)
		},
	},
	0x26: {
		"LD H, d8",
		func(c *CPU) {
			c.loadRegister8(&c.H)
		},
	},
	0x27: {
		"DAA",
		func(c *CPU) {
			if !c.isFlagSet(flagSubtract) {
				if c.isFlagSet(flagCarry) || c.A > 0x99 {
					c.A += 0x60
					c.F |= flagCarry
				}
				if c.isFlagSet(flagHalfCarry) || c.A&0xF > 0x9 {
					c.A += 0x06
					c.clearFlag(flagHalfCarry)
				}
			} else if c.isFlagSet(flagCarry) && c.isFlagSet(flagHalfCarry) {
				c.A += 0x9a
				c.clearFlag(flagHalfCarry)
			} else if c.isFlagSet(flagCarry) {
				c.A += 0xa0
			} else if c.isFlagSet(flagHalfCarry) {
				c.A += 0xfa
				c.clearFlag(flagHalfCarry)
			}
			if c.A == 0 {
				c.F |= flagZero
			} else {
				c.clearFlag(flagZero)
			}
		},
	},
	0x28: {
		"JR Z, r8",
		func(c *CPU) {
			c.jumpRelative(c.isFlagSet(flagZero))
		},
	},
	0x29: {
		"ADD HL, HL",
		func(c *CPU) {
			c.addHLRR(c.HL)
		},
	},
	0x2A: {
		"LD A, (HL+)",
		func(c *CPU) {
			c.handleOAMCorruption(c.HL.Uint16())
			c.loadMemoryToRegister(&c.A, c.HL.Uint16())
			c.HL.SetUint16(c.HL.Uint16() + 1)
		},
	},
	0x2B: {
		"DEC HL",
		func(c *CPU) {
			c.decrementNN(c.HL)
		},
	},
	0x2C: {
		"INC L",
		func(c *CPU) {
			c.L = c.increment(c.L)
		},
	},
	0x2D: {
		"DEC L",
		func(c *CPU) {
			c.L = c.decrement(c.L)
		},
	},
	0x2E: {
		"LD L, d8",
		func(c *CPU) {
			c.loadRegister8(&c.L)
		},
	},
	0x2F: {
		"CPL",
		func(c *CPU) {
			c.A = 0xFF ^ c.A
			c.setFlags(c.isFlagSet(flagZero), true, true, c.isFlagSet(flagCarry))
		},
	},
	0x30: {
		"JR NC, r8",
		func(c *CPU) {
			c.jumpRelative(!c.isFlagSet(flagCarry))
		},
	},
	0x31: {
		"LD SP, d16",
		func(c *CPU) {
			low := c.readOperand()
			high := c.readOperand()

			c.SP = uint16(high)<<8 | uint16(low)
		},
	},
	0x32: {
		"LD (HL-), A",
		func(c *CPU) {
			c.loadRegisterToMemory(c.A, c.HL.Uint16())
			c.HL.SetUint16(c.HL.Uint16() - 1)
		},
	},
	0x33: {
		"INC SP",
		func(c *CPU) {
			if c.SP >= 0xFE00 && c.SP <= 0xFEFF && c.b.Get(types.STAT)&0b11 == ppu.ModeOAM {
				c.ppu.WriteCorruptionOAM()
			}
			c.SP++
			c.s.Tick(4)
		},
	},
	0x34: {
		"INC (HL)",
		func(c *CPU) {
			c.b.ClockedWrite(c.HL.Uint16(), c.increment(c.b.ClockedRead(c.HL.Uint16())))
		},
	},
	0x35: {
		"DEC (HL)",
		func(c *CPU) {
			c.b.ClockedWrite(c.HL.Uint16(), c.decrement(c.b.ClockedRead(c.HL.Uint16())))
		},
	},
	0x36: {
		"LD (HL), d8",
		func(c *CPU) {
			c.b.ClockedWrite(c.HL.Uint16(), c.readOperand())
		},
	},
	0x37: {
		"SCF",
		func(c *CPU) {
			c.setFlags(c.isFlagSet(flagZero), false, false, true)
		},
	},
	0x38: {
		"JR C, r8",
		func(c *CPU) {
			c.jumpRelative(c.isFlagSet(flagCarry))
		},
	},
	0x39: {
		"ADD HL, SP",
		func(c *CPU) {
			c.HL.SetUint16(c.addUint16(c.HL.Uint16(), c.SP))
			c.s.Tick(4)
		},
	},
	0x3A: {
		"LD A, (HL-)",
		func(c *CPU) {
			c.handleOAMCorruption(c.HL.Uint16())
			c.loadMemoryToRegister(&c.A, c.HL.Uint16())
			c.HL.SetUint16(c.HL.Uint16() - 1)
		},
	},
	0x3B: {
		"DEC SP",
		func(c *CPU) {
			c.SP--
			c.s.Tick(4)
		},
	},
	0x3C: {
		"INC A",
		func(c *CPU) {
			c.A = c.increment(c.A)
		},
	},
	0x3D: {
		"DEC A",
		func(c *CPU) {
			c.A = c.decrement(c.A)
		},
	},
	0x3E: {
		"LD A, d8",
		func(c *CPU) {
			c.loadRegister8(&c.A)
		},
	},
	0x3F: {
		"CCF",
		func(c *CPU) {
			c.setFlags(c.isFlagSet(flagZero), false, false, !c.isFlagSet(flagCarry))
		},
	},
	0x40: {
		"LD B, B",
		func(c *CPU) {
			// LD B, B is often used as a debug breakpoint
			if c.Debug {
				c.DebugBreakpoint = true
				c.shouldInt = true
			}
		},
	},
	0x41: {
		"LD B, C",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.B, &c.C)
		},
	},
	0x42: {
		"LD B, D",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.B, &c.D)
		},
	},
	0x43: {
		"LD B, E",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.B, &c.E)
		},
	},
	0x44: {
		"LD B, H",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.B, &c.H)
		},
	},
	0x45: {
		"LD B, L",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.B, &c.L)
		},
	},
	0x46: {
		"LD B, (HL)",
		func(c *CPU) {
			c.loadMemoryToRegister(&c.B, c.HL.Uint16())
		},
	},
	0x47: {
		"LD B, A",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.B, &c.A)
		},
	},
	0x48: {
		"LD C, B",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.C, &c.B)
		},
	},
	0x49: {
		"LD C, C",
		func(c *CPU) {},
	},
	0x4A: {
		"LD C, D",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.C, &c.D)
		},
	},
	0x4B: {
		"LD C, E",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.C, &c.E)
		},
	},
	0x4C: {
		"LD C, H",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.C, &c.H)
		},
	},
	0x4D: {
		"LD C, L",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.C, &c.L)
		},
	},
	0x4E: {
		"LD C, (HL)",
		func(c *CPU) {
			c.loadMemoryToRegister(&c.C, c.HL.Uint16())
		},
	},
	0x4F: {
		"LD C, A",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.C, &c.A)
		},
	},
	0x50: {
		"LD D, B",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.D, &c.B)
		},
	},
	0x51: {
		"LD D, C",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.D, &c.C)
		},
	},
	0x52: {
		"LD D, D",
		func(c *CPU) {},
	},
	0x53: {
		"LD D, E",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.D, &c.E)
		},
	},
	0x54: {
		"LD D, H",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.D, &c.H)
		},
	},
	0x55: {
		"LD D, L",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.D, &c.L)
		},
	},
	0x56: {
		"LD D, (HL)",
		func(c *CPU) {
			c.loadMemoryToRegister(&c.D, c.HL.Uint16())
		},
	},
	0x57: {
		"LD D, A",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.D, &c.A)
		},
	},
	0x58: {
		"LD E, B",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.E, &c.B)
		},
	},
	0x59: {
		"LD E, C",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.E, &c.C)
		},
	},
	0x5A: {
		"LD E, D",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.E, &c.D)
		},
	},
	0x5B: {
		"LD E, E",
		func(c *CPU) {},
	},
	0x5C: {
		"LD E, H",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.E, &c.H)
		},
	},
	0x5D: {
		"LD E, L",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.E, &c.L)
		},
	},
	0x5E: {
		"LD E, (HL)",
		func(c *CPU) {
			c.loadMemoryToRegister(&c.E, c.HL.Uint16())
		},
	},
	0x5F: {
		"LD E, A",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.E, &c.A)
		},
	},
	0x60: {
		"LD H, B",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.H, &c.B)
		},
	},
	0x61: {
		"LD H, C",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.H, &c.C)
		},
	},
	0x62: {
		"LD H, D",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.H, &c.D)
		},
	},
	0x63: {
		"LD H, E",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.H, &c.E)
		},
	},
	0x64: {
		"LD H, H",
		func(c *CPU) {},
	},
	0x65: {
		"LD H, L",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.H, &c.L)
		},
	},
	0x66: {
		"LD H, (HL)",
		func(c *CPU) {
			c.loadMemoryToRegister(&c.H, c.HL.Uint16())
		},
	},
	0x67: {
		"LD H, A",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.H, &c.A)
		},
	},
	0x68: {
		"LD L, B",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.L, &c.B)
		},
	},
	0x69: {
		"LD L, C",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.L, &c.C)
		},
	},
	0x6A: {
		"LD L, D",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.L, &c.D)
		},
	},
	0x6B: {
		"LD L, E",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.L, &c.E)
		},
	},
	0x6C: {
		"LD L, H",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.L, &c.H)
		},
	},
	0x6D: {
		"LD L, L",
		func(c *CPU) {},
	},
	0x6E: {
		"LD L, (HL)",
		func(c *CPU) {
			c.loadMemoryToRegister(&c.L, c.HL.Uint16())
		},
	},
	0x6F: {
		"LD L, A",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.L, &c.A)
		},
	},
	0x70: {
		"LD (HL), B",
		func(c *CPU) {
			c.loadRegisterToMemory(c.B, c.HL.Uint16())
		},
	},
	0x71: {
		"LD (HL), C",
		func(c *CPU) {
			c.loadRegisterToMemory(c.C, c.HL.Uint16())
		},
	},
	0x72: {
		"LD (HL), D",
		func(c *CPU) {
			c.loadRegisterToMemory(c.D, c.HL.Uint16())
		},
	},
	0x73: {
		"LD (HL), E",
		func(c *CPU) {
			c.loadRegisterToMemory(c.E, c.HL.Uint16())
		},
	},
	0x74: {
		"LD (HL), H",
		func(c *CPU) {
			c.loadRegisterToMemory(c.H, c.HL.Uint16())
		},
	},
	0x75: {
		"LD (HL), L",
		func(c *CPU) {
			c.loadRegisterToMemory(c.L, c.HL.Uint16())
		},
	},
	0x76: {
		"HALT",
		func(c *CPU) {
			if c.b.InterruptsEnabled() {
				//panic("halt with interrupts enabled")
				c.skipHALT()
			} else {
				if c.b.HasInterrupts() {
					c.doHALTBug()
				} else {
					switch c.b.Model() {
					case types.MGB: // TODO handle MGB oam HALT weirdness
						c.DebugBreakpoint = true
					default:
						c.skipHALT()
					}
				}
			}
		},
	},
	0x77: {
		"LD (HL), A",
		func(c *CPU) {
			c.loadRegisterToMemory(c.A, c.HL.Uint16())
		},
	},
	0x78: {
		"LD A, B",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.A, &c.B)
		},
	},
	0x79: {
		"LD A, C",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.A, &c.C)
		},
	},
	0x7A: {
		"LD A, D",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.A, &c.D)
		},
	},
	0x7B: {
		"LD A, E",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.A, &c.E)
		},
	},
	0x7C: {
		"LD A, H",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.A, &c.H)
		},
	},
	0x7D: {
		"LD A, L",
		func(c *CPU) {
			c.loadRegisterToRegister(&c.A, &c.L)
		},
	},
	0x7E: {
		"LD A, (HL)",
		func(c *CPU) {
			c.loadMemoryToRegister(&c.A, c.HL.Uint16())
		},
	},
	0x7F: {
		"LD A, A",
		func(c *CPU) {},
	},
	0x80: {
		"ADD A, B",
		func(c *CPU) {
			c.add(c.B, false)
		},
	},
	0x81: {
		"ADD A, C",
		func(c *CPU) {
			c.add(c.C, false)
		},
	},
	0x82: {
		"ADD A, D",
		func(c *CPU) {
			c.add(c.D, false)
		},
	},
	0x83: {
		"ADD A, E",
		func(c *CPU) {
			c.add(c.E, false)
		},
	},
	0x84: {
		"ADD A, H",
		func(c *CPU) {
			c.add(c.H, false)
		},
	},
	0x85: {
		"ADD A, L",
		func(c *CPU) {
			c.add(c.L, false)
		},
	},
	0x86: {
		"ADD A, (HL)",
		func(c *CPU) {
			c.add(c.b.ClockedRead(c.HL.Uint16()), false)
		},
	},
	0x87: {
		"ADD A, A",
		func(c *CPU) {
			c.add(c.A, false)
		},
	},
	0x88: {
		"ADC A, B",
		func(c *CPU) {
			c.add(c.B, true)
		},
	},
	0x89: {
		"ADC A, C",
		func(c *CPU) {
			c.add(c.C, true)
		},
	},
	0x8A: {
		"ADC A, D",
		func(c *CPU) {
			c.add(c.D, true)
		},
	},
	0x8B: {
		"ADC A, E",
		func(c *CPU) {
			c.add(c.E, true)
		},
	},
	0x8C: {
		"ADC A, H",
		func(c *CPU) {
			c.add(c.H, true)
		},
	},
	0x8D: {
		"ADC A, L",
		func(c *CPU) {
			c.add(c.L, true)
		},
	},
	0x8E: {
		"ADC A, (HL)",
		func(c *CPU) {
			c.add(c.b.ClockedRead(c.HL.Uint16()), true)
		},
	},
	0x8F: {
		"ADC A, A",
		func(c *CPU) {
			c.add(c.A, true)
		},
	},
	0x90: {
		"SUB B",
		func(c *CPU) {
			c.sub(c.B, false)
		},
	},
	0x91: {
		"SUB C",
		func(c *CPU) {
			c.sub(c.C, false)
		},
	},
	0x92: {
		"SUB D",
		func(c *CPU) {
			c.sub(c.D, false)
		},
	},
	0x93: {
		"SUB E",
		func(c *CPU) {
			c.sub(c.E, false)
		},
	},
	0x94: {
		"SUB H",
		func(c *CPU) {
			c.sub(c.H, false)
		},
	},
	0x95: {
		"SUB L",
		func(c *CPU) {
			c.sub(c.L, false)
		},
	},
	0x96: {
		"SUB (HL)",
		func(c *CPU) {
			c.sub(c.b.ClockedRead(c.HL.Uint16()), false)
		},
	},
	0x97: {
		"SUB A",
		func(c *CPU) {
			c.sub(c.A, false)
		},
	},
	0x98: {
		"SBC B",
		func(c *CPU) {
			c.sub(c.B, true)
		},
	},
	0x99: {
		"SBC C",
		func(c *CPU) {
			c.sub(c.C, true)
		},
	},
	0x9A: {
		"SBC D",
		func(c *CPU) {
			c.sub(c.D, true)
		},
	},
	0x9B: {
		"SBC E",
		func(c *CPU) {
			c.sub(c.E, true)
		},
	},
	0x9C: {
		"SBC H",
		func(c *CPU) {
			c.sub(c.H, true)
		},
	},
	0x9D: {
		"SBC L",
		func(c *CPU) {
			c.sub(c.L, true)
		},
	},
	0x9E: {
		"SBC (HL)",
		func(c *CPU) {
			c.sub(c.b.ClockedRead(c.HL.Uint16()), true)
		},
	},
	0x9F: {
		"SBC A",
		func(c *CPU) {
			c.sub(c.A, true)
		},
	},
	0xA0: {
		"AND B",
		func(c *CPU) {
			c.and(c.B)
		},
	},
	0xA1: {
		"AND C",
		func(c *CPU) {
			c.and(c.C)
		},
	},
	0xA2: {
		"AND D",
		func(c *CPU) {
			c.and(c.D)
		},
	},
	0xA3: {
		"AND E",
		func(c *CPU) {
			c.and(c.E)
		},
	},
	0xA4: {
		"AND H",
		func(c *CPU) {
			c.and(c.H)
		},
	},
	0xA5: {
		"AND L",
		func(c *CPU) {
			c.and(c.L)
		},
	},
	0xA6: {
		"AND (HL)",
		func(c *CPU) {
			c.and(c.b.ClockedRead(c.HL.Uint16()))
		},
	},
	0xA7: {
		"AND A",
		func(c *CPU) {
			c.and(c.A)
		},
	},
	0xA8: {
		"XOR B",
		func(c *CPU) {
			c.xor(c.B)
		},
	},
	0xA9: {
		"XOR C",
		func(c *CPU) {
			c.xor(c.C)
		},
	},
	0xAA: {
		"XOR D",
		func(c *CPU) {
			c.xor(c.D)
		},
	},
	0xAB: {
		"XOR E",
		func(c *CPU) {
			c.xor(c.E)
		},
	},
	0xAC: {
		"XOR H",
		func(c *CPU) {
			c.xor(c.H)
		},
	},
	0xAD: {
		"XOR L",
		func(c *CPU) {
			c.xor(c.L)
		},
	},
	0xAE: {
		"XOR (HL)",
		func(c *CPU) {
			c.xor(c.b.ClockedRead(c.HL.Uint16()))
		},
	},
	0xAF: {
		"XOR A",
		func(c *CPU) {
			c.xor(c.A)
		},
	},
	0xB0: {
		"OR B",
		func(c *CPU) {
			c.or(c.B)
		},
	},
	0xB1: {
		"OR C",
		func(c *CPU) {
			c.or(c.C)
		},
	},
	0xB2: {
		"OR D",
		func(c *CPU) {
			c.or(c.D)
		},
	},
	0xB3: {
		"OR E",
		func(c *CPU) {
			c.or(c.E)
		},
	},
	0xB4: {
		"OR H",
		func(c *CPU) {
			c.or(c.H)
		},
	},
	0xB5: {
		"OR L",
		func(c *CPU) {
			c.or(c.L)
		},
	},
	0xB6: {
		"OR (HL)",
		func(c *CPU) {
			c.or(c.b.ClockedRead(c.HL.Uint16()))
		},
	},
	0xB7: {
		"OR A",
		func(c *CPU) {
			c.or(c.A)
		},
	},
	0xB8: {
		"CP B",
		func(c *CPU) {
			c.compare(c.B)
		},
	},
	0xB9: {
		"CP C",
		func(c *CPU) {
			c.compare(c.C)
		},
	},
	0xBA: {
		"CP D",
		func(c *CPU) {
			c.compare(c.D)
		},
	},
	0xBB: {
		"CP E",
		func(c *CPU) {
			c.compare(c.E)
		},
	},
	0xBC: {
		"CP H",
		func(c *CPU) {
			c.compare(c.H)
		},
	},
	0xBD: {
		"CP L",
		func(c *CPU) {
			c.compare(c.L)
		},
	},
	0xBE: {
		"CP (HL)",
		func(c *CPU) {
			c.compare(c.b.ClockedRead(c.HL.Uint16()))
		},
	},
	0xBF: {
		"CP A",
		func(c *CPU) {
			c.compare(c.A)
		},
	},
	0xC0: {
		"RET NZ",
		func(c *CPU) {
			c.s.Tick(4)
			c.ret(!c.isFlagSet(flagZero))
		},
	},
	0xC1: {
		"POP BC",
		func(c *CPU) {
			c.popNN(&c.B, &c.C)
		},
	},
	0xC2: {
		"JP NZ, a16",
		func(c *CPU) {
			c.jumpAbsolute(!c.isFlagSet(flagZero))
		},
	},
	0xC3: {
		"JP a16",
		func(c *CPU) {
			c.jumpAbsolute(true)
		},
	},
	0xC4: {
		"CALL NZ, a16",
		func(c *CPU) {
			c.call(!c.isFlagSet(flagZero))
		},
	},
	0xC5: {
		"PUSH BC",
		func(c *CPU) {
			c.pushNN(c.B, c.C)
		},
	},
	0xC6: {
		"ADD A, d8",
		func(c *CPU) {
			c.add(c.readOperand(), false)
		},
	},
	0xC7: {
		"RST 0",
		func(c *CPU) {
			c.rst(0x00)
		},
	},
	0xC8: {
		"RET Z",
		func(c *CPU) {
			c.s.Tick(4)
			c.ret(c.isFlagSet(flagZero))
		},
	},
	0xC9: {
		"RET",
		func(c *CPU) {
			c.ret(true)
		},
	},
	0xCA: {
		"JP Z, nn",
		func(c *CPU) {
			c.jumpAbsolute(c.isFlagSet(flagZero))
		},
	},
	0xCB: {
		"CB Prefix",
		func(c *CPU) {
			c.decodeCB(c.readOperand())
		},
	},
	0xCC: {
		"CALL Z, nn",
		func(c *CPU) {
			c.call(c.isFlagSet(flagZero))
		},
	},
	0xCD: {
		"CALL nn",
		func(c *CPU) {
			c.call(true)
		},
	},
	0xCE: {
		"ADC A, d8",
		func(c *CPU) {
			c.add(c.readOperand(), true)
		},
	},
	0xCF: {
		"RST 1",
		func(c *CPU) {
			c.rst(0x08)
		},
	},
	0xD0: {
		"RET NC",
		func(c *CPU) {
			c.s.Tick(4)
			c.ret(!c.isFlagSet(flagCarry))
		},
	},
	0xD1: {
		"POP DE",
		func(c *CPU) {
			c.popNN(&c.D, &c.E)
		},
	},
	0xD2: {
		"JP NC, a16",
		func(c *CPU) {
			c.jumpAbsolute(!c.isFlagSet(flagCarry))
		},
	},
	0xD3: disallowedOpcode(0xD3),
	0xD4: {
		"CALL NC, a16",
		func(c *CPU) {
			c.call(!c.isFlagSet(flagCarry))
		},
	},
	0xD5: {
		"PUSH DE",
		func(c *CPU) {
			c.pushNN(c.D, c.E)
		},
	},
	0xD6: {
		"SUB d8",
		func(c *CPU) {
			c.sub(c.readOperand(), false)
		},
	},
	0xD7: {
		"RST 2",
		func(c *CPU) {
			c.rst(0x10)
		},
	},
	0xD8: {
		"RET C",
		func(c *CPU) {
			c.s.Tick(4)
			c.ret(c.isFlagSet(flagCarry))
		},
	},
	0xD9: {
		"RETI",
		func(c *CPU) {
			c.b.EnableInterrupts()
			c.ret(true)
		},
	},
	0xDA: {
		"JP C, nn",
		func(c *CPU) {
			c.jumpAbsolute(c.isFlagSet(flagCarry))
		},
	},
	0xDB: disallowedOpcode(0xDB),
	0xDC: {
		"CALL C, nn",
		func(c *CPU) {
			c.call(c.isFlagSet(flagCarry))
		},
	},
	0xDD: disallowedOpcode(0xDD),
	0xDE: {
		"SBC A, d8",
		func(c *CPU) {
			c.sub(c.readOperand(), true)
		},
	},
	0xDF: {
		"RST 3",
		func(c *CPU) {
			c.rst(0x18)
		},
	},
	0xE0: {
		"LDH (a8), A",
		func(c *CPU) {
			c.loadRegisterToHardware(c.A, c.readOperand())
		},
	},
	0xE1: {
		"POP HL",
		func(c *CPU) {
			c.popNN(&c.H, &c.L)
		},
	},
	0xE2: {
		"LD (C), A",
		func(c *CPU) {
			c.loadRegisterToHardware(c.A, c.C)
		},
	},
	0xE3: disallowedOpcode(0xE3),
	0xE4: disallowedOpcode(0xE4),
	0xE5: {
		"PUSH HL",
		func(c *CPU) {
			c.pushNN(c.H, c.L)
		},
	},
	0xE6: {
		"AND d8",
		func(c *CPU) {
			c.and(c.readOperand())
		},
	},
	0xE7: {
		"RST 4",
		func(c *CPU) {
			c.rst(0x20)
		},
	},
	0xE8: {
		"ADD SP, r8",
		func(c *CPU) {
			c.SP = c.addSPSigned()
			c.s.Tick(4)
		},
	},
	0xE9: {
		"JP HL",
		func(c *CPU) {
			c.PC = c.HL.Uint16()
		},
	},
	0xEA: {
		"LD (a16), A",
		func(c *CPU) {
			low := c.readOperand()
			high := c.readOperand()
			c.loadRegisterToMemory(c.A, uint16(high)<<8|uint16(low))
		},
	},
	0xEB: disallowedOpcode(0xEB),
	0xEC: disallowedOpcode(0xEC),
	0xED: disallowedOpcode(0xED),
	0xEE: {
		"XOR d8",
		func(c *CPU) {
			c.xor(c.readOperand())
		},
	},
	0xEF: {
		"RST 5",
		func(c *CPU) {
			c.rst(0x28)
		},
	},
	0xF0: {
		"LDH A, (a8)",
		func(c *CPU) {
			address := uint16(0xff00) + uint16(c.readOperand())
			c.loadMemoryToRegister(&c.A, address)
		},
	},
	0xF1: {
		"POP AF",
		func(c *CPU) {
			c.popNN(&c.A, &c.F)
			c.F &= 0xF0
		},
	},
	0xF2: {
		"LD A, (C)",
		func(c *CPU) {
			c.loadMemoryToRegister(&c.A, uint16(0xFF00)+uint16(c.C))
		},
	},
	0xF3: {
		"DI",
		func(c *CPU) {
			c.b.DisableInterrupts()
		},
	},
	0xF4: disallowedOpcode(0xF4),
	0xF5: {
		"PUSH AF",
		func(c *CPU) {
			c.pushNN(c.A, c.F)
		},
	},
	0xF6: {
		"OR d8",
		func(c *CPU) {
			c.or(c.readOperand())
		},
	},
	0xF7: {
		"RST 6",
		func(c *CPU) {
			c.rst(0x30)
		},
	},
	0xF8: {
		"LD HL, SP+r8",
		func(c *CPU) {
			c.HL.SetUint16(c.addSPSigned())
		},
	},
	0xF9: {
		"LD SP, HL",
		func(c *CPU) {
			c.SP = c.HL.Uint16()
			c.s.Tick(4)
		},
	},
	0xFA: {
		"LD A, (a16)",
		func(c *CPU) {
			low := c.readOperand()
			high := c.readOperand()
			c.loadMemoryToRegister(&c.A, uint16(high)<<8|uint16(low))
		},
	},
	0xFB: {
		"EI",
		func(c *CPU) {
			// handle ei_delay_halt (see https://github.com/LIJI32/SameSuite/blob/master/interrupt/ei_delay_halt.asm)
			if c.b.Get(c.PC) == 0x76 && c.b.Get(types.IE)&c.b.Get(types.IF) != 0 {
				// if an EI instruction is directly succeeded by a HALT instruction,
				// and there is a pending interrupt, the interrupt will be serviced
				// first, before the interrupt returns control to the HALT instruction,
				// effectively delaying the execution of HALT by one instruction.
				c.s.ScheduleEvent(scheduler.EIHaltDelay, 4)
			} else {
				c.s.ScheduleEvent(scheduler.EIPending, 4)
			}
		},
	},
	0xFC: disallowedOpcode(0xFC),
	0xFD: disallowedOpcode(0xFD),
	0xFE: {
		"CP d8",
		func(c *CPU) {
			c.compare(c.readOperand())
		},
	},
	0xFF: {
		"RST 7",
		func(c *CPU) {
			c.rst(0x38)
		},
	},
}

func (c *CPU) decodeCB(instr byte) {
	var memWrite = false
	var dest *uint8
	switch instr & 0x7 {
	case 0:
		dest = &c.B
	case 1:
		dest = &c.C
	case 2:
		dest = &c.D
	case 3:
		dest = &c.E
	case 4:
		dest = &c.H
	case 5:
		dest = &c.L
	case 6:
		// we only set the value of dest here, we still need to write the
		// new value back to the bus
		val := c.b.ClockedRead(c.HL.Uint16())
		dest = &val
		memWrite = true
	case 7:
		dest = &c.A
	}

	switch instr >> 6 & 0x3 {
	case 0:
		// 0b00 000 000
		//   ^^ ^^^ ^^^
		//   ma op  des
		//
		switch instr >> 3 & 0x7 {
		case 0: // RLC
			*dest = c.rotateLeftCarry(*dest)
		case 1: // RRC
			*dest = c.rotateRightCarry(*dest)
		case 2: // RL
			*dest = c.rotateLeftThroughCarry(*dest)
		case 3: // RR
			*dest = c.rotateRightThroughCarry(*dest)
		case 4: // SLA
			*dest = c.shiftLeftArithmetic(*dest)
		case 5: // SRA
			*dest = c.shiftRightArithmetic(*dest)
		case 6: // SWAP
			*dest = c.swap(*dest)
		case 7: // SRL
			*dest = c.shiftRightLogical(*dest)
		}
	case 1:
		// BIT
		// 0b00 000 000
		//   ^^ ^^^ ^^^
		//      bit des
		c.testBit(*dest, 1<<(instr>>3&0x7))
		memWrite = false
	case 2:
		// RES
		*dest = utils.Reset(*dest, 1<<(instr>>3&0x7))
	case 3:
		// SET
		*dest = utils.Set(*dest, 1<<(instr>>3&0x7))
	}

	if memWrite {
		// write new value back to the bus
		c.b.ClockedWrite(c.HL.Uint16(), *dest)
	}
}

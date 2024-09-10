package cpu

import (
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
)

type Instruction struct {
	name string     // name of the instruction
	fn   func(*CPU) // fn called when executing the instruction
}

// disallowedOpcode creates an instruction that will panic if executed.
func disallowedOpcode(opcode uint8) Instruction {
	return Instruction{
		name: fmt.Sprintf("disallowed opcode %X", opcode),
		fn:   func(cpu *CPU) {},
	}
}

// InstructionSet holds the first 256 instructions.
var InstructionSet = [256]Instruction{
	0x00: {"NOP", func(c *CPU) {}},
	0x01: {"LD BC,d16", func(c *CPU) { *c.BC[1] = c.readOperand(); *c.BC[0] = c.readOperand() }},
	0x02: {"LD (BC), A", func(c *CPU) { c.b.ClockedWrite(c.BC.Uint16(), c.A) }},
	0x03: {"INC BC", func(c *CPU) { c.BC.SetUint16(c.BC.Uint16() + 1); c.s.Tick(4) }},
	0x04: {"INC B", func(c *CPU) { h := c.B&0xf == 0xf; c.B++; c.setFlags(c.B == 0, false, h, c.isFlagSet(flagCarry)) }},
	0x05: {"DEC B", func(c *CPU) { h := c.B&0xf == 0; c.B--; c.setFlags(c.B == 0, true, h, c.isFlagSet(flagCarry)) }},
	0x06: {"LD B, d8", func(c *CPU) { c.B = c.readOperand() }},
	0x07: {"RLCA", func(c *CPU) { n := c.A & 0x80; c.A = c.A<<1 | n>>7; c.setFlags(false, false, false, n > 0) }},
	0x08: {"LD (a16), SP", func(c *CPU) {
		low := c.readOperand()
		high := c.readOperand()

		address := uint16(high)<<8 | uint16(low)
		c.b.ClockedWrite(address, uint8(c.SP&0xFF))
		c.b.ClockedWrite(address+1, uint8(c.SP>>8))
	}},
	0x09: {"ADD HL, BC", func(c *CPU) {
		s := int32(c.HL.Uint16()) + int32(c.BC.Uint16())
		c.setFlags(c.isFlagSet(flagZero), false, (c.HL.Uint16()&0xfff)+(c.BC.Uint16()&0xfff) > 0xfff, s > 0xffff)
		c.HL.SetUint16(uint16(s))
		c.s.Tick(4)
	}},
	0x0A: {"LD A, (BC)", func(c *CPU) { c.A = c.b.ClockedRead(c.BC.Uint16()) }},
	0x0B: {"DEC BC", func(c *CPU) { c.BC.SetUint16(c.BC.Uint16() - 1); c.s.Tick(4) }},
	0x0C: {"INC C", func(c *CPU) { h := c.C&0xf == 0xf; c.C++; c.setFlags(c.C == 0, false, h, c.isFlagSet(flagCarry)) }},
	0x0D: {"DEC C", func(c *CPU) { h := c.C&0xf == 0; c.C--; c.setFlags(c.C == 0, true, h, c.isFlagSet(flagCarry)) }},
	0x0E: {"LD C, d8", func(c *CPU) { c.C = c.readOperand() }},
	0x0F: {"RRCA", func(c *CPU) { n := c.A & 1; c.A = c.A>>1 | n<<7; c.setFlags(false, false, false, n > 0) }},
	0x10: {"STOP", func(c *CPU) {
		// reset div clock
		c.s.SysClockReset()

		// if there's no pending interrupt then STOP becomes a 2-byte opcode
		if !c.b.HasInterrupts() {
			c.PC++
		}

		// are we in gbc mode (STOP is alternatively used for speed-switching)
		if c.b.Model() == types.CGB0 || c.b.Model() == types.CGBABC &&
			c.b.Get(types.KEY1)&types.Bit0 == types.Bit0 {
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
	}},
	0x11: {"LD DE, d16", func(c *CPU) { *c.DE[1] = c.readOperand(); *c.DE[0] = c.readOperand() }},
	0x12: {"LD (DE), A", func(c *CPU) { c.b.ClockedWrite(c.DE.Uint16(), c.A) }},
	0x13: {"INC DE", func(c *CPU) { c.DE.SetUint16(c.DE.Uint16() + 1); c.s.Tick(4) }},
	0x14: {"INC D", func(c *CPU) { h := c.D&0xf == 0xf; c.D++; c.setFlags(c.D == 0, false, h, c.isFlagSet(flagCarry)) }},
	0x15: {"DEC D", func(c *CPU) { h := c.D&0xf == 0; c.D--; c.setFlags(c.D == 0, true, h, c.isFlagSet(flagCarry)) }},
	0x16: {"LD D, d8", func(c *CPU) { c.D = c.readOperand() }},
	0x17: {"RLA", func(c *CPU) { r := c.A<<1 | c.F&flagCarry>>4; c.F = c.A & 0x80 >> 3; c.A = r }},
	0x18: {"JR r8", func(c *CPU) { c.PC = uint16(int16(c.PC) + int16(int8(c.readOperand()))); c.s.Tick(4) }},
	0x19: {"ADD HL, DE", func(c *CPU) {
		s := int32(c.HL.Uint16()) + int32(c.DE.Uint16())
		c.setFlags(c.isFlagSet(flagZero), false, (c.HL.Uint16()&0xfff)+(c.DE.Uint16()&0xfff) > 0xfff, s > 0xffff)
		c.HL.SetUint16(uint16(s))
		c.s.Tick(4)
	}},
	0x1A: {"LD A, (DE)", func(c *CPU) { c.A = c.b.ClockedRead(c.DE.Uint16()) }},
	0x1B: {"DEC DE", func(c *CPU) { c.DE.SetUint16(c.DE.Uint16() - 1); c.s.Tick(4) }},
	0x1C: {"INC E", func(c *CPU) { h := c.E&0xf == 0xf; c.E++; c.setFlags(c.E == 0, false, h, c.isFlagSet(flagCarry)) }},
	0x1D: {"DEC E", func(c *CPU) { h := c.E&0xf == 0; c.E--; c.setFlags(c.E == 0, true, h, c.isFlagSet(flagCarry)) }},
	0x1E: {"LD E, d8", func(c *CPU) { c.E = c.readOperand() }},
	0x1F: {"RRA", func(c *CPU) { r := c.A>>1 | c.F&flagCarry<<3; c.F = c.A & 1 << 4; c.A = r }},
	0x20: {"JR NZ, r8", func(c *CPU) {
		if !c.isFlagSet(flagZero) {
			c.PC = uint16(int16(c.PC) + int16(int8(c.readOperand())))
			c.s.Tick(4)
		} else {
			c.s.Tick(4)
			c.PC++
		}
	}},
	0x21: {"LD HL, d16", func(c *CPU) { *c.HL[1] = c.readOperand(); *c.HL[0] = c.readOperand() }},
	0x22: {"LD (HL+), A", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.A); c.HL.SetUint16(c.HL.Uint16() + 1) }},
	0x23: {"INC HL", func(c *CPU) { c.HL.SetUint16(c.HL.Uint16() + 1); c.s.Tick(4) }},
	0x24: {"INC H", func(c *CPU) { h := c.H&0xf == 0xf; c.H++; c.setFlags(c.H == 0, false, h, c.isFlagSet(flagCarry)) }},
	0x25: {"DEC H", func(c *CPU) { h := c.H&0xf == 0; c.H--; c.setFlags(c.H == 0, true, h, c.isFlagSet(flagCarry)) }},
	0x26: {"LD H, d8", func(c *CPU) { c.H = c.readOperand() }},
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
	0x28: {"JR Z, r8", func(c *CPU) {
		if c.isFlagSet(flagZero) {
			c.PC = uint16(int16(c.PC) + int16(int8(c.readOperand())))
			c.s.Tick(4)
		} else {
			c.s.Tick(4)
			c.PC++
		}
	}},
	0x29: {"ADD HL, HL", func(c *CPU) {
		s := int32(c.HL.Uint16()) + int32(c.HL.Uint16())
		c.setFlags(c.isFlagSet(flagZero), false, (c.HL.Uint16()&0xfff)+(c.HL.Uint16()&0xfff) > 0xfff, s > 0xffff)
		c.HL.SetUint16(uint16(s))
		c.s.Tick(4)
	}},
	0x2A: {"LD A, (HL+)", func(c *CPU) {
		c.A = c.b.ClockedRead(c.HL.Uint16())
		c.HL.SetUint16(c.HL.Uint16() + 1)
	}},
	0x2B: {"DEC HL", func(c *CPU) { c.HL.SetUint16(c.HL.Uint16() - 1); c.s.Tick(4) }},
	0x2C: {"INC L", func(c *CPU) { h := c.L&0xf == 0xf; c.L++; c.setFlags(c.L == 0, false, h, c.isFlagSet(flagCarry)) }},
	0x2D: {"DEC L", func(c *CPU) { h := c.L&0xf == 0; c.L--; c.setFlags(c.L == 0, true, h, c.isFlagSet(flagCarry)) }},
	0x2E: {"LD L, d8", func(c *CPU) { c.L = c.readOperand() }},
	0x2F: {"CPL", func(c *CPU) { c.A = 0xFF ^ c.A; c.setFlags(c.isFlagSet(flagZero), true, true, c.isFlagSet(flagCarry)) }},
	0x30: {"JR NC, r8", func(c *CPU) {
		if !c.isFlagSet(flagCarry) {
			c.PC = uint16(int16(c.PC) + int16(int8(c.readOperand())))
			c.s.Tick(4)
		} else {
			c.s.Tick(4)
			c.PC++
		}
	}},
	0x31: {"LD SP, d16", func(c *CPU) { c.SP = uint16(c.readOperand()) | uint16(c.readOperand())<<8 }},
	0x32: {"LD (HL-), A", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.A); c.HL.SetUint16(c.HL.Uint16() - 1) }},
	0x33: {"INC SP", func(c *CPU) { c.SP++; c.s.Tick(4) }},
	0x34: {"INC (HL)", func(c *CPU) {
		v := c.b.ClockedRead(c.HL.Uint16())
		r := v + 1
		c.b.ClockedWrite(c.HL.Uint16(), r)
		c.setFlags(r == 0, false, v&0xf == 0xf, c.isFlagSet(flagCarry))
	}},
	0x35: {"DEC (HL)", func(c *CPU) {
		v := c.b.ClockedRead(c.HL.Uint16())
		r := v - 1
		c.b.ClockedWrite(c.HL.Uint16(), r)
		c.setFlags(r == 0, true, v&0xf == 0, c.isFlagSet(flagCarry))
	}},
	0x36: {"LD (HL), d8", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.readOperand()) }},
	0x37: {"SCF", func(c *CPU) { c.setFlags(c.isFlagSet(flagZero), false, false, true) }},
	0x38: {"JR C, r8", func(c *CPU) {
		if c.isFlagSet(flagCarry) {
			c.PC = uint16(int16(c.PC) + int16(int8(c.readOperand())))
			c.s.Tick(4)
		} else {
			c.s.Tick(4)
			c.PC++
		}
	}},
	0x39: {"ADD HL, SP", func(c *CPU) {
		s := int32(c.HL.Uint16()) + int32(c.SP)
		c.setFlags(c.isFlagSet(flagZero), false, (c.HL.Uint16()&0xfff)+(c.SP&0xfff) > 0xfff, s > 0xffff)
		c.HL.SetUint16(uint16(s))
		c.s.Tick(4)
	}},
	0x3A: {"LD A, (HL-)", func(c *CPU) {
		c.A = c.b.ClockedRead(c.HL.Uint16())
		c.HL.SetUint16(c.HL.Uint16() - 1)
	}},
	0x3B: {"DEC SP", func(c *CPU) { c.SP--; c.s.Tick(4) }},
	0x3C: {"INC A", func(c *CPU) { h := c.A&0xf == 0xf; c.A++; c.setFlags(c.A == 0, false, h, c.isFlagSet(flagCarry)) }},
	0x3D: {"DEC A", func(c *CPU) { h := c.A&0xf == 0; c.A--; c.setFlags(c.A == 0, true, h, c.isFlagSet(flagCarry)) }},
	0x3E: {"LD A, d8", func(c *CPU) { c.A = c.readOperand() }},
	0x3F: {"CCF", func(c *CPU) { c.setFlags(c.isFlagSet(flagZero), false, false, !c.isFlagSet(flagCarry)) }},
	0x40: {"LD B, B", func(c *CPU) { c.DebugBreakpoint = c.Debug }},
	0x41: {"LD B, C", func(c *CPU) { c.B = c.C }},
	0x42: {"LD B, D", func(c *CPU) { c.B = c.D }},
	0x43: {"LD B, E", func(c *CPU) { c.B = c.E }},
	0x44: {"LD B, H", func(c *CPU) { c.B = c.H }},
	0x45: {"LD B, L", func(c *CPU) { c.B = c.L }},
	0x46: {"LD B, (HL)", func(c *CPU) { c.B = c.b.ClockedRead(c.HL.Uint16()) }},
	0x47: {"LD B, A", func(c *CPU) { c.B = c.A }},
	0x48: {"LD C, B", func(c *CPU) { c.C = c.B }},
	0x49: {"LD C, C", func(c *CPU) {}},
	0x4A: {"LD C, D", func(c *CPU) { c.C = c.D }},
	0x4B: {"LD C, E", func(c *CPU) { c.C = c.E }},
	0x4C: {"LD C, H", func(c *CPU) { c.C = c.H }},
	0x4D: {"LD C, L", func(c *CPU) { c.C = c.L }},
	0x4E: {"LD C, (HL)", func(c *CPU) { c.C = c.b.ClockedRead(c.HL.Uint16()) }},
	0x4F: {"LD C, A", func(c *CPU) { c.C = c.A }},
	0x50: {"LD D, B", func(c *CPU) { c.D = c.B }},
	0x51: {"LD D, C", func(c *CPU) { c.D = c.C }},
	0x52: {"LD D, D", func(c *CPU) {}},
	0x53: {"LD D, E", func(c *CPU) { c.D = c.E }},
	0x54: {"LD D, H", func(c *CPU) { c.D = c.H }},
	0x55: {"LD D, L", func(c *CPU) { c.D = c.L }},
	0x56: {"LD D, (HL)", func(c *CPU) { c.D = c.b.ClockedRead(c.HL.Uint16()) }},
	0x57: {"LD D, A", func(c *CPU) { c.D = c.A }},
	0x58: {"LD E, B", func(c *CPU) { c.E = c.B }},
	0x59: {"LD E, C", func(c *CPU) { c.E = c.C }},
	0x5A: {"LD E, D", func(c *CPU) { c.E = c.D }},
	0x5B: {"LD E, E", func(c *CPU) {}},
	0x5C: {"LD E, H", func(c *CPU) { c.E = c.H }},
	0x5D: {"LD E, L", func(c *CPU) { c.E = c.L }},
	0x5E: {"LD E, (HL)", func(c *CPU) { c.E = c.b.ClockedRead(c.HL.Uint16()) }},
	0x5F: {"LD E, A", func(c *CPU) { c.E = c.A }},
	0x60: {"LD H, B", func(c *CPU) { c.H = c.B }},
	0x61: {"LD H, C", func(c *CPU) { c.H = c.C }},
	0x62: {"LD H, D", func(c *CPU) { c.H = c.D }},
	0x63: {"LD H, E", func(c *CPU) { c.H = c.E }},
	0x64: {"LD H, H", func(c *CPU) {}},
	0x65: {"LD H, L", func(c *CPU) { c.H = c.L }},
	0x66: {"LD H, (HL)", func(c *CPU) { c.H = c.b.ClockedRead(c.HL.Uint16()) }},
	0x67: {"LD H, A", func(c *CPU) { c.H = c.A }},
	0x68: {"LD L, B", func(c *CPU) { c.L = c.B }},
	0x69: {"LD L, C", func(c *CPU) { c.L = c.C }},
	0x6A: {"LD L, D", func(c *CPU) { c.L = c.D }},
	0x6B: {"LD L, E", func(c *CPU) { c.L = c.E }},
	0x6C: {"LD L, H", func(c *CPU) { c.L = c.H }},
	0x6D: {"LD L, L", func(c *CPU) {}},
	0x6E: {"LD L, (HL)", func(c *CPU) { c.L = c.b.ClockedRead(c.HL.Uint16()) }},
	0x6F: {"LD L, A", func(c *CPU) { c.L = c.A }},
	0x70: {"LD (HL), B", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.B) }},
	0x71: {"LD (HL), C", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.C) }},
	0x72: {"LD (HL), D", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.D) }},
	0x73: {"LD (HL), E", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.E) }},
	0x74: {"LD (HL), H", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.H) }},
	0x75: {"LD (HL), L", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.L) }},
	0x76: {"HALT", func(c *CPU) {
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
	}},
	0x77: {"LD (HL), A", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.A) }},
	0x78: {"LD A, B", func(c *CPU) { c.A = c.B }},
	0x79: {"LD A, C", func(c *CPU) { c.A = c.C }},
	0x7A: {"LD A, D", func(c *CPU) { c.A = c.D }},
	0x7B: {"LD A, E", func(c *CPU) { c.A = c.E }},
	0x7C: {"LD A, H", func(c *CPU) { c.A = c.H }},
	0x7D: {"LD A, L", func(c *CPU) { c.A = c.L }},
	0x7E: {"LD A, (HL)", func(c *CPU) { c.A = c.b.ClockedRead(c.HL.Uint16()) }},
	0x7F: {"LD A, A", func(c *CPU) {}},
	0x80: {"ADD A, B", func(c *CPU) {
		s := uint16(c.A) + uint16(c.B)
		c.setFlags(s&0xff == 0, false, (c.A&0xf)+(c.B&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x81: {"ADD A, C", func(c *CPU) {
		s := uint16(c.A) + uint16(c.C)
		c.setFlags(s&0xff == 0, false, (c.A&0xf)+(c.C&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x82: {"ADD A, D", func(c *CPU) {
		s := uint16(c.A) + uint16(c.D)
		c.setFlags(s&0xff == 0, false, (c.A&0xf)+(c.D&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x83: {"ADD A, E", func(c *CPU) {
		s := uint16(c.A) + uint16(c.E)
		c.setFlags(s&0xff == 0, false, (c.A&0xf)+(c.E&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x84: {"ADD A, H", func(c *CPU) {
		s := uint16(c.A) + uint16(c.H)
		c.setFlags(s&0xff == 0, false, (c.A&0xf)+(c.H&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x85: {"ADD A, L", func(c *CPU) {
		s := uint16(c.A) + uint16(c.L)
		c.setFlags(s&0xff == 0, false, (c.A&0xf)+(c.L&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x86: {"ADD A, (HL)", func(c *CPU) {
		value := c.b.ClockedRead(c.HL.Uint16())
		s := uint16(c.A) + uint16(value)
		c.setFlags(s&0xff == 0, false, (c.A&0xf)+(value&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x87: {"ADD A, A", func(c *CPU) {
		s := uint16(c.A) + uint16(c.A)
		c.setFlags(s&0xff == 0, false, (c.A&0xf)+(c.A&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x88: {"ADC A, B", func(c *CPU) {
		s := uint16(c.A) + uint16(c.B) + (uint16(c.F&flagCarry) >> 4)
		c.setFlags(s&0xff == 0, false, (c.A&0xF)+(c.B&0xF)+(c.F>>4&1) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x89: {"ADC A, C", func(c *CPU) {
		s := uint16(c.A) + uint16(c.C) + (uint16(c.F&flagCarry) >> 4)
		c.setFlags(s&0xff == 0, false, (c.A&0xF)+(c.C&0xF)+(c.F>>4&1) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x8A: {"ADC A, D", func(c *CPU) {
		s := uint16(c.A) + uint16(c.D) + (uint16(c.F&flagCarry) >> 4)
		c.setFlags(s&0xff == 0, false, (c.A&0xF)+(c.D&0xF)+(c.F>>4&1) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x8B: {"ADC A, E", func(c *CPU) {
		s := uint16(c.A) + uint16(c.E) + (uint16(c.F&flagCarry) >> 4)
		c.setFlags(s&0xff == 0, false, (c.A&0xF)+(c.E&0xF)+(c.F>>4&1) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x8C: {"ADC A, H", func(c *CPU) {
		s := uint16(c.A) + uint16(c.H) + (uint16(c.F&flagCarry) >> 4)
		c.setFlags(s&0xff == 0, false, (c.A&0xF)+(c.H&0xF)+(c.F>>4&1) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x8D: {"ADC A, L", func(c *CPU) {
		s := uint16(c.A) + uint16(c.L) + (uint16(c.F&flagCarry) >> 4)
		c.setFlags(s&0xff == 0, false, (c.A&0xF)+(c.L&0xF)+(c.F>>4&1) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x8E: {"ADC A, (HL)", func(c *CPU) {
		value := c.b.ClockedRead(c.HL.Uint16())
		s := uint16(c.A) + uint16(value) + (uint16(c.F&flagCarry) >> 4)
		c.setFlags(s&0xff == 0, false, (c.A&0xF)+(value&0xF)+(c.F>>4&1) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x8F: {"ADC A, A", func(c *CPU) {
		s := uint16(c.A) + uint16(c.A) + (uint16(c.F&flagCarry) >> 4)
		c.setFlags(s&0xff == 0, false, (c.A&0xF)+(c.A&0xF)+(c.F>>4&1) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x90: {"SUB B", func(c *CPU) {
		s := uint16(c.A) - uint16(c.B)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(c.B&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x91: {"SUB C", func(c *CPU) {
		s := uint16(c.A) - uint16(c.C)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(c.C&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x92: {"SUB D", func(c *CPU) {
		s := uint16(c.A) - uint16(c.D)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(c.D&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x93: {"SUB E", func(c *CPU) {
		s := uint16(c.A) - uint16(c.E)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(c.E&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x94: {"SUB H", func(c *CPU) {
		s := uint16(c.A) - uint16(c.H)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(c.H&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x95: {"SUB L", func(c *CPU) {
		s := uint16(c.A) - uint16(c.L)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(c.L&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x96: {"SUB (HL)", func(c *CPU) {
		value := c.b.ClockedRead(c.HL.Uint16())
		s := uint16(c.A) - uint16(value)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(value&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x97: {"SUB A", func(c *CPU) {
		s := uint16(c.A) - uint16(c.A)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(c.A&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x98: {"SBC B", func(c *CPU) {
		s := uint16(c.A) - uint16(c.B) - (uint16(c.F&flagCarry) >> 4)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(c.B&0xf)-(c.F>>4&1) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x99: {"SBC C", func(c *CPU) {
		s := uint16(c.A) - uint16(c.C) - (uint16(c.F&flagCarry) >> 4)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(c.C&0xf)-(c.F>>4&1) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x9A: {"SBC D", func(c *CPU) {
		s := uint16(c.A) - uint16(c.D) - (uint16(c.F&flagCarry) >> 4)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(c.D&0xf)-(c.F>>4&1) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x9B: {"SBC E", func(c *CPU) {
		s := uint16(c.A) - uint16(c.E) - (uint16(c.F&flagCarry) >> 4)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(c.E&0xf)-(c.F>>4&1) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x9C: {"SBC H", func(c *CPU) {
		s := uint16(c.A) - uint16(c.H) - (uint16(c.F&flagCarry) >> 4)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(c.H&0xf)-(c.F>>4&1) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x9D: {"SBC L", func(c *CPU) {
		s := uint16(c.A) - uint16(c.L) - (uint16(c.F&flagCarry) >> 4)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(c.L&0xf)-(c.F>>4&1) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x9E: {"SBC (HL)", func(c *CPU) {
		value := c.b.ClockedRead(c.HL.Uint16())
		s := uint16(c.A) - uint16(value) - (uint16(c.F&flagCarry) >> 4)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(value&0xf)-(c.F>>4&1) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0x9F: {"SBC A", func(c *CPU) {
		s := uint16(c.A) - uint16(c.A) - (uint16(c.F&flagCarry) >> 4)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(c.A&0xf)-(c.F>>4&1) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0xA0: {"AND B", func(c *CPU) { c.A &= c.B; c.setFlags(c.A == 0, false, true, false) }},
	0xA1: {"AND C", func(c *CPU) { c.A &= c.C; c.setFlags(c.A == 0, false, true, false) }},
	0xA2: {"AND D", func(c *CPU) { c.A &= c.D; c.setFlags(c.A == 0, false, true, false) }},
	0xA3: {"AND E", func(c *CPU) { c.A &= c.E; c.setFlags(c.A == 0, false, true, false) }},
	0xA4: {"AND H", func(c *CPU) { c.A &= c.H; c.setFlags(c.A == 0, false, true, false) }},
	0xA5: {"AND L", func(c *CPU) { c.A &= c.L; c.setFlags(c.A == 0, false, true, false) }},
	0xA6: {"AND (HL)", func(c *CPU) {
		value := c.b.ClockedRead(c.HL.Uint16())
		c.A &= value
		c.setFlags(c.A == 0, false, true, false)
	}},
	0xA7: {"AND A", func(c *CPU) { c.A &= c.A; c.setFlags(c.A == 0, false, true, false) }},
	0xA8: {"XOR B", func(c *CPU) { c.A ^= c.B; c.setFlags(c.A == 0, false, false, false) }},
	0xA9: {"XOR C", func(c *CPU) { c.A ^= c.C; c.setFlags(c.A == 0, false, false, false) }},
	0xAA: {"XOR D", func(c *CPU) { c.A ^= c.D; c.setFlags(c.A == 0, false, false, false) }},
	0xAB: {"XOR E", func(c *CPU) { c.A ^= c.E; c.setFlags(c.A == 0, false, false, false) }},
	0xAC: {"XOR H", func(c *CPU) { c.A ^= c.H; c.setFlags(c.A == 0, false, false, false) }},
	0xAD: {"XOR L", func(c *CPU) { c.A ^= c.L; c.setFlags(c.A == 0, false, false, false) }},
	0xAE: {"XOR (HL)", func(c *CPU) {
		value := c.b.ClockedRead(c.HL.Uint16())
		c.A ^= value
		c.setFlags(c.A == 0, false, false, false)
	}},
	0xAF: {"XOR A", func(c *CPU) { c.A ^= c.A; c.setFlags(c.A == 0, false, false, false) }},
	0xB0: {"OR B", func(c *CPU) { c.A |= c.B; c.setFlags(c.A == 0, false, false, false) }},
	0xB1: {"OR C", func(c *CPU) { c.A |= c.C; c.setFlags(c.A == 0, false, false, false) }},
	0xB2: {"OR D", func(c *CPU) { c.A |= c.D; c.setFlags(c.A == 0, false, false, false) }},
	0xB3: {"OR E", func(c *CPU) { c.A |= c.E; c.setFlags(c.A == 0, false, false, false) }},
	0xB4: {"OR H", func(c *CPU) { c.A |= c.H; c.setFlags(c.A == 0, false, false, false) }},
	0xB5: {"OR L", func(c *CPU) { c.A |= c.L; c.setFlags(c.A == 0, false, false, false) }},
	0xB6: {"OR (HL)", func(c *CPU) {
		value := c.b.ClockedRead(c.HL.Uint16())
		c.A |= value
		c.setFlags(c.A == 0, false, false, false)
	}},
	0xB7: {"OR A", func(c *CPU) { c.A |= c.A; c.setFlags(c.A == 0, false, false, false) }},
	0xB8: {"CP B", func(c *CPU) { c.setFlags(c.A-c.B == 0, true, c.B&0xf > c.A&0xf, c.B > c.A) }},
	0xB9: {"CP C", func(c *CPU) { c.setFlags(c.A-c.C == 0, true, c.C&0xf > c.A&0xf, c.C > c.A) }},
	0xBA: {"CP D", func(c *CPU) { c.setFlags(c.A-c.D == 0, true, c.D&0xf > c.A&0xf, c.D > c.A) }},
	0xBB: {"CP E", func(c *CPU) { c.setFlags(c.A-c.E == 0, true, c.E&0xf > c.A&0xf, c.E > c.A) }},
	0xBC: {"CP H", func(c *CPU) { c.setFlags(c.A-c.H == 0, true, c.H&0xf > c.A&0xf, c.H > c.A) }},
	0xBD: {"CP L", func(c *CPU) { c.setFlags(c.A-c.L == 0, true, c.L&0xf > c.A&0xf, c.L > c.A) }},
	0xBE: {"CP (HL)", func(c *CPU) {
		value := c.b.ClockedRead(c.HL.Uint16())
		c.setFlags(c.A-value == 0, true, value&0xf > c.A&0xf, value > c.A)
	}},
	0xBF: {"CP A", func(c *CPU) { c.setFlags(c.A-c.A == 0, true, c.A&0xf > c.A&0xf, c.A > c.A) }},
	0xC0: {"RET NZ", func(c *CPU) { c.s.Tick(4); c.ret(!c.isFlagSet(flagZero)) }},
	0xC1: {"POP BC", func(c *CPU) {
		*c.BC[1] = c.b.ClockedRead(c.SP)
		c.SP++
		*c.BC[0] = c.b.ClockedRead(c.SP)
		c.SP++
	}},
	0xC2: {"JP NZ, a16", func(c *CPU) { c.jumpAbsolute(!c.isFlagSet(flagZero)) }},
	0xC3: {"JP a16", func(c *CPU) { c.jumpAbsolute(true) }},
	0xC4: {"CALL NZ, a16", func(c *CPU) { c.call(!c.isFlagSet(flagZero)) }},
	0xC5: {"PUSH BC", func(c *CPU) { c.s.Tick(4); c.push(c.B, c.C) }},
	0xC6: {"ADD A, d8", func(c *CPU) {
		v := c.readOperand()
		s := uint16(c.A) + uint16(v)
		c.setFlags(s&0xff == 0, false, (c.A&0xf)+(v&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0xC7: {"RST 0", func(c *CPU) { c.s.Tick(4); c.push(uint8(c.PC>>8), uint8(c.PC)); c.PC = 0 }},
	0xC8: {"RET Z", func(c *CPU) { c.s.Tick(4); c.ret(c.isFlagSet(flagZero)) }},
	0xC9: {"RET", func(c *CPU) { c.ret(true) }},
	0xCA: {"JP Z, nn", func(c *CPU) { c.jumpAbsolute(c.isFlagSet(flagZero)) }},
	0xCB: {"CB Prefix", func(c *CPU) { InstructionSetCB[c.readOperand()].fn(c) }},
	0xCC: {"CALL Z, nn", func(c *CPU) { c.call(c.isFlagSet(flagZero)) }},
	0xCD: {"CALL nn", func(c *CPU) { c.call(true) }},
	0xCE: {"ADC A, d8", func(c *CPU) {
		v := c.readOperand()
		s := uint16(c.A) + uint16(v) + uint16(c.F&flagCarry>>4)
		c.setFlags(s&0xff == 0, false, (c.A&0xf)+(v&0xf)+(c.F&flagCarry>>4) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0xCF: {"RST 1", func(c *CPU) { c.s.Tick(4); c.push(uint8(c.PC>>8), uint8(c.PC)); c.PC = 0x08 }},
	0xD0: {"RET NC", func(c *CPU) { c.s.Tick(4); c.ret(!c.isFlagSet(flagCarry)) }},
	0xD1: {"POP DE", func(c *CPU) {
		*c.DE[1] = c.b.ClockedRead(c.SP)
		c.SP++
		*c.DE[0] = c.b.ClockedRead(c.SP)
		c.SP++
	}},
	0xD2: {"JP NC, a16", func(c *CPU) { c.jumpAbsolute(!c.isFlagSet(flagCarry)) }},
	0xD3: disallowedOpcode(0xD3),
	0xD4: {"CALL NC, a16", func(c *CPU) { c.call(!c.isFlagSet(flagCarry)) }},
	0xD5: {"PUSH DE", func(c *CPU) { c.s.Tick(4); c.push(c.D, c.E) }},
	0xD6: {"SUB d8", func(c *CPU) {
		v := c.readOperand()
		s := uint16(c.A) - uint16(v)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(v&0xf) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0xD7: {"RST 2", func(c *CPU) { c.s.Tick(4); c.push(uint8(c.PC>>8), uint8(c.PC)); c.PC = 0x10 }},
	0xD8: {"RET C", func(c *CPU) { c.s.Tick(4); c.ret(c.isFlagSet(flagCarry)) }},
	0xD9: {"RETI", func(c *CPU) { c.b.EnableInterrupts(); c.ret(true) }},
	0xDA: {"JP C, nn", func(c *CPU) { c.jumpAbsolute(c.isFlagSet(flagCarry)) }},
	0xDB: disallowedOpcode(0xDB),
	0xDC: {"CALL C, nn", func(c *CPU) { c.call(c.isFlagSet(flagCarry)) }},
	0xDD: disallowedOpcode(0xDD),
	0xDE: {"SBC A, d8", func(c *CPU) {
		v := c.readOperand()
		s := uint16(c.A) - uint16(v) - uint16(c.F&flagCarry>>4)
		c.setFlags(s&0xff == 0, true, (c.A&0xf)-(v&0xf)-(c.F&flagCarry>>4) > 0xf, s > 0xff)
		c.A = uint8(s)
	}},
	0xDF: {"RST 3", func(c *CPU) { c.s.Tick(4); c.push(uint8(c.PC>>8), uint8(c.PC)); c.PC = 0x18 }},
	0xE0: {"LDH (a8), A", func(c *CPU) { c.b.ClockedWrite(0xff00+uint16(c.readOperand()), c.A) }},
	0xE1: {"POP HL", func(c *CPU) {
		*c.HL[1] = c.b.ClockedRead(c.SP)
		c.SP++
		*c.HL[0] = c.b.ClockedRead(c.SP)
		c.SP++
	}},
	0xE2: {"LD (C), A", func(c *CPU) { c.b.ClockedWrite(0xff00+uint16(c.C), c.A) }},
	0xE3: disallowedOpcode(0xE3),
	0xE4: disallowedOpcode(0xE4),
	0xE5: {"PUSH HL", func(c *CPU) { c.s.Tick(4); c.push(c.H, c.L) }},
	0xE6: {"AND d8", func(c *CPU) { c.A &= c.readOperand(); c.setFlags(c.A == 0, false, true, false) }},
	0xE7: {"RST 4", func(c *CPU) { c.s.Tick(4); c.push(uint8(c.PC>>8), uint8(c.PC)); c.PC = 0x20 }},
	0xE8: {"ADD SP, r8", func(c *CPU) { c.SP = c.addSPSigned(); c.s.Tick(4) }},
	0xE9: {"JP HL", func(c *CPU) { c.PC = c.HL.Uint16() }},
	0xEA: {"LD (a16), A", func(c *CPU) { c.b.ClockedWrite(uint16(c.readOperand())|uint16(c.readOperand())<<8, c.A) }},
	0xEB: disallowedOpcode(0xEB),
	0xEC: disallowedOpcode(0xEC),
	0xED: disallowedOpcode(0xED),
	0xEE: {"XOR d8", func(c *CPU) { c.A ^= c.readOperand(); c.setFlags(c.A == 0, false, false, false) }},
	0xEF: {"RST 5", func(c *CPU) { c.s.Tick(4); c.push(uint8(c.PC>>8), uint8(c.PC)); c.PC = 0x28 }},
	0xF0: {"LDH A, (a8)", func(c *CPU) { c.A = c.b.ClockedRead(0xff00 + uint16(c.readOperand())) }},
	0xF1: {"POP AF", func(c *CPU) {
		*c.AF[1] = c.b.ClockedRead(c.SP)
		c.SP++
		*c.AF[0] = c.b.ClockedRead(c.SP)
		c.SP++
		c.F &= 0xF0
	}},
	0xF2: {"LD A, (C)", func(c *CPU) { c.A = c.b.ClockedRead(0xff00 + uint16(c.C)) }},
	0xF3: {"DI", func(c *CPU) { c.b.DisableInterrupts() }},
	0xF4: disallowedOpcode(0xF4),
	0xF5: {"PUSH AF", func(c *CPU) { c.s.Tick(4); c.push(c.A, c.F) }},
	0xF6: {"OR d8", func(c *CPU) { c.A |= c.readOperand(); c.setFlags(c.A == 0, false, false, false) }},
	0xF7: {"RST 6", func(c *CPU) { c.s.Tick(4); c.push(uint8(c.PC>>8), uint8(c.PC)); c.PC = 0x30 }},
	0xF8: {"LD HL, SP+r8", func(c *CPU) { c.HL.SetUint16(c.addSPSigned()) }},
	0xF9: {"LD SP, HL", func(c *CPU) { c.SP = c.HL.Uint16(); c.s.Tick(4) }},
	0xFA: {"LD A, (a16)", func(c *CPU) { c.A = c.b.ClockedRead(uint16(c.readOperand()) | uint16(c.readOperand())<<8) }},
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
	0xFE: {"CP d8", func(c *CPU) { v := c.readOperand(); c.setFlags(c.A-v == 0, true, v&0xf > c.A&0xf, v > c.A) }},
	0xFF: {"RST 7", func(c *CPU) { c.s.Tick(4); c.push(uint8(c.PC>>8), uint8(c.PC)); c.PC = 0x38 }},
}

// InstructionSetCB is the set of instructions for the CB prefix.
var InstructionSetCB = [256]Instruction{
	0x00: {"RLC B", func(c *CPU) { n := c.B & 0x80; c.B = (c.B << 1) | n>>7; c.setFlags(c.B == 0, false, false, n == 0x80) }},
	0x01: {"RLC C", func(c *CPU) { n := c.C & 0x80; c.C = (c.C << 1) | n>>7; c.setFlags(c.C == 0, false, false, n == 0x80) }},
	0x02: {"RLC D", func(c *CPU) { n := c.D & 0x80; c.D = (c.D << 1) | n>>7; c.setFlags(c.D == 0, false, false, n == 0x80) }},
	0x03: {"RLC E", func(c *CPU) { n := c.E & 0x80; c.E = (c.E << 1) | n>>7; c.setFlags(c.E == 0, false, false, n == 0x80) }},
	0x04: {"RLC H", func(c *CPU) { n := c.H & 0x80; c.H = (c.H << 1) | n>>7; c.setFlags(c.H == 0, false, false, n == 0x80) }},
	0x05: {"RLC L", func(c *CPU) { n := c.L & 0x80; c.L = (c.L << 1) | n>>7; c.setFlags(c.L == 0, false, false, n == 0x80) }},
	0x06: {"RLC (HL)", func(c *CPU) {
		value := c.b.ClockedRead(c.HL.Uint16())
		n := value & 0x80
		value = (value << 1) | n>>7
		c.b.ClockedWrite(c.HL.Uint16(), value)
		c.setFlags(value == 0, false, false, n == 0x80)
	}},
	0x07: {"RLC A", func(c *CPU) { n := c.A & 0x80; c.A = (c.A << 1) | n>>7; c.setFlags(c.A == 0, false, false, n == 0x80) }},
	0x08: {"RRC B", func(c *CPU) { n := c.B & 1; c.B = (c.B >> 1) | n<<7; c.setFlags(c.B == 0, false, false, n == 1) }},
	0x09: {"RRC C", func(c *CPU) { n := c.C & 1; c.C = (c.C >> 1) | n<<7; c.setFlags(c.C == 0, false, false, n == 1) }},
	0x0A: {"RRC D", func(c *CPU) { n := c.D & 1; c.D = (c.D >> 1) | n<<7; c.setFlags(c.D == 0, false, false, n == 1) }},
	0x0B: {"RRC E", func(c *CPU) { n := c.E & 1; c.E = (c.E >> 1) | n<<7; c.setFlags(c.E == 0, false, false, n == 1) }},
	0x0C: {"RRC H", func(c *CPU) { n := c.H & 1; c.H = (c.H >> 1) | n<<7; c.setFlags(c.H == 0, false, false, n == 1) }},
	0x0D: {"RRC L", func(c *CPU) { n := c.L & 1; c.L = (c.L >> 1) | n<<7; c.setFlags(c.L == 0, false, false, n == 1) }},
	0x0E: {"RRC (HL)", func(c *CPU) {
		value := c.b.ClockedRead(c.HL.Uint16())
		n := value & 1
		value = (value >> 1) | n<<7
		c.b.ClockedWrite(c.HL.Uint16(), value)
		c.setFlags(value == 0, false, false, n == 1)
	}},
	0x0F: {"RRC A", func(c *CPU) { n := c.A & 1; c.A = (c.A >> 1) | n<<7; c.setFlags(c.A == 0, false, false, n == 1) }},
	0x10: {"RL B", func(c *CPU) {
		n := c.B&0x80 > 0
		c.B = (c.B << 1) | (c.F&flagCarry)>>4
		c.setFlags(c.B == 0, false, false, n)
	}},
	0x11: {"RL C", func(c *CPU) {
		n := c.C&0x80 > 0
		c.C = (c.C << 1) | (c.F&flagCarry)>>4
		c.setFlags(c.C == 0, false, false, n)
	}},
	0x12: {"RL D", func(c *CPU) {
		n := c.D&0x80 > 0
		c.D = (c.D << 1) | (c.F&flagCarry)>>4
		c.setFlags(c.D == 0, false, false, n)
	}},
	0x13: {"RL E", func(c *CPU) {
		n := c.E&0x80 > 0
		c.E = (c.E << 1) | (c.F&flagCarry)>>4
		c.setFlags(c.E == 0, false, false, n)
	}},
	0x14: {"RL H", func(c *CPU) {
		n := c.H&0x80 > 0
		c.H = (c.H << 1) | (c.F&flagCarry)>>4
		c.setFlags(c.H == 0, false, false, n)
	}},
	0x15: {"RL L", func(c *CPU) {
		n := c.L&0x80 > 0
		c.L = (c.L << 1) | (c.F&flagCarry)>>4
		c.setFlags(c.L == 0, false, false, n)
	}},
	0x16: {"RL (HL)", func(c *CPU) {
		value := c.b.ClockedRead(c.HL.Uint16())
		n := value&0x80 > 0
		value = (value << 1) | (c.F&flagCarry)>>4
		c.b.ClockedWrite(c.HL.Uint16(), value)
		c.setFlags(value == 0, false, false, n)
	}},
	0x17: {"RL A", func(c *CPU) {
		n := c.A&0x80 > 0
		c.A = (c.A << 1) | (c.F&flagCarry)>>4
		c.setFlags(c.A == 0, false, false, n)
	}},
	0x18: {"RR B", func(c *CPU) {
		n := c.B&0x01 > 0
		c.B = (c.B >> 1) | (c.F & flagCarry << 3)
		c.setFlags(c.B == 0, false, false, n)
	}},
	0x19: {"RR C", func(c *CPU) {
		n := c.C&0x01 > 0
		c.C = (c.C >> 1) | (c.F & flagCarry << 3)
		c.setFlags(c.C == 0, false, false, n)
	}},
	0x1A: {"RR D", func(c *CPU) {
		n := c.D&0x01 > 0
		c.D = (c.D >> 1) | (c.F & flagCarry << 3)
		c.setFlags(c.D == 0, false, false, n)
	}},
	0x1B: {"RR E", func(c *CPU) {
		n := c.E&0x01 > 0
		c.E = (c.E >> 1) | (c.F & flagCarry << 3)
		c.setFlags(c.E == 0, false, false, n)
	}},
	0x1C: {"RR H", func(c *CPU) {
		n := c.H&0x01 > 0
		c.H = (c.H >> 1) | (c.F & flagCarry << 3)
		c.setFlags(c.H == 0, false, false, n)
	}},
	0x1D: {"RR L", func(c *CPU) {
		n := c.L&0x01 > 0
		c.L = (c.L >> 1) | (c.F & flagCarry << 3)
		c.setFlags(c.L == 0, false, false, n)
	}},
	0x1E: {"RR (HL)", func(c *CPU) {
		value := c.b.ClockedRead(c.HL.Uint16())
		n := value&0x01 > 0
		value = (value >> 1) | (c.F & flagCarry << 3)
		c.b.ClockedWrite(c.HL.Uint16(), value)
		c.setFlags(value == 0, false, false, n)
	}},
	0x1F: {"RR A", func(c *CPU) {
		n := c.A&0x01 > 0
		c.A = (c.A >> 1) | (c.F & flagCarry << 3)
		c.setFlags(c.A == 0, false, false, n)
	}},

	0x20: {"SLA B", func(c *CPU) { n := c.B&0x80 > 0; c.B <<= 1; c.setFlags(c.B == 0, false, false, n) }},
	0x21: {"SLA C", func(c *CPU) { n := c.C&0x80 > 0; c.C <<= 1; c.setFlags(c.C == 0, false, false, n) }},
	0x22: {"SLA D", func(c *CPU) { n := c.D&0x80 > 0; c.D <<= 1; c.setFlags(c.D == 0, false, false, n) }},
	0x23: {"SLA E", func(c *CPU) { n := c.E&0x80 > 0; c.E <<= 1; c.setFlags(c.E == 0, false, false, n) }},
	0x24: {"SLA H", func(c *CPU) { n := c.H&0x80 > 0; c.H <<= 1; c.setFlags(c.H == 0, false, false, n) }},
	0x25: {"SLA L", func(c *CPU) { n := c.L&0x80 > 0; c.L <<= 1; c.setFlags(c.L == 0, false, false, n) }},
	0x26: {"SLA (HL)", func(c *CPU) {
		value := c.b.ClockedRead(c.HL.Uint16())
		n := value&0x80 > 0
		value <<= 1
		c.b.ClockedWrite(c.HL.Uint16(), value)
		c.setFlags(value == 0, false, false, n)
	}},
	0x27: {"SLA A", func(c *CPU) { n := c.A&0x80 > 0; c.A <<= 1; c.setFlags(c.A == 0, false, false, n) }},
	0x28: {"SRA B", func(c *CPU) { n := c.B&1 > 0; c.B = (c.B >> 1) | (c.B & 0x80); c.setFlags(c.B == 0, false, false, n) }},
	0x29: {"SRA C", func(c *CPU) { n := c.C&1 > 0; c.C = (c.C >> 1) | (c.C & 0x80); c.setFlags(c.C == 0, false, false, n) }},
	0x2A: {"SRA D", func(c *CPU) { n := c.D&1 > 0; c.D = (c.D >> 1) | (c.D & 0x80); c.setFlags(c.D == 0, false, false, n) }},
	0x2B: {"SRA E", func(c *CPU) { n := c.E&1 > 0; c.E = (c.E >> 1) | (c.E & 0x80); c.setFlags(c.E == 0, false, false, n) }},
	0x2C: {"SRA H", func(c *CPU) { n := c.H&1 > 0; c.H = (c.H >> 1) | (c.H & 0x80); c.setFlags(c.H == 0, false, false, n) }},
	0x2D: {"SRA L", func(c *CPU) { n := c.L&1 > 0; c.L = (c.L >> 1) | (c.L & 0x80); c.setFlags(c.L == 0, false, false, n) }},
	0x2E: {"SRA (HL)", func(c *CPU) {
		value := c.b.ClockedRead(c.HL.Uint16())
		n := value&1 > 0
		value = (value >> 1) | (value & 0x80)
		c.b.ClockedWrite(c.HL.Uint16(), value)
		c.setFlags(value == 0, false, false, n)
	}},
	0x2F: {"SRA A", func(c *CPU) { n := c.A&1 > 0; c.A = (c.A >> 1) | (c.A & 0x80); c.setFlags(c.A == 0, false, false, n) }},
	0x30: {"SWAP B", func(c *CPU) { c.B = c.B<<4 | c.B>>4; c.setFlags(c.B == 0, false, false, false) }},
	0x31: {"SWAP C", func(c *CPU) { c.C = c.C<<4 | c.C>>4; c.setFlags(c.C == 0, false, false, false) }},
	0x32: {"SWAP D", func(c *CPU) { c.D = c.D<<4 | c.D>>4; c.setFlags(c.D == 0, false, false, false) }},
	0x33: {"SWAP E", func(c *CPU) { c.E = c.E<<4 | c.E>>4; c.setFlags(c.E == 0, false, false, false) }},
	0x34: {"SWAP H", func(c *CPU) { c.H = c.H<<4 | c.H>>4; c.setFlags(c.H == 0, false, false, false) }},
	0x35: {"SWAP L", func(c *CPU) { c.L = c.L<<4 | c.L>>4; c.setFlags(c.L == 0, false, false, false) }},
	0x36: {"SWAP (HL)", func(c *CPU) {
		value := c.b.ClockedRead(c.HL.Uint16())
		value = value<<4 | value>>4
		c.b.ClockedWrite(c.HL.Uint16(), value)
		c.setFlags(value == 0, false, false, false)
	}},
	0x37: {"SWAP A", func(c *CPU) { c.A = c.A<<4 | c.A>>4; c.setFlags(c.A == 0, false, false, false) }},
	0x38: {"SRL B", func(c *CPU) { n := c.B&1 > 0; c.B >>= 1; c.setFlags(c.B == 0, false, false, n) }},
	0x39: {"SRL C", func(c *CPU) { n := c.C&1 > 0; c.C >>= 1; c.setFlags(c.C == 0, false, false, n) }},
	0x3A: {"SRL D", func(c *CPU) { n := c.D&1 > 0; c.D >>= 1; c.setFlags(c.D == 0, false, false, n) }},
	0x3B: {"SRL E", func(c *CPU) { n := c.E&1 > 0; c.E >>= 1; c.setFlags(c.E == 0, false, false, n) }},
	0x3C: {"SRL H", func(c *CPU) { n := c.H&1 > 0; c.H >>= 1; c.setFlags(c.H == 0, false, false, n) }},
	0x3D: {"SRL L", func(c *CPU) { n := c.L&1 > 0; c.L >>= 1; c.setFlags(c.L == 0, false, false, n) }},
	0x3E: {"SRL (HL)", func(c *CPU) {
		value := c.b.ClockedRead(c.HL.Uint16())
		n := value&1 > 0
		value >>= 1
		c.setFlags(value == 0, false, false, n)
		c.b.ClockedWrite(c.HL.Uint16(), value)
	}},
	0x3F: {"SRL A", func(c *CPU) { n := c.A&1 > 0; c.A >>= 1; c.setFlags(c.A == 0, false, false, n) }},
	0x40: {"BIT 0, B", func(c *CPU) { c.setFlags(c.B&types.Bit0 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x41: {"BIT 0, C", func(c *CPU) { c.setFlags(c.C&types.Bit0 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x42: {"BIT 0, D", func(c *CPU) { c.setFlags(c.D&types.Bit0 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x43: {"BIT 0, E", func(c *CPU) { c.setFlags(c.E&types.Bit0 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x44: {"BIT 0, H", func(c *CPU) { c.setFlags(c.H&types.Bit0 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x45: {"BIT 0, L", func(c *CPU) { c.setFlags(c.L&types.Bit0 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x46: {"BIT 0, (HL)", func(c *CPU) {
		c.setFlags(c.b.ClockedRead(c.HL.Uint16())&types.Bit0 == 0, false, true, c.isFlagSet(flagCarry))
	}},
	0x47: {"BIT 0, A", func(c *CPU) { c.setFlags(c.A&types.Bit0 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x48: {"BIT 1, B", func(c *CPU) { c.setFlags(c.B&types.Bit1 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x49: {"BIT 1, C", func(c *CPU) { c.setFlags(c.C&types.Bit1 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x4A: {"BIT 1, D", func(c *CPU) { c.setFlags(c.D&types.Bit1 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x4B: {"BIT 1, E", func(c *CPU) { c.setFlags(c.E&types.Bit1 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x4C: {"BIT 1, H", func(c *CPU) { c.setFlags(c.H&types.Bit1 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x4D: {"BIT 1, L", func(c *CPU) { c.setFlags(c.L&types.Bit1 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x4E: {"BIT 1, (HL)", func(c *CPU) {
		c.setFlags(c.b.ClockedRead(c.HL.Uint16())&types.Bit1 == 0, false, true, c.isFlagSet(flagCarry))
	}},
	0x4F: {"BIT 1, A", func(c *CPU) { c.setFlags(c.A&types.Bit1 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x50: {"BIT 2, B", func(c *CPU) { c.setFlags(c.B&types.Bit2 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x51: {"BIT 2, C", func(c *CPU) { c.setFlags(c.C&types.Bit2 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x52: {"BIT 2, D", func(c *CPU) { c.setFlags(c.D&types.Bit2 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x53: {"BIT 2, E", func(c *CPU) { c.setFlags(c.E&types.Bit2 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x54: {"BIT 2, H", func(c *CPU) { c.setFlags(c.H&types.Bit2 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x55: {"BIT 2, L", func(c *CPU) { c.setFlags(c.L&types.Bit2 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x56: {"BIT 2, (HL)", func(c *CPU) {
		c.setFlags(c.b.ClockedRead(c.HL.Uint16())&types.Bit2 == 0, false, true, c.isFlagSet(flagCarry))
	}},
	0x57: {"BIT 2, A", func(c *CPU) { c.setFlags(c.A&types.Bit2 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x58: {"BIT 3, B", func(c *CPU) { c.setFlags(c.B&types.Bit3 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x59: {"BIT 3, C", func(c *CPU) { c.setFlags(c.C&types.Bit3 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x5A: {"BIT 3, D", func(c *CPU) { c.setFlags(c.D&types.Bit3 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x5B: {"BIT 3, E", func(c *CPU) { c.setFlags(c.E&types.Bit3 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x5C: {"BIT 3, H", func(c *CPU) { c.setFlags(c.H&types.Bit3 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x5D: {"BIT 3, L", func(c *CPU) { c.setFlags(c.L&types.Bit3 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x5E: {"BIT 3, (HL)", func(c *CPU) {
		c.setFlags(c.b.ClockedRead(c.HL.Uint16())&types.Bit3 == 0, false, true, c.isFlagSet(flagCarry))
	}},
	0x5F: {"BIT 3, A", func(c *CPU) { c.setFlags(c.A&types.Bit3 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x60: {"BIT 4, B", func(c *CPU) { c.setFlags(c.B&types.Bit4 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x61: {"BIT 4, C", func(c *CPU) { c.setFlags(c.C&types.Bit4 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x62: {"BIT 4, D", func(c *CPU) { c.setFlags(c.D&types.Bit4 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x63: {"BIT 4, E", func(c *CPU) { c.setFlags(c.E&types.Bit4 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x64: {"BIT 4, H", func(c *CPU) { c.setFlags(c.H&types.Bit4 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x65: {"BIT 4, L", func(c *CPU) { c.setFlags(c.L&types.Bit4 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x66: {"BIT 4, (HL)", func(c *CPU) {
		c.setFlags(c.b.ClockedRead(c.HL.Uint16())&types.Bit4 == 0, false, true, c.isFlagSet(flagCarry))
	}},
	0x67: {"BIT 4, A", func(c *CPU) { c.setFlags(c.A&types.Bit4 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x68: {"BIT 5, B", func(c *CPU) { c.setFlags(c.B&types.Bit5 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x69: {"BIT 5, C", func(c *CPU) { c.setFlags(c.C&types.Bit5 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x6A: {"BIT 5, D", func(c *CPU) { c.setFlags(c.D&types.Bit5 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x6B: {"BIT 5, E", func(c *CPU) { c.setFlags(c.E&types.Bit5 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x6C: {"BIT 5, H", func(c *CPU) { c.setFlags(c.H&types.Bit5 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x6D: {"BIT 5, L", func(c *CPU) { c.setFlags(c.L&types.Bit5 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x6E: {"BIT 5, (HL)", func(c *CPU) {
		c.setFlags(c.b.ClockedRead(c.HL.Uint16())&types.Bit5 == 0, false, true, c.isFlagSet(flagCarry))
	}},
	0x6F: {"BIT 5, A", func(c *CPU) { c.setFlags(c.A&types.Bit5 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x70: {"BIT 6, B", func(c *CPU) { c.setFlags(c.B&types.Bit6 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x71: {"BIT 6, C", func(c *CPU) { c.setFlags(c.C&types.Bit6 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x72: {"BIT 6, D", func(c *CPU) { c.setFlags(c.D&types.Bit6 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x73: {"BIT 6, E", func(c *CPU) { c.setFlags(c.E&types.Bit6 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x74: {"BIT 6, H", func(c *CPU) { c.setFlags(c.H&types.Bit6 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x75: {"BIT 6, L", func(c *CPU) { c.setFlags(c.L&types.Bit6 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x76: {"BIT 6, (HL)", func(c *CPU) {
		c.setFlags(c.b.ClockedRead(c.HL.Uint16())&types.Bit6 == 0, false, true, c.isFlagSet(flagCarry))
	}},
	0x77: {"BIT 6, A", func(c *CPU) { c.setFlags(c.A&types.Bit6 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x78: {"BIT 7, B", func(c *CPU) { c.setFlags(c.B&types.Bit7 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x79: {"BIT 7, C", func(c *CPU) { c.setFlags(c.C&types.Bit7 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x7A: {"BIT 7, D", func(c *CPU) { c.setFlags(c.D&types.Bit7 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x7B: {"BIT 7, E", func(c *CPU) { c.setFlags(c.E&types.Bit7 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x7C: {"BIT 7, H", func(c *CPU) { c.setFlags(c.H&types.Bit7 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x7D: {"BIT 7, L", func(c *CPU) { c.setFlags(c.L&types.Bit7 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x7E: {"BIT 7, (HL)", func(c *CPU) {
		c.setFlags(c.b.ClockedRead(c.HL.Uint16())&types.Bit7 == 0, false, true, c.isFlagSet(flagCarry))
	}},
	0x7F: {"BIT 7, A", func(c *CPU) { c.setFlags(c.A&types.Bit7 == 0, false, true, c.isFlagSet(flagCarry)) }},
	0x80: {"RES 0, B", func(c *CPU) { c.B &^= types.Bit0 }},
	0x81: {"RES 0, C", func(c *CPU) { c.C &^= types.Bit0 }},
	0x82: {"RES 0, D", func(c *CPU) { c.D &^= types.Bit0 }},
	0x83: {"RES 0, E", func(c *CPU) { c.E &^= types.Bit0 }},
	0x84: {"RES 0, H", func(c *CPU) { c.H &^= types.Bit0 }},
	0x85: {"RES 0, L", func(c *CPU) { c.L &^= types.Bit0 }},
	0x86: {"RES 0, (HL)", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.b.ClockedRead(c.HL.Uint16())&^types.Bit0) }},
	0x87: {"RES 0, A", func(c *CPU) { c.A &^= types.Bit0 }},
	0x88: {"RES 1, B", func(c *CPU) { c.B &^= types.Bit1 }},
	0x89: {"RES 1, C", func(c *CPU) { c.C &^= types.Bit1 }},
	0x8A: {"RES 1, D", func(c *CPU) { c.D &^= types.Bit1 }},
	0x8B: {"RES 1, E", func(c *CPU) { c.E &^= types.Bit1 }},
	0x8C: {"RES 1, H", func(c *CPU) { c.H &^= types.Bit1 }},
	0x8D: {"RES 1, L", func(c *CPU) { c.L &^= types.Bit1 }},
	0x8E: {"RES 1, (HL)", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.b.ClockedRead(c.HL.Uint16())&^types.Bit1) }},
	0x8F: {"RES 1, A", func(c *CPU) { c.A &^= types.Bit1 }},
	0x90: {"RES 2, B", func(c *CPU) { c.B &^= types.Bit2 }},
	0x91: {"RES 2, C", func(c *CPU) { c.C &^= types.Bit2 }},
	0x92: {"RES 2, D", func(c *CPU) { c.D &^= types.Bit2 }},
	0x93: {"RES 2, E", func(c *CPU) { c.E &^= types.Bit2 }},
	0x94: {"RES 2, H", func(c *CPU) { c.H &^= types.Bit2 }},
	0x95: {"RES 2, L", func(c *CPU) { c.L &^= types.Bit2 }},
	0x96: {"RES 2, (HL)", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.b.ClockedRead(c.HL.Uint16())&^types.Bit2) }},
	0x97: {"RES 2, A", func(c *CPU) { c.A &^= types.Bit2 }},
	0x98: {"RES 3, B", func(c *CPU) { c.B &^= types.Bit3 }},
	0x99: {"RES 3, C", func(c *CPU) { c.C &^= types.Bit3 }},
	0x9A: {"RES 3, D", func(c *CPU) { c.D &^= types.Bit3 }},
	0x9B: {"RES 3, E", func(c *CPU) { c.E &^= types.Bit3 }},
	0x9C: {"RES 3, H", func(c *CPU) { c.H &^= types.Bit3 }},
	0x9D: {"RES 3, L", func(c *CPU) { c.L &^= types.Bit3 }},
	0x9E: {"RES 3, (HL)", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.b.ClockedRead(c.HL.Uint16())&^types.Bit3) }},
	0x9F: {"RES 3, A", func(c *CPU) { c.A &^= types.Bit3 }},
	0xA0: {"RES 4, B", func(c *CPU) { c.B &^= types.Bit4 }},
	0xA1: {"RES 4, C", func(c *CPU) { c.C &^= types.Bit4 }},
	0xA2: {"RES 4, D", func(c *CPU) { c.D &^= types.Bit4 }},
	0xA3: {"RES 4, E", func(c *CPU) { c.E &^= types.Bit4 }},
	0xA4: {"RES 4, H", func(c *CPU) { c.H &^= types.Bit4 }},
	0xA5: {"RES 4, L", func(c *CPU) { c.L &^= types.Bit4 }},
	0xA6: {"RES 4, (HL)", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.b.ClockedRead(c.HL.Uint16())&^types.Bit4) }},
	0xA7: {"RES 4, A", func(c *CPU) { c.A &^= types.Bit4 }},
	0xA8: {"RES 5, B", func(c *CPU) { c.B &^= types.Bit5 }},
	0xA9: {"RES 5, C", func(c *CPU) { c.C &^= types.Bit5 }},
	0xAA: {"RES 5, D", func(c *CPU) { c.D &^= types.Bit5 }},
	0xAB: {"RES 5, E", func(c *CPU) { c.E &^= types.Bit5 }},
	0xAC: {"RES 5, H", func(c *CPU) { c.H &^= types.Bit5 }},
	0xAD: {"RES 5, L", func(c *CPU) { c.L &^= types.Bit5 }},
	0xAE: {"RES 5, (HL)", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.b.ClockedRead(c.HL.Uint16())&^types.Bit5) }},
	0xAF: {"RES 5, A", func(c *CPU) { c.A &^= types.Bit5 }},
	0xB0: {"RES 6, B", func(c *CPU) { c.B &^= types.Bit6 }},
	0xB1: {"RES 6, C", func(c *CPU) { c.C &^= types.Bit6 }},
	0xB2: {"RES 6, D", func(c *CPU) { c.D &^= types.Bit6 }},
	0xB3: {"RES 6, E", func(c *CPU) { c.E &^= types.Bit6 }},
	0xB4: {"RES 6, H", func(c *CPU) { c.H &^= types.Bit6 }},
	0xB5: {"RES 6, L", func(c *CPU) { c.L &^= types.Bit6 }},
	0xB6: {"RES 6, (HL)", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.b.ClockedRead(c.HL.Uint16())&^types.Bit6) }},
	0xB7: {"RES 6, A", func(c *CPU) { c.A &^= types.Bit6 }},
	0xB8: {"RES 7, B", func(c *CPU) { c.B &^= types.Bit7 }},
	0xB9: {"RES 7, C", func(c *CPU) { c.C &^= types.Bit7 }},
	0xBA: {"RES 7, D", func(c *CPU) { c.D &^= types.Bit7 }},
	0xBB: {"RES 7, E", func(c *CPU) { c.E &^= types.Bit7 }},
	0xBC: {"RES 7, H", func(c *CPU) { c.H &^= types.Bit7 }},
	0xBD: {"RES 7, L", func(c *CPU) { c.L &^= types.Bit7 }},
	0xBE: {"RES 7, (HL)", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.b.ClockedRead(c.HL.Uint16())&^types.Bit7) }},
	0xBF: {"RES 7, A", func(c *CPU) { c.A &^= types.Bit7 }},
	0xC0: {"SET 0, B", func(c *CPU) { c.B |= types.Bit0 }},
	0xC1: {"SET 0, C", func(c *CPU) { c.C |= types.Bit0 }},
	0xC2: {"SET 0, D", func(c *CPU) { c.D |= types.Bit0 }},
	0xC3: {"SET 0, E", func(c *CPU) { c.E |= types.Bit0 }},
	0xC4: {"SET 0, H", func(c *CPU) { c.H |= types.Bit0 }},
	0xC5: {"SET 0, L", func(c *CPU) { c.L |= types.Bit0 }},
	0xC6: {"SET 0, (HL)", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.b.ClockedRead(c.HL.Uint16())|types.Bit0) }},
	0xC7: {"SET 0, A", func(c *CPU) { c.A |= types.Bit0 }},
	0xC8: {"SET 1, B", func(c *CPU) { c.B |= types.Bit1 }},
	0xC9: {"SET 1, C", func(c *CPU) { c.C |= types.Bit1 }},
	0xCA: {"SET 1, D", func(c *CPU) { c.D |= types.Bit1 }},
	0xCB: {"SET 1, E", func(c *CPU) { c.E |= types.Bit1 }},
	0xCC: {"SET 1, H", func(c *CPU) { c.H |= types.Bit1 }},
	0xCD: {"SET 1, L", func(c *CPU) { c.L |= types.Bit1 }},
	0xCE: {"SET 1, (HL)", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.b.ClockedRead(c.HL.Uint16())|types.Bit1) }},
	0xCF: {"SET 1, A", func(c *CPU) { c.A |= types.Bit1 }},
	0xD0: {"SET 2, B", func(c *CPU) { c.B |= types.Bit2 }},
	0xD1: {"SET 2, C", func(c *CPU) { c.C |= types.Bit2 }},
	0xD2: {"SET 2, D", func(c *CPU) { c.D |= types.Bit2 }},
	0xD3: {"SET 2, E", func(c *CPU) { c.E |= types.Bit2 }},
	0xD4: {"SET 2, H", func(c *CPU) { c.H |= types.Bit2 }},
	0xD5: {"SET 2, L", func(c *CPU) { c.L |= types.Bit2 }},
	0xD6: {"SET 2, (HL)", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.b.ClockedRead(c.HL.Uint16())|types.Bit2) }},
	0xD7: {"SET 2, A", func(c *CPU) { c.A |= types.Bit2 }},
	0xD8: {"SET 3, B", func(c *CPU) { c.B |= types.Bit3 }},
	0xD9: {"SET 3, C", func(c *CPU) { c.C |= types.Bit3 }},
	0xDA: {"SET 3, D", func(c *CPU) { c.D |= types.Bit3 }},
	0xDB: {"SET 3, E", func(c *CPU) { c.E |= types.Bit3 }},
	0xDC: {"SET 3, H", func(c *CPU) { c.H |= types.Bit3 }},
	0xDD: {"SET 3, L", func(c *CPU) { c.L |= types.Bit3 }},
	0xDE: {"SET 3, (HL)", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.b.ClockedRead(c.HL.Uint16())|types.Bit3) }},
	0xDF: {"SET 3, A", func(c *CPU) { c.A |= types.Bit3 }},
	0xE0: {"SET 4, B", func(c *CPU) { c.B |= types.Bit4 }},
	0xE1: {"SET 4, C", func(c *CPU) { c.C |= types.Bit4 }},
	0xE2: {"SET 4, D", func(c *CPU) { c.D |= types.Bit4 }},
	0xE3: {"SET 4, E", func(c *CPU) { c.E |= types.Bit4 }},
	0xE4: {"SET 4, H", func(c *CPU) { c.H |= types.Bit4 }},
	0xE5: {"SET 4, L", func(c *CPU) { c.L |= types.Bit4 }},
	0xE6: {"SET 4, (HL)", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.b.ClockedRead(c.HL.Uint16())|types.Bit4) }},
	0xE7: {"SET 4, A", func(c *CPU) { c.A |= types.Bit4 }},
	0xE8: {"SET 5, B", func(c *CPU) { c.B |= types.Bit5 }},
	0xE9: {"SET 5, C", func(c *CPU) { c.C |= types.Bit5 }},
	0xEA: {"SET 5, D", func(c *CPU) { c.D |= types.Bit5 }},
	0xEB: {"SET 5, E", func(c *CPU) { c.E |= types.Bit5 }},
	0xEC: {"SET 5, H", func(c *CPU) { c.H |= types.Bit5 }},
	0xED: {"SET 5, L", func(c *CPU) { c.L |= types.Bit5 }},
	0xEE: {"SET 5, (HL)", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.b.ClockedRead(c.HL.Uint16())|types.Bit5) }},
	0xEF: {"SET 5, A", func(c *CPU) { c.A |= types.Bit5 }},
	0xF0: {"SET 6, B", func(c *CPU) { c.B |= types.Bit6 }},
	0xF1: {"SET 6, C", func(c *CPU) { c.C |= types.Bit6 }},
	0xF2: {"SET 6, D", func(c *CPU) { c.D |= types.Bit6 }},
	0xF3: {"SET 6, E", func(c *CPU) { c.E |= types.Bit6 }},
	0xF4: {"SET 6, H", func(c *CPU) { c.H |= types.Bit6 }},
	0xF5: {"SET 6, L", func(c *CPU) { c.L |= types.Bit6 }},
	0xF6: {"SET 6, (HL)", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.b.ClockedRead(c.HL.Uint16())|types.Bit6) }},
	0xF7: {"SET 6, A", func(c *CPU) { c.A |= types.Bit6 }},
	0xF8: {"SET 7, B", func(c *CPU) { c.B |= types.Bit7 }},
	0xF9: {"SET 7, C", func(c *CPU) { c.C |= types.Bit7 }},
	0xFA: {"SET 7, D", func(c *CPU) { c.D |= types.Bit7 }},
	0xFB: {"SET 7, E", func(c *CPU) { c.E |= types.Bit7 }},
	0xFC: {"SET 7, H", func(c *CPU) { c.H |= types.Bit7 }},
	0xFD: {"SET 7, L", func(c *CPU) { c.L |= types.Bit7 }},
	0xFE: {"SET 7, (HL)", func(c *CPU) { c.b.ClockedWrite(c.HL.Uint16(), c.b.ClockedRead(c.HL.Uint16())|types.Bit7) }},
	0xFF: {"SET 7, A", func(c *CPU) { c.A |= types.Bit7 }},
}

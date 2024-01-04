package cpu

import (
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/utils"
)

var incDecBit = []uint16{0x01, 0xffff}
var incDecMask = []uint8{0x0f, 0x00}

func (c *CPU) decode(instr byte) {
	switch instr { // handle instructions that can't be decoded (or I'm too lazy)
	case 0x00: // NOP
	case 0x08: // LD (a16), SP
		address := uint16(c.readOperand()) | uint16(c.readOperand())<<8
		c.b.ClockedWrite(address, uint8(c.SP&0xFF))
		c.b.ClockedWrite(address+1, uint8(c.SP>>8))
	case 0x10: // STOP
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
	case 0x18: // JR r8
		c.jumpRelative(true)
	case 0x31: // LD SP, d16
		c.SP = uint16(c.readOperand()) | uint16(c.readOperand())<<8
	case 0x32: // LD (HL-), A
		c.b.ClockedWrite(c.HL.Uint16(), c.A)
		c.HL.SetUint16(c.HL.Uint16() - 1)
	case 0x33: // INC SP
		if c.SP >= 0xFE00 && c.SP <= 0xFEFF && c.b.Get(types.STAT)&0b11 == ppu.ModeOAM {
			c.ppu.WriteCorruptionOAM()
		}
		c.SP++
		c.s.Tick(4)
	case 0x39: // ADD HL, SP
		c.HL.SetUint16(c.addUint16(c.HL.Uint16(), c.SP))
		c.s.Tick(4)
	case 0x3A: // LD A, (HL-)
		c.handleOAMCorruption(c.HL.Uint16())
		c.A = c.b.ClockedRead(c.HL.Uint16())
		c.HL.SetUint16(c.HL.Uint16() - 1)
	case 0x3B: // DEC SP
		c.SP--
		c.s.Tick(4)
	case 0x40: // LD B, B
		if c.Debug {
			c.DebugBreakpoint = true
		}
	case 0x76: // HALT
		if c.b.InterruptsEnabled() {
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
	case 0xC3: // JP a16
		c.jumpAbsolute(true)
	case 0xC9: // RET
		c.ret(true)
	case 0xCB: // CB Prefix
		c.decodeCB(c.readOperand())
	case 0xCD: // CALL nn
		c.call(true)
	case 0xD9: // RETI
		c.b.EnableInterrupts()
		c.ret(true)
	case 0xE0: // LDH (a8), A
		c.b.ClockedWrite(0xFF00+uint16(c.readOperand()), c.A)
	case 0xE2: // LD (C), A
		c.b.ClockedWrite(0xFF00+uint16(c.C), c.A)
	case 0xE8: // ADD SP, r8
		c.SP = c.addSPSigned()
		c.s.Tick(4)
	case 0xE9: // JP HL
		c.PC = c.HL.Uint16()
	case 0xEA: // LD (a16), A
		c.b.ClockedWrite(uint16(c.readOperand())|uint16(c.readOperand())<<8, c.A)
	case 0xF0: // LDH A, (a8)
		c.A = c.b.ClockedRead(0xFF00 + uint16(c.readOperand()))
	case 0xF2: // LD A, (C)
		c.A = c.b.ClockedRead(0xFF00 + uint16(c.C))
	case 0xF3: // DI
		c.b.DisableInterrupts()
	case 0xF8: // LD HL, SP+r8
		c.HL.SetUint16(c.addSPSigned())
	case 0xF9: // LD SP, HL
		c.SP = c.HL.Uint16()
		c.s.Tick(4)
	case 0xFA: // LD A, (a16)
		c.A = c.b.ClockedRead(uint16(c.readOperand()) | uint16(c.readOperand())<<8)
	case 0xFB: // EI
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
	default:
		switch instr >> 6 & 0x3 {
		case 0: // 0x00 - 0x3F
			switch instr & 0x7 {
			case 0:
				c.jumpRelative(c.getFlagCondition(instr))
			case 1:
				p := c.getRegisterPair(instr)
				if instr>>3&1 == 1 { // ADD HL, (nn)
					c.HL.SetUint16(c.addUint16(c.HL.Uint16(), p.Uint16()))
					c.s.Tick(4)
				} else { // LD (nn), d16
					p.SetUint16(uint16(c.readOperand()) | uint16(c.readOperand())<<8)
				}
			case 2: // LD
				if instr>>3&1 == 1 { // LD A, (nn)
					c.A = c.b.ClockedRead(c.getRegisterPair(instr).Uint16())
					if instr>>4&3 == 2 { // LD A, (HL+)
						c.handleOAMCorruption(c.HL.Uint16())
						c.HL.SetUint16(c.HL.Uint16() + 1)
					}
				} else { // LD (nn), A
					c.b.ClockedWrite(c.getRegisterPair(instr).Uint16(), c.A)
					if instr>>4&3 == 2 { // LD (HL+), A
						c.HL.SetUint16(c.HL.Uint16() + 1)
					}
				}
			case 3: // INC/DEC nn
				p := c.getRegisterPair(instr)
				c.handleOAMCorruption(p.Uint16())
				p.SetUint16(p.Uint16() + incDecBit[instr>>3&1])
				c.s.Tick(4)
			case 4, 5: // INC/DEC n
				src, srcMem := c.getSourceRegister(instr >> 3)
				val := *src
				val += uint8(incDecBit[instr&1])
				c.setFlags(val == 0, instr&1 == 1, *src&0xf == incDecMask[instr&1], c.isFlagSet(flagCarry))
				if srcMem {
					c.b.ClockedWrite(c.HL.Uint16(), val)
				} else {
					*src = val
				}
			case 6: // LD n, d8
				src, srcMem := c.getSourceRegister(instr >> 3)
				*src = c.readOperand()
				if srcMem {
					c.b.Write(c.HL.Uint16(), *src)
				}
			case 7: // various maths
				switch instr >> 3 & 0x7 {
				case 0:
					res := c.A<<1 | c.A&types.Bit7>>7
					c.F = c.A & types.Bit7 >> 3
					c.A = res
				case 1: // RRCA
					res := c.A>>1 | c.A&types.Bit0<<7
					c.F = c.A & types.Bit0 << 4
					c.A = res
				case 2: // RLA
					res := c.A<<1 | c.F&flagCarry>>4
					c.F = c.A & types.Bit7 >> 3 & flagCarry
					c.A = res
				case 3: // RRA
					res := c.A>>1 | c.F&flagCarry<<3
					c.F = c.A & types.Bit0 << 4 & flagCarry
					c.A = res
				case 4: // DAA
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
				case 5: // CPL
					c.A = 0xFF ^ c.A
					c.setFlags(c.isFlagSet(flagZero), true, true, c.isFlagSet(flagCarry))
				case 6: // SCF
					c.setFlags(c.isFlagSet(flagZero), false, false, true)
				case 7: // CCF
					c.setFlags(c.isFlagSet(flagZero), false, false, !c.isFlagSet(flagCarry))
				}
			}
		case 1:
			dst, _ := c.getSourceRegister(instr)
			src, srcMem := c.getSourceRegister(instr >> 3)

			*src = *dst
			// if the src register is memory, we need to write the new value back
			// to the bus
			if srcMem {
				c.b.Write(c.HL.Uint16(), *src)
			}
		case 2: // ALU n
			dest, _ := c.getSourceRegister(instr)
			c.decodeALU(instr, *dest)
		case 3: //
			switch instr & 0x7 {
			case 0: // RET
				c.s.Tick(4)
				c.ret(c.getFlagCondition(instr))
			case 1: // POP nn
				p := c.getRegisterPair(instr)
				*p.Low = c.b.ClockedRead(c.SP)
				c.SP++
				c.handleOAMCorruption(c.SP)
				*p.High = c.b.ClockedRead(c.SP)
				c.SP++

				if instr&0xf0 == 0xf0 {
					c.F &= 0xf0 // clear unused bits
				}
			case 2: // JP
				c.jumpAbsolute(c.getFlagCondition(instr))
			case 4: // CALL
				c.call(c.getFlagCondition(instr))
			case 5: // PUSH nn
				p := c.getRegisterPair(instr)
				c.s.Tick(4)
				c.push(*p.High, *p.Low)
			case 6: // ALU d8
				c.decodeALU(instr, c.readOperand())
			case 7: // RST
				c.s.Tick(4)
				c.push(uint8(c.PC>>8), uint8(c.PC&0xFF))
				c.PC = uint16(instr >> 3 & 0x7 * 8)
			}
		}
	}
}

// decodeCB decodes a CB-prefixed instruction.
//
//	00 000 000
//	^^ ^^^ ^^^
//	op src dst
func (c *CPU) decodeCB(instr byte) {
	src, srcMem := c.getSourceRegister(instr)

	val := *src
decode:
	switch instr >> 6 & 0x3 {
	case 0:
		// 0b00 000 000
		//   ^^ ^^^ ^^^
		//   ma op  des
		//
		switch instr >> 3 & 0x7 {
		case 0: // RLC
			val = val<<1 | (val&types.Bit7)>>7
		case 1: // RRC
			val = val>>1 | (val&types.Bit0)<<7
		case 2: // RL
			val = val<<1 | c.F&flagCarry>>4
		case 3: // RR
			val = val>>1 | c.F&flagCarry<<3
		case 4: // SLA
			val <<= 1
		case 5: // SRA
			val = val&types.Bit7 | val>>1
		case 6: // SWAP
			val = val<<4 | val>>4
			c.setFlags(val == 0, false, false, false)
			break decode // SWAP resets the carry flag unlike the other instructions
		case 7: // SRL
			val >>= 1
		}
		c.setFlags(val == 0, false, false, instr>>3&1 == 1 && *src&types.Bit0 == types.Bit0 || instr>>3&1 == 0 && *src&types.Bit7 == types.Bit7)
	case 1: // BIT
		bitIndex := uint8(1 << (instr >> 3 & 0x7))
		c.setFlags(val&bitIndex != bitIndex, false, true, c.isFlagSet(flagCarry))
		return // BIT doesn't change the value of the source register
	case 2: // RES
		val = utils.Reset(val, 1<<(instr>>3&0x7))
	case 3: // SET
		val = utils.Set(val, 1<<(instr>>3&0x7))
	}

	if srcMem {
		// write new value back to the bus
		c.b.ClockedWrite(c.HL.Uint16(), val)
	} else {
		*src = val
	}
}

// decodeALU decodes an ALU instruction, performing various maths on the
// A register.
func (c *CPU) decodeALU(instr, ask byte) {
	switch instr >> 3 & 0x7 {
	case 0: // ADD
		sum := uint16(c.A) + uint16(ask)
		c.setFlags(sum&0xff == 0, false, (c.A&0xf)+(ask&0xf) > 0xf, sum > 0xff)
		c.A = Register(sum)
	case 1: // ADC
		sum := uint16(c.A) + uint16(ask) + uint16(c.F>>4&1)
		c.setFlags(sum&0xff == 0, false, (c.A&0xf)+(ask&0xf)+(c.F>>4&1) > 0xf, sum > 0xff)
		c.A = Register(sum)
	case 2: // SUB
		sum := uint16(c.A) - uint16(ask)
		c.setFlags(sum&0xff == 0, true, (c.A&0xf)-(ask&0xf) > 0xf, sum > 0xff)
		c.A = Register(sum)
	case 3: // SBC
		sum := uint16(c.A) - uint16(ask) - uint16(c.F>>4&1)
		c.setFlags(sum&0xff == 0, true, (c.A&0xf)-(ask&0xf)-(c.F>>4&1) > 0xf, sum > 0xff)
		c.A = Register(sum)
	case 4: // AND
		c.A &= ask
		c.setFlags(c.A == 0, false, true, false)
	case 5: // XOR
		c.A ^= ask
		c.setFlags(c.A == 0, false, false, false)
	case 6: // OR
		c.A |= ask
		c.setFlags(c.A == 0, false, false, false)
	case 7:
		c.setFlags(c.A-ask == 0, true, ask&0x0f > c.A&0x0f, ask > c.A)
	}
}

// getSourceRegister returns a pointer to the register specified by the
// given register index.
func (c *CPU) getSourceRegister(reg byte) (*uint8, bool) {
	switch reg & 0x7 {
	case 0:
		return &c.B, false
	case 1:
		return &c.C, false
	case 2:
		return &c.D, false
	case 3:
		return &c.E, false
	case 4:
		return &c.H, false
	case 5:
		return &c.L, false
	case 6:
		val := c.b.ClockedRead(c.HL.Uint16())
		return &val, true
	case 7:
		return &c.A, false
	}
	return nil, false
}

// getFlagCondition returns the condition of the flag specified by the
// given instruction.
func (c *CPU) getFlagCondition(instr byte) bool {
	var f bool
	switch instr >> 4 & 1 {
	case 0:
		f = c.isFlagSet(flagZero)
	case 1:
		f = c.isFlagSet(flagCarry)
	}

	if instr>>3&1 == 0 {
		f = !f
	}

	return f
}

// getRegisterPair returns the register pair specified by the given
// instruction.
func (c *CPU) getRegisterPair(instr byte) *RegisterPair {
	switch instr >> 4 & 0x3 {
	case 0:
		return c.BC
	case 1:
		return c.DE
	case 2:
		return c.HL
	case 3:
		return c.AF
	}
	return nil
}

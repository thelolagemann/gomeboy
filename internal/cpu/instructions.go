package cpu

import (
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/utils"
)

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
	case 0xF1: // POP AF
		c.popNN(&c.A, &c.F)
		c.F &= 0xF0 // clear unused bits
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
					c.addHLRR(p)
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
			case 3:
				switch instr >> 3 & 1 {
				case 0: // INC nn
					c.incrementNN(c.getRegisterPair(instr))
				case 1: // DEC nn
					c.decrementNN(c.getRegisterPair(instr))
				}
			case 4: // INC
				src, srcMem := c.getSourceRegister(instr >> 3)
				*src = c.increment(*src)
				if srcMem {
					c.b.ClockedWrite(c.HL.Uint16(), *src)
				}
			case 5: // DEC
				src, srcMem := c.getSourceRegister(instr >> 3)
				*src = c.decrement(*src)
				if srcMem {
					c.b.ClockedWrite(c.HL.Uint16(), *src)
				}
			case 6: // LD d8
				src, srcMem := c.getSourceRegister(instr >> 3)
				*src = c.readOperand()
				if srcMem {
					c.b.Write(c.HL.Uint16(), *src)
				}
			case 7: // various maths
				switch instr >> 3 & 0x7 {
				case 0:
					c.rotateLeftCarryAccumulator()
				case 1: // RRCA
					c.rotateRightAccumulator()
				case 2: // RLA
					c.rotateLeftAccumulatorThroughCarry()
				case 3: // RRA
					c.rotateRightAccumulatorThroughCarry()
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
				c.popNN(p.High, p.Low)
			case 2: // JP
				c.jumpAbsolute(c.getFlagCondition(instr))
			case 4: // CALL
				c.call(c.getFlagCondition(instr))
			case 5: // PUSH nn
				p := c.getRegisterPair(instr)
				c.pushNN(*p.High, *p.Low)
			case 6: // ALU d8
				c.decodeALU(instr, c.readOperand())
			case 7: // RST
				c.rst(uint16(instr >> 3 & 0x7 * 8))
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

	switch instr >> 6 & 0x3 {
	case 0:
		// 0b00 000 000
		//   ^^ ^^^ ^^^
		//   ma op  des
		//
		switch instr >> 3 & 0x7 {
		case 0: // RLC
			*src = c.rotateLeftCarry(*src)
		case 1: // RRC
			*src = c.rotateRightCarry(*src)
		case 2: // RL
			*src = c.rotateLeftThroughCarry(*src)
		case 3: // RR
			*src = c.rotateRightThroughCarry(*src)
		case 4: // SLA
			*src = c.shiftLeftArithmetic(*src)
		case 5: // SRA
			*src = c.shiftRightArithmetic(*src)
		case 6: // SWAP
			*src = c.swap(*src)
		case 7: // SRL
			*src = c.shiftRightLogical(*src)
		}
	case 1: // BIT
		c.testBit(*src, 1<<(instr>>3&0x7))
		srcMem = false
	case 2: // RES
		*src = utils.Reset(*src, 1<<(instr>>3&0x7))
	case 3: // SET
		*src = utils.Set(*src, 1<<(instr>>3&0x7))
	}

	if srcMem {
		// write new value back to the bus
		c.b.ClockedWrite(c.HL.Uint16(), *src)
	}
}

func (c *CPU) decodeALU(instr, ask byte) {
	switch instr >> 3 & 0x7 {
	case 0:
		c.add(ask, false)
	case 1:
		c.add(ask, true)
	case 2:
		c.sub(ask, false)
	case 3:
		c.sub(ask, true)
	case 4:
		c.and(ask)
	case 5:
		c.xor(ask)
	case 6:
		c.or(ask)
	case 7:
		c.compare(ask)
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

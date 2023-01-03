package cpu

import "testing"

func TestInstruction_Calls(t *testing.T) {
	// 0xC4 - CALL NZ,nn
	testInstruction(t, "CALL NZ,nn", 0xC4, callFlagConditionalTest(FlagZero, false))
	// 0xCC - CALL Z,nn
	testInstruction(t, "CALL Z,nn", 0xCC, callFlagConditionalTest(FlagZero, true))
	// 0xCD - CALL nn
	testInstruction(t, "CALL nn", 0xCD, func(t *testing.T, instruction Instruction) {
		cpu.PC = 0x1234
		cpu.SP = 0xFFFE

		instruction.Execute(cpu, []byte{0x42, 0x42}) // 0x1234 (PC) written to address 0xFFFE (SP), PC set to 0x4242

		// ensure that PC was jumped
		if cpu.PC != 0x4242 {
			t.Errorf("expected PC to be 0x4242, got 0x%04X", cpu.PC)
		}

		// ensure that SP was decremented
		if cpu.SP != 0xFFFC {
			t.Errorf("expected SP to be 0xFFFC, got 0x%04X", cpu.SP)
		}

		// ensure that address 0xFFFE contains 0x12
		if cpu.mmu.Read(0xFFFD) != 0x12 {
			t.Errorf("expected 0x12 at address 0xFFFD, got 0x%02X", cpu.mmu.Read(0xFFFD))
		}

		// ensure that address 0xFFFD contains 0x34
		if cpu.mmu.Read(0xFFFC) != 0x34 {
			t.Errorf("expected 0x34 at address 0xFFFE, got 0x%02X", cpu.mmu.Read(0xFFFE))
		}
	})
	// 0xD4 - CALL NC,nn
	testInstruction(t, "CALL NC,nn", 0xD4, callFlagConditionalTest(FlagCarry, false))
	// 0xDC - CALL C,nn
	testInstruction(t, "CALL C,nn", 0xDC, callFlagConditionalTest(FlagCarry, true))
}

func TestInstruction_Jumps(t *testing.T) {
	// 0x18 - JR n - Jump relative to PC
	testInstruction(t, "JR n", 0x18, func(t *testing.T, inst Instruction) {
		cpu.PC = 0x0000

		inst.Execute(cpu, []byte{0x03})

		// ensure that PC was jumped
		if cpu.PC != 3 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}

		// as the instruction takes a signed byte, ensure that negative values jump backwards
		cpu.PC = 0x0500

		inst.Execute(cpu, []byte{0xFF})

		if cpu.PC != 0x04FF {
			t.Errorf("expected PC to be 0x04FF, got 0x%04X", cpu.PC)
		}
	})
	// 0x20 - JR NZ, n - Jump relative to PC if zero flag is not set
	testInstruction(t, "JR NZ, n", 0x20, func(t *testing.T, inst Instruction) {
		cpu.PC = 0x0000
		cpu.clearFlag(FlagZero)

		inst.Execute(cpu, []byte{0x03})

		// ensure that PC was jumped
		if cpu.PC != 0x0003 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}

		// ensure that PC was not jumped if zero flag is set
		cpu.setFlag(FlagZero)
		inst.Execute(cpu, []byte{0x03})

		if cpu.PC != 0x0003 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}
	})
	// 0x28 - JR Z, n - Jump relative to PC if zero flag is set
	testInstruction(t, "JR Z, n", 0x28, func(t *testing.T, inst Instruction) {
		cpu.PC = 0x0000
		cpu.setFlag(FlagZero)

		inst.Execute(cpu, []byte{0x03})

		// ensure that PC was jumped
		if cpu.PC != 0x0003 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}

		// ensure that PC was not jumped if zero flag is not set
		cpu.clearFlag(FlagZero)
		inst.Execute(cpu, []byte{0x03})

		if cpu.PC != 0x0003 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}
	})
	// 0x30 - JR NC, n - Jump relative to PC if carry flag is not set
	testInstruction(t, "JR NC, n", 0x30, func(t *testing.T, inst Instruction) {
		cpu.PC = 0x0000
		cpu.clearFlag(FlagCarry)

		inst.Execute(cpu, []byte{0x03})

		// ensure that PC was jumped
		if cpu.PC != 0x0003 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}

		// ensure that PC was not jumped if carry flag is set
		cpu.setFlag(FlagCarry)
		inst.Execute(cpu, []byte{0x03})

		if cpu.PC != 0x0003 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}
	})
	// 0x38 - JR C, n - Jump relative to PC if carry flag is set
	testInstruction(t, "JR C, n", 0x38, func(t *testing.T, inst Instruction) {
		cpu.PC = 0x0000
		cpu.setFlag(FlagCarry)

		inst.Execute(cpu, []byte{0x03})

		// ensure that PC was jumped
		if cpu.PC != 0x0003 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}

		// ensure that PC was not jumped if carry flag is not set
		cpu.clearFlag(FlagCarry)
		inst.Execute(cpu, []byte{0x03})

		if cpu.PC != 0x0003 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}
	})
	// 0xC2 - JP NZ, nn - Jump to address if zero flag is not set
	testInstruction(t, "JP NZ, nn", 0xC2, func(t *testing.T, inst Instruction) {
		cpu.PC = 0x0000
		cpu.clearFlag(FlagZero)

		inst.Execute(cpu, []uint8{0x00, 0x03})

		// ensure that PC was jumped
		if cpu.PC != 0x0300 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}

		// ensure that PC was not jumped if zero flag is set
		cpu.setFlag(FlagZero)
		inst.Execute(cpu, []byte{0x00, 0x03})

		if cpu.PC != 0x0300 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}
	})
	// 0xC3 - JP nn - Jump to address
	testInstruction(t, "JP nn", 0xC3, func(t *testing.T, inst Instruction) {
		cpu.PC = 0x0000

		inst.Execute(cpu, []byte{0x00, 0x03})

		// ensure that PC was jumped
		if cpu.PC != 0x0300 {
			t.Errorf("expected PC to be 0x0300, got 0x%04X", cpu.PC)
		}
	})
	// 0xCA - JP Z, nn - Jump to address if zero flag is set
	testInstruction(t, "JP Z, nn", 0xCA, func(t *testing.T, inst Instruction) {
		cpu.PC = 0x0000
		cpu.setFlag(FlagZero)

		inst.Execute(cpu, []byte{0x00, 0x03})

		// ensure that PC was jumped
		if cpu.PC != 0x0300 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}

		// ensure that PC was not jumped if zero flag is not set
		cpu.clearFlag(FlagZero)
		inst.Execute(cpu, []byte{0x00, 0x03})

		if cpu.PC != 0x0300 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}
	})
	// 0xD2 - JP NC, nn - Jump to address if carry flag is not set
	testInstruction(t, "JP NC, nn", 0xD2, func(t *testing.T, inst Instruction) {
		cpu.PC = 0x0000
		cpu.clearFlag(FlagCarry)

		inst.Execute(cpu, []byte{0x00, 0x03})

		// ensure that PC was jumped
		if cpu.PC != 0x0300 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}

		// ensure that PC was not jumped if carry flag is set
		cpu.setFlag(FlagCarry)
		inst.Execute(cpu, []byte{0x00, 0x03})

		if cpu.PC != 0x0300 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}
	})
	// 0xDA - JP C, nn - Jump to address if carry flag is set
	testInstruction(t, "JP C, nn", 0xDA, func(t *testing.T, inst Instruction) {
		cpu.PC = 0x0000
		cpu.setFlag(FlagCarry)

		inst.Execute(cpu, []byte{0x00, 0x03})

		// ensure that PC was jumped
		if cpu.PC != 0x0300 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}

		// ensure that PC was not jumped if carry flag is not set
		cpu.clearFlag(FlagCarry)
		inst.Execute(cpu, []byte{0x00, 0x03})

		if cpu.PC != 0x0300 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}
	})
	// 0xE9 - JP (HL) - Jump to address stored in HL
	testInstruction(t, "JP (HL)", 0xE9, func(t *testing.T, inst Instruction) {
		cpu.PC = 0x0000
		cpu.HL.SetUint16(0x0003)

		inst.Execute(cpu, nil)

		// ensure that PC was jumped
		if cpu.PC != 0x0003 {
			t.Errorf("expected PC to be 0x0003, got 0x%04X", cpu.PC)
		}
	})
}

func TestInstruction_Resets(t *testing.T) {
	// 0xC7 - RST 00H - Push present address onto stack and jump to address
	testInstruction(t, "RST 00H", 0xC7, resetTestInstruction(0x00))
	// 0xCF - RST 08H - Push present address onto stack and jump to address
	testInstruction(t, "RST 08H", 0xCF, resetTestInstruction(0x08))
	// 0xD7 - RST 10H - Push present address onto stack and jump to address
	testInstruction(t, "RST 10H", 0xD7, resetTestInstruction(0x10))
	// 0xDF - RST 18H - Push present address onto stack and jump to address
	testInstruction(t, "RST 18H", 0xDF, resetTestInstruction(0x18))
	// 0xE7 - RST 20H - Push present address onto stack and jump to address
	testInstruction(t, "RST 20H", 0xE7, resetTestInstruction(0x20))
	// 0xEF - RST 28H - Push present address onto stack and jump to address
	testInstruction(t, "RST 28H", 0xEF, resetTestInstruction(0x28))
	// 0xF7 - RST 30H - Push present address onto stack and jump to address
	testInstruction(t, "RST 30H", 0xF7, resetTestInstruction(0x30))
	// 0xFF - RST 38H - Push present address onto stack and jump to address
	testInstruction(t, "RST 38H", 0xFF, resetTestInstruction(0x38))
}

func TestInstruction_Returns(t *testing.T) {
	// 0xC0 - RET NZ - Return if zero flag is not set
	testInstruction(t, "RET NZ", 0xC0, returnFlagConditional(FlagZero, false))
	// 0xC8 - RET Z - Return if zero flag is set
	testInstruction(t, "RET Z", 0xC8, returnFlagConditional(FlagZero, true))
	// 0xC9 - RET - Return
	testInstruction(t, "RET", 0xC9, func(t *testing.T, inst Instruction) {
		cpu.PC = 0x1234
		cpu.SP = 0xFFFE

		cpu.mmu.Write(0xFFFE, 0x42)
		cpu.mmu.Write(0xFFFF, 0x42)

		inst.Execute(cpu, nil)

		if cpu.PC != 0x4242 {
			t.Errorf("expected PC to be 0x4242, got 0x%04X", cpu.PC)
		}

		if cpu.SP != 0x0000 {
			t.Errorf("expected SP to be 0x0000, got 0x%04X", cpu.SP)
		}
	})
	// 0xD0 - RET NC - Return if carry flag is not set
	testInstruction(t, "RET NC", 0xD0, returnFlagConditional(FlagCarry, false))
	// 0xD8 - RET C - Return if carry flag is set
	testInstruction(t, "RET C", 0xD8, returnFlagConditional(FlagCarry, true))
	// 0xD9 - RETI - Return and enable interrupts
	testInstruction(t, "RETI", 0xD9, func(t *testing.T, inst Instruction) {
		cpu.PC = 0x1234
		cpu.SP = 0xFFFE

		cpu.mmu.Write(0xFFFE, 0x42)
		cpu.mmu.Write(0xFFFF, 0x42)

		inst.Execute(cpu, nil)

		if cpu.PC != 0x4242 {
			t.Errorf("expected PC to be 0x4242, got 0x%04X", cpu.PC)
		}

		if cpu.SP != 0x0000 {
			t.Errorf("expected SP to be 0x0000, got 0x%04X", cpu.SP)
		}

		if !cpu.mmu.Bus.Interrupts().IME {
			t.Error("expected interrupts to be enabled")
		}
	})
}

func callFlagConditionalTest(flag Flag, condition bool) func(*testing.T, Instruction) {
	return func(t *testing.T, instruction Instruction) {
		if condition {
			cpu.setFlag(flag)
		} else {
			cpu.clearFlag(flag)
		}

		cpu.PC = 0x1234
		cpu.SP = 0xFFFE

		instruction.Execute(cpu, []byte{0x42, 0x42}) // 0x1234 (PC) written to address 0xFFFE (SP), PC set to 0x4242

		// ensure that PC was jumped
		if cpu.PC != 0x4242 {
			t.Errorf("expected PC to be 0x4242, got 0x%04X", cpu.PC)
		}

		// ensure that SP was decremented
		if cpu.SP != 0xFFFC {
			t.Errorf("expected SP to be 0xFFFC, got 0x%04X", cpu.SP)
		}

		// ensure that address 0xFFFE contains 0x12
		if cpu.mmu.Read(0xFFFD) != 0x12 {
			t.Errorf("expected 0x12 at address 0xFFFD, got 0x%02X", cpu.mmu.Read(0xFFFD))
		}

		// ensure that address 0xFFFD contains 0x34
		if cpu.mmu.Read(0xFFFC) != 0x34 {
			t.Errorf("expected 0x34 at address 0xFFFE, got 0x%02X", cpu.mmu.Read(0xFFFE))
		}

		// test condition not met
		t.Run("Condition Not Met", func(t *testing.T) {
			if condition {
				cpu.clearFlag(flag)
			} else {
				cpu.setFlag(flag)
			}

			cpu.PC = 0x1234
			cpu.SP = 0xFFFE

			instruction.Execute(cpu, []byte{0x42, 0x42}) // 0x1234 (PC) written to address 0xFFFE (SP), PC set to 0x4242

			// ensure that PC was not jumped
			if cpu.PC != 0x1234 {
				t.Errorf("expected PC to be 0x1234, got 0x%04X", cpu.PC)
			}

			// ensure that SP was not decremented
			if cpu.SP != 0xFFFE {
				t.Errorf("expected SP to be 0xFFFE, got 0x%04X", cpu.SP)
			}
		})
	}
}

func returnFlagConditional(flag Flag, condition bool) func(*testing.T, Instruction) {
	return func(t *testing.T, instruction Instruction) {
		if condition {
			cpu.setFlag(flag)
		} else {
			cpu.clearFlag(flag)
		}

		cpu.PC = 0x1234

		cpu.SP = 0xFFFC
		cpu.mmu.Write(0xFFFC, 0x42)
		cpu.mmu.Write(0xFFFD, 0x42)

		instruction.Execute(cpu, nil)

		// ensure that PC was jumped
		if cpu.PC != 0x4242 {
			t.Errorf("expected PC to be 0x4242, got 0x%04X", cpu.PC)
		}

		// ensure that SP was incremented
		if cpu.SP != 0xFFFE {
			t.Errorf("expected SP to be 0xFFFE, got 0x%04X", cpu.SP)
		}

		// test condition not met
		t.Run("Condition Not Met", func(t *testing.T) {
			if condition {
				cpu.clearFlag(flag)
			} else {
				cpu.setFlag(flag)
			}

			cpu.PC = 0x1234

			cpu.SP = 0xFFFC
			cpu.mmu.Write(0xFFFC, 0x42)
			cpu.mmu.Write(0xFFFD, 0x42)

			instruction.Execute(cpu, nil)

			// ensure that PC was not jumped
			if cpu.PC != 0x1234 {
				t.Errorf("expected PC to be 0x1234, got 0x%04X", cpu.PC)
			}

			// ensure that SP was not incremented
			if cpu.SP != 0xFFFC {
				t.Errorf("expected SP to be 0xFFFC, got 0x%04X", cpu.SP)
			}
		})
	}
}

func resetTestInstruction(n uint8) func(*testing.T, Instruction) {
	return func(t *testing.T, instruction Instruction) {
		cpu.PC = 0x1234
		cpu.SP = 0xFFFE

		instruction.Execute(cpu, nil)

		// ensure that PC was jumped
		if cpu.PC != 0x0000+uint16(n) {
			t.Errorf("expected PC to be 0x%04X, got 0x%04X", 0x0000+uint16(n), cpu.PC)
		}

		// ensure that SP was decremented
		if cpu.SP != 0xFFFC {
			t.Errorf("expected SP to be 0xFFFC, got 0x%04X", cpu.SP)
		}

		// ensure that PC was pushed to stack
		if cpu.mmu.Read16(0xFFFC) != 0x1234 {
			t.Errorf("expected 0x1234 at address 0xFFFC, got 0x%04X", cpu.mmu.Read16(0xFFFC))
		}
	}
}

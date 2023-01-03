package cpu

import "testing"

func TestInstruction_Load(t *testing.T) {
	// 0x02 - LD (BC), A
	testInstruction(t, "LD (BC), A", 0x02, func(t *testing.T, instruction Instruction) {
		cpu.A = 0x42
		cpu.BC.SetUint16(0x1234)
		instruction.Execute(cpu, nil)
		if cpu.mmu.Read(0x1234) != 0x42 {
			t.Errorf("Expected 0x42 to be written to 0x1234, got %d", cpu.mmu.Read(0x1234))
		}
	})
	// 0x06 - LD B,n
	testInstruction(t, "LD B,n", 0x06, testLoadRegisterNInstruction("B"))
	// 0x0E - LD C,n
	testInstruction(t, "LD C,n", 0x0E, testLoadRegisterNInstruction("C"))
	// 0x16 - LD D,n
	testInstruction(t, "LD D,n", 0x16, testLoadRegisterNInstruction("D"))
	// 0x1E - LD E,n
	testInstruction(t, "LD E,n", 0x1E, testLoadRegisterNInstruction("E"))
	// 0x26 - LD H,n
	testInstruction(t, "LD H,n", 0x26, testLoadRegisterNInstruction("H"))
	// 0x2E - LD L,n
	testInstruction(t, "LD L,n", 0x2E, testLoadRegisterNInstruction("L"))
	// 0x3E - LD A,n
	testInstruction(t, "LD A,n", 0x3E, testLoadRegisterNInstruction("A"))
	// 0x40 - 0x47 - LD B,B (Except 0x46)
	for i := uint8(0); i < uint8(len(registerNames)); i++ {
		if i == 6 {
			continue
		}
		testInstruction(t, "LD B,"+registerNames[i], 0x40+i, testLoadRegisterToRegisterInstruction(registerNames[i], "B"))
	}
	// 0x48 - 0x4F - LD C,B (Except 0x4E)
	for i := uint8(0); i < uint8(len(registerNames)); i++ {
		if i == 6 {
			continue
		}
		testInstruction(t, "LD C,"+registerNames[i], 0x48+i, testLoadRegisterToRegisterInstruction(registerNames[i], "C"))
	}
	// 0x50 - 0x57 - LD D,B (Except 0x56)
	for i := uint8(0); i < uint8(len(registerNames)); i++ {
		if i == 6 {
			continue
		}
		testInstruction(t, "LD D,"+registerNames[i], 0x50+i, testLoadRegisterToRegisterInstruction(registerNames[i], "D"))
	}
	// 0x58 - 0x5F - LD E,B (Except 0x5E)
	for i := uint8(0); i < uint8(len(registerNames)); i++ {
		if i == 6 {
			continue
		}
		testInstruction(t, "LD E,"+registerNames[i], 0x58+i, testLoadRegisterToRegisterInstruction(registerNames[i], "E"))
	}
	// 0x60 - 0x67 - LD H,B (Except 0x66)
	for i := uint8(0); i < uint8(len(registerNames)); i++ {
		if i == 6 {
			continue
		}
		testInstruction(t, "LD H,"+registerNames[i], 0x60+i, testLoadRegisterToRegisterInstruction(registerNames[i], "H"))
	}
	// 0x68 - 0x6F - LD L,B (Except 0x6E)
	for i := uint8(0); i < uint8(len(registerNames)); i++ {
		if i == 6 {
			continue
		}
		testInstruction(t, "LD L,"+registerNames[i], 0x68+i, testLoadRegisterToRegisterInstruction(registerNames[i], "L"))
	}
	// 0x78 - 0x7F - LD A,B (Except 0x7E)
	for i := uint8(0); i < uint8(len(registerNames)); i++ {
		if i == 6 {
			continue
		}
		testInstruction(t, "LD A,"+registerNames[i], 0x78+i, testLoadRegisterToRegisterInstruction(registerNames[i], "A"))
	}

	// LD n, (HL)
	for i, regName := range registerNames {
		if i == 6 {
			continue
		}
		testInstruction(t, "LD "+regName+", (HL)", 0x46+uint8(i*8), func(t *testing.T, instruction Instruction) {
			cpu.HL.SetUint16(0x1234)
			cpu.mmu.Write(0x1234, 0x42)
			instruction.Execute(cpu, nil)
			if *cpu.registerMap(regName) != 0x42 {
				t.Errorf("Expected %s to be 0x42, got %d", regName, *cpu.registerMap(regName))
			}
		})
	}

	// LD (HL), n
	for i, regName := range registerNames {
		if i == 6 {
			continue
		}
		testInstruction(t, "LD (HL), "+regName, 0x70+uint8(i), func(t *testing.T, instruction Instruction) {
			cpu.HL.SetUint16(0x1234)
			*cpu.registerMap(regName) = 0x42
			instruction.Execute(cpu, nil)
			if cpu.mmu.Read(0x1234) != 0x42 {
				t.Errorf("Expected 0x42 to be written to 0x1234, got %d", cpu.mmu.Read(0x1234))
			}
		})
	}

	// 0xE0 - LDH (n), A
	testInstruction(t, "LDH (n), A", 0xE0, func(t *testing.T, instruction Instruction) {
		cpu.A = 0x42
		instruction.Execute(cpu, []uint8{0x34})
		if cpu.mmu.Read(0xFF00+0x34) != 0x42 {
			t.Errorf("Expected 0x42 to be written to 0xFF00+0x34, got %d", cpu.mmu.Read(0xFF00+0x34))
		}
	})
	// 0xE2 - LDH (C), A
	testInstruction(t, "LDH (C), A", 0xE2, func(t *testing.T, instruction Instruction) {
		cpu.A = 0x42
		cpu.C = 0x34
		instruction.Execute(cpu, nil)
		if cpu.mmu.Read(0xFF00+0x34) != 0x42 {
			t.Errorf("Expected 0x42 to be written to 0xFF00+0x34, got %d", cpu.mmu.Read(0xFF00+0x34))
		}
	})
	// 0xEA - LD (nn), A
	testInstruction(t, "LD (nn), A", 0xEA, func(t *testing.T, instruction Instruction) {
		cpu.A = 0x42
		instruction.Execute(cpu, []uint8{0x34, 0x12})
		if cpu.mmu.Read(0x1234) != 0x42 {
			t.Errorf("Expected 0x42 to be written to 0x1234, got %d", cpu.mmu.Read(0x1234))
		}
	})
	// 0xF0 - LDH A, (n)
	testInstruction(t, "LDH A, (n)", 0xF0, func(t *testing.T, instruction Instruction) {
		cpu.mmu.Write(0xFF00+0x34, 0x42)
		instruction.Execute(cpu, []uint8{0x34})
		if cpu.A != 0x42 {
			t.Errorf("Expected A to be 0x42, got %d", cpu.A)
		}
	})
	// 0xF2 - LD A, (C)
	testInstruction(t, "LD A, (C)", 0xF2, func(t *testing.T, instruction Instruction) {
		cpu.C = 0x42
		cpu.mmu.Write(0xFF00+0x42, 0x42)
		instruction.Execute(cpu, nil)
		if cpu.A != 0x42 {
			t.Errorf("Expected A to be 0x42, got %d", cpu.A)
		}
	})
	// 0xFA - LD A, (nn)
	testInstruction(t, "LD A, (nn)", 0xFA, func(t *testing.T, instruction Instruction) {
		cpu.mmu.Write(0x1234, 0x42)
		instruction.Execute(cpu, []uint8{0x34, 0x12})
		if cpu.A != 0x42 {
			t.Errorf("Expected A to be 0x42, got %d", cpu.A)
		}
	})
}

func TestInstruction_16BitLoad(t *testing.T) {
	// 0x01 - LD BC, nn - Load 16-bit immediate value into BC
	testInstruction(t, "LD BC,nn", 0x01, func(t *testing.T, instruction Instruction) {
		// Test all possible 16-bit values
		for i := uint16(0); i < 0xFFFF; i++ {
			instruction.Execute(cpu, []uint8{uint8(i), uint8(i >> 8)})
			if cpu.BC.Uint16() != i {
				t.Errorf("Expected BC to be %d, got %d", i, cpu.BC.Uint16())
			}
		}
	})
	// 0x08 - LD (nn), SP - Load SP into address pointed to by 16-bit immediate value
	testInstruction(t, "LD (nn), SP", 0x08, func(t *testing.T, instruction Instruction) {
		cpu.SP = 0x1234
		instruction.Execute(cpu, []uint8{0x42, 0x42}) // SP 0x1234 loaded into 0x4242

		if cpu.mmu.Read(0x4242) != 0x34 {
			t.Errorf("expected 0x34, got 0x%02X", cpu.mmu.Read(0x4242))
		}
		if cpu.mmu.Read(0x4243) != 0x12 {
			t.Errorf("expected 0x12, got 0x%02X", cpu.mmu.Read(0x4243))
		}
	})
	// 0x11 - LD DE, nn - Load 16-bit immediate value into DE
	testInstruction(t, "LD DE,nn", 0x11, func(t *testing.T, instruction Instruction) {
		// Test all possible 16-bit values
		for i := uint16(0); i < 0xFFFF; i++ {
			instruction.Execute(cpu, []uint8{uint8(i), uint8(i >> 8)})
			if cpu.DE.Uint16() != i {
				t.Errorf("Expected DE to be %d, got %d", i, cpu.DE.Uint16())
			}
		}
	})
	// 0x21 - LD HL, nn - Load 16-bit immediate value into HL
	testInstruction(t, "LD HL,nn", 0x21, func(t *testing.T, instruction Instruction) {
		// Test all possible 16-bit values
		for i := uint16(0); i < 0xFFFF; i++ {
			instruction.Execute(cpu, []uint8{uint8(i), uint8(i >> 8)})
			if cpu.HL.Uint16() != i {
				t.Errorf("Expected HL to be %d, got %d", i, cpu.HL.Uint16())
			}
		}
	})
	// 0x31 - LD SP, nn - Load 16-bit immediate value into SP
	testInstruction(t, "LD SP,nn", 0x31, func(t *testing.T, instruction Instruction) {
		// Test all possible 16-bit values
		for i := uint16(0); i < 0xFFFF; i++ {
			instruction.Execute(cpu, []uint8{uint8(i), uint8(i >> 8)})
			if cpu.SP != i {
				t.Errorf("Expected SP to be %d, got %d", i, cpu.SP)
			}
		}
	})
	// 0xC1 - POP BC - Pop 16-bit value from stack into BC
	t.Run("POP BC", func(t *testing.T) {
		instr := InstructionSet[0xC1]
		cpu.SP = 0xFFFE
		cpu.mmu.Write(0xFFFE, 0x34)
		cpu.mmu.Write(0xFFFF, 0x12)

		instr.Execute(cpu, nil)

		if cpu.BC.Uint16() != 0x1234 {
			t.Errorf("expected BC to be 0x1234, got 0x%04X", cpu.BC.Uint16())
		}
	})
	// 0xC5 - PUSH BC - Push 16-bit value from BC onto stack
	t.Run("PUSH BC", func(t *testing.T) {
		instr := InstructionSet[0xC5]
		cpu.BC.SetUint16(0x1234)

		instr.Execute(cpu, nil)

		if cpu.mmu.Read(0xFFFE) != 0x34 {
			t.Errorf("expected memory at 0xFFFE to be 0x34, got 0x%02X", cpu.mmu.Read(0xFFFE))
		}
		if cpu.mmu.Read(0xFFFF) != 0x12 {
			t.Errorf("expected memory at 0xFFFF to be 0x12, got 0x%02X", cpu.mmu.Read(0xFFFF))
		}
	})
	// 0xD1 - POP DE - Pop 16-bit value from stack into DE
	t.Run("POP DE", func(t *testing.T) {
		instr := InstructionSet[0xD1]
		cpu.SP = 0xFFFE
		cpu.mmu.Write(0xFFFE, 0x34)
		cpu.mmu.Write(0xFFFF, 0x12)

		instr.Execute(cpu, nil)

		if cpu.DE.Uint16() != 0x1234 {
			t.Errorf("expected DE to be 0x1234, got 0x%04X", cpu.DE.Uint16())
		}
	})
	// 0xD5 - PUSH DE - Push 16-bit value from DE onto stack
	t.Run("PUSH DE", func(t *testing.T) {
		instr := InstructionSet[0xD5]
		cpu.DE.SetUint16(0x1234)

		instr.Execute(cpu, nil)

		if cpu.mmu.Read(0xFFFE) != 0x34 {
			t.Errorf("expected memory at 0xFFFE to be 0x34, got 0x%02X", cpu.mmu.Read(0xFFFE))
		}
		if cpu.mmu.Read(0xFFFF) != 0x12 {
			t.Errorf("expected memory at 0xFFFF to be 0x12, got 0x%02X", cpu.mmu.Read(0xFFFF))
		}
	})
	// 0xE1 - POP HL - Pop 16-bit value from stack into HL
	t.Run("POP HL", func(t *testing.T) {
		instr := InstructionSet[0xE1]
		cpu.SP = 0xFFFE
		cpu.mmu.Write(0xFFFE, 0x34)
		cpu.mmu.Write(0xFFFF, 0x12)

		instr.Execute(cpu, nil)

		if cpu.HL.Uint16() != 0x1234 {
			t.Errorf("expected HL to be 0x1234, got 0x%04X", cpu.HL.Uint16())
		}
	})
	// 0xE5 - PUSH HL - Push 16-bit value from HL onto stack
	t.Run("PUSH HL", func(t *testing.T) {
		instr := InstructionSet[0xE5]
		cpu.HL.SetUint16(0x1234)

		instr.Execute(cpu, nil)

		if cpu.mmu.Read(0xFFFE) != 0x34 {
			t.Errorf("expected memory at 0xFFFE to be 0x34, got 0x%02X", cpu.mmu.Read(0xFFFE))
		}
		if cpu.mmu.Read(0xFFFF) != 0x12 {
			t.Errorf("expected memory at 0xFFFF to be 0x12, got 0x%02X", cpu.mmu.Read(0xFFFF))
		}
	})
	// 0xF1 - POP AF - Pop 16-bit value from stack into AF
	t.Run("POP AF", func(t *testing.T) {
		instr := InstructionSet[0xF1]
		cpu.SP = 0xFFFE
		cpu.mmu.Write(0xFFFE, 0x34)
		cpu.mmu.Write(0xFFFF, 0x12)

		instr.Execute(cpu, nil)

		if cpu.AF.Uint16() != 0x1234 {
			t.Errorf("expected AF to be 0x1234, got 0x%04X", cpu.AF.Uint16())
		}

		// ensure all flags are set correctly
		// TODO
	})
	// 0xF5 - PUSH AF - Push 16-bit value from AF onto stack
	t.Run("PUSH AF", func(t *testing.T) {
		instr := InstructionSet[0xF5]
		cpu.AF.SetUint16(0x1234)

		instr.Execute(cpu, nil)

		if cpu.mmu.Read(0xFFFE) != 0x34 {
			t.Errorf("expected memory at 0xFFFE to be 0x34, got 0x%02X", cpu.mmu.Read(0xFFFE))
		}
		if cpu.mmu.Read(0xFFFF) != 0x12 {
			t.Errorf("expected memory at 0xFFFF to be 0x12, got 0x%02X", cpu.mmu.Read(0xFFFF))
		}
	})
	// 0xF8 - LD HL,SP+r8 - Load 16-bit value from SP+r8 into HL
	t.Run("LD HL,SP+r8", func(t *testing.T) {
		instr := InstructionSet[0xF8]
		cpu.SP = 0x1234
		cpu.mmu.Write(0x1234, 0x34)
		cpu.mmu.Write(0x1235, 0x12)

		instr.Execute(cpu, []uint8{0x01})

		if cpu.HL.Uint16() != 0x1235 {
			t.Errorf("expected HL to be 0x1235, got 0x%04X", cpu.HL.Uint16())
		}
	})
	// 0xF9 - LD SP,HL - Load 16-bit value from HL into SP
	t.Run("LD SP,HL", func(t *testing.T) {
		instr := InstructionSet[0xF9]
		cpu.HL.SetUint16(0x1234)

		instr.Execute(cpu, nil)

		if cpu.SP != 0x1234 {
			t.Errorf("expected SP to be 0x1234, got 0x%04X", cpu.SP)
		}
	})
}

func testLoadRegisterToRegisterInstruction(a, b string) func(*testing.T, Instruction) {
	return func(t *testing.T, instruction Instruction) {
		for i := uint8(0); i < 0xFF; i++ {
			*cpu.registerMap(a) = i
			instruction.Execute(cpu, nil)
			if *cpu.registerMap(b) != i {
				t.Errorf("Expected %s to be %d, got %d", b, i, *cpu.registerMap(b))
			}
		}
	}
}

func testLoadRegisterNInstruction(regName string) func(*testing.T, Instruction) {
	return func(t *testing.T, instruction Instruction) {
		for i := uint8(0); i < 0xFF; i++ {
			instruction.Execute(cpu, []uint8{i})
			if *cpu.registerMap(regName) != i {
				t.Errorf("Expected %s to be %d, got %d", regName, i, *cpu.registerMap(regName))
			}
		}
	}
}

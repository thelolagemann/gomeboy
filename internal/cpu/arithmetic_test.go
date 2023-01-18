package cpu

import (
	"math/rand"
	"testing"
)

func TestInstruction_Arithmetic(t *testing.T) {
	// 0x04 - INC B
	testInstruction(t, "INC B", 0x04, incrementRegisterTest("B"))
	// 0x05 - DEC B
	testInstruction(t, "DEC B", 0x05, decrementRegisterTest("B"))
	// 0x0C - INC C
	testInstruction(t, "INC C", 0x0C, incrementRegisterTest("C"))
	// 0x0D - DEC C
	testInstruction(t, "DEC C", 0x0D, decrementRegisterTest("C"))
	// 0x14 - INC D
	testInstruction(t, "INC D", 0x14, incrementRegisterTest("D"))
	// 0x15 - DEC D
	testInstruction(t, "DEC D", 0x15, decrementRegisterTest("D"))
	// 0x1C - INC E
	testInstruction(t, "INC E", 0x1C, incrementRegisterTest("E"))
	// 0x1D - DEC E
	testInstruction(t, "DEC E", 0x1D, decrementRegisterTest("E"))
	// 0x24 - INC H
	testInstruction(t, "INC H", 0x24, incrementRegisterTest("H"))
	// 0x25 - DEC H
	testInstruction(t, "DEC H", 0x25, decrementRegisterTest("H"))
	// 0x2C - INC L
	testInstruction(t, "INC L", 0x2C, incrementRegisterTest("L"))
	// 0x2D - DEC L
	testInstruction(t, "DEC L", 0x2D, decrementRegisterTest("L"))
	// 0x3C - INC A
	testInstruction(t, "INC A", 0x3C, incrementRegisterTest("A"))
	// 0x3D - DEC A
	testInstruction(t, "DEC A", 0x3D, decrementRegisterTest("A"))
	// 0x34 - INC (HL)
	testInstruction(t, "INC (HL)", 0x34, func(t *testing.T, instr Instructor) {
		cpu.HL.SetUint16(0x1234)
		cpu.mmu.Write(cpu.HL.Uint16(), 0x42)

		instr.Execute(cpu)

		if cpu.mmu.Read(cpu.HL.Uint16()) != 0x43 {
			t.Errorf("Expected memory at 0x1234 to be 0x43, got 0x%02x", cpu.mmu.Read(cpu.HL.Uint16()))
		}

		if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) && cpu.isFlagsNotSet(FlagCarry) {
			t.Errorf("Expected flags to be 0, got 0x%02x", cpu.F)
		}
	})
	// 0x35 - DEC (HL)
	testInstruction(t, "DEC (HL)", 0x35, func(t *testing.T, instr Instructor) {
		cpu.HL.SetUint16(0x1234)
		cpu.mmu.Write(cpu.HL.Uint16(), 0x42)

		instr.Execute(cpu)

		if cpu.mmu.Read(cpu.HL.Uint16()) != 0x41 {
			t.Errorf("Expected memory at 0x1234 to be 0x41, got 0x%02x", cpu.mmu.Read(cpu.HL.Uint16()))
		}

		if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) && cpu.isFlagsNotSet(FlagCarry) {
			t.Errorf("Expected flags to be 0, got 0x%02x", cpu.F)
		}
	})
	// 0x80 - 0x87 (Except 0x86) - ADD A, r
	for i, regName := range registerNames {
		if regName == "(HL)" {
			continue
		}
		testInstruction(t, "ADD A, "+regName, 0x80+i, addRegisterTest(regName))
	}
	// 0x88 - 0x8F (Except 0x8E) - ADC A, r
	for i, regName := range registerNames {
		if regName == "(HL)" {
			continue
		}
		testInstruction(t, "ADC A, "+regName, 0x88+i, addCarryRegisterTest(regName))
	}
	// 0x90 - 0x97 (Except 0x96) - SUB A, r
	for i, regName := range registerNames {
		if regName == "(HL)" {
			continue
		}
		testInstruction(t, "SUB A, "+regName, 0x90+i, subtractRegisterTest(regName))
	}
	// 0x98 - 0x9F (Except 0x9E) - SBC A, r
	for i, regName := range registerNames {
		if regName == "(HL)" {
			continue
		}
		testInstruction(t, "SBC A, "+regName, 0x98+i, subtractCarryRegisterTest(regName))
	}
}

func testInstruction_16BitArithmetic(t *testing.T) {
	// 0x03 - INC BC
	testInstruction(t, "INC BC", 0x03, incrementRegisterPairTest("BC"))
	// 0x09 - ADD HL, BC
	testInstruction(t, "ADD HL, BC", 0x09, addRegisterPairHLTest("BC"))
	// 0x0B - DEC BC
	testInstruction(t, "DEC BC", 0x0B, decrementRegisterPairTest("BC"))
	// 0x13 - INC DE
	testInstruction(t, "INC DE", 0x13, incrementRegisterPairTest("DE"))
	// 0x19 - ADD HL, DE
	testInstruction(t, "ADD HL, DE", 0x19, addRegisterPairHLTest("DE"))
	// 0x1B - DEC DE
	testInstruction(t, "DEC DE", 0x1B, decrementRegisterPairTest("DE"))
	// 0x23 - INC HL
	testInstruction(t, "INC HL", 0x23, incrementRegisterPairTest("HL"))
	// 0x29 - ADD HL, HL
	testInstruction(t, "ADD HL, HL", 0x29, addRegisterPairHLTest("HL"))
	// 0x2B - DEC HL
	testInstruction(t, "DEC HL", 0x2B, decrementRegisterPairTest("HL"))
	// 0x33 - INC SP
	testInstruction(t, "INC SP", 0x33, func(t *testing.T, instr Instructor) {
		cpu.SP = 0x1234

		instr.Execute(cpu)

		if cpu.SP != 0x1235 {
			t.Errorf("Expected SP to be 0x1235, got 0x%04x", cpu.SP)
		}

		if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) && cpu.isFlagsNotSet(FlagCarry) {
			t.Errorf("Expected flags to be 0, got 0x%02x", cpu.F)
		}
	})
	// 0x39 - ADD HL, SP
	testInstruction(t, "ADD HL, SP", 0x39, func(t *testing.T, instr Instructor) {
		cpu.SP = 0x1234
		cpu.HL.SetUint16(0x5678)

		instr.Execute(cpu)

		if cpu.HL.Uint16() != 0x68AC {
			t.Errorf("Expected HL to be 0x68AC, got 0x%04x", cpu.HL.Uint16())
		}

		if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) && cpu.isFlagsNotSet(FlagCarry) {
			t.Errorf("Expected flags to be 0, got 0x%02x", cpu.F)
		}
	})
	// 0x3B - DEC SP
	testInstruction(t, "DEC SP", 0x3B, func(t *testing.T, instr Instructor) {
		cpu.SP = 0x1234

		instr.Execute(cpu)

		if cpu.SP != 0x1233 {
			t.Errorf("Expected SP to be 0x1233, got 0x%04x", cpu.SP)
		}

		if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) && cpu.isFlagsNotSet(FlagCarry) {
			t.Errorf("Expected flags to be 0, got 0x%02x", cpu.F)
		}
	})
	// 0xE8 - ADD SP, n
	testInstruction(t, "ADD SP, n", 0xE8, func(t *testing.T, instr Instructor) {
		cpu.SP = 0x1234

		instr.Execute(cpu)

		if cpu.SP != 0x1276 {
			t.Errorf("Expected SP to be 0x1276, got 0x%04x", cpu.SP)
		}

		if cpu.isFlagsSet(FlagZero, FlagSubtract) && cpu.isFlagsNotSet(FlagCarry, FlagHalfCarry) {
			t.Errorf("Expected flags to be 0, got 0x%02x", cpu.F)
		}
	})
}

func incrementRegisterTest(regName string) func(*testing.T, Instructor) {
	return func(t *testing.T, instr Instructor) {
		// get the register
		reg := cpu.registerMap(regName)

		*reg = 0x42

		instr.Execute(cpu)

		if *cpu.registerMap(regName) != 0x43 {
			t.Errorf("Expected register to be 0x43, got 0x%02x", *reg)
		}

		if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) && cpu.isFlagsNotSet(FlagCarry) {
			t.Errorf("Expected flags to be 0, got 0x%02x", cpu.F)
		}

		// reset the flags
		cpu.F = 0

		// test the zero flag
		t.Run("Zero Flag", func(t *testing.T) {
			*cpu.registerMap(regName) = 0xFF

			instr.Execute(cpu)

			if cpu.isFlagsNotSet(FlagZero) {
				t.Errorf("Expected flags to be 0x80, got 0x%02x", cpu.F)
			}
		})
		// test the half carry flag
		t.Run("Half Carry Flag", func(t *testing.T) {
			*cpu.registerMap(regName) = 0x0F

			instr.Execute(cpu)

			if cpu.isFlagsNotSet(FlagHalfCarry) {
				t.Errorf("Expected flags to be 0x20, got 0x%02x", cpu.F)
			}
		})
		// test the carry flag is not set
		t.Run("Carry Flag", func(t *testing.T) {
			*cpu.registerMap(regName) = 0xFF

			instr.Execute(cpu)

			if cpu.isFlagsSet(FlagCarry) {
				t.Errorf("Expected flags to be 0x00, got 0x%02x", cpu.F)
			}
		})
	}
}

func decrementRegisterTest(regName string) func(*testing.T, Instructor) {
	return func(t *testing.T, instr Instructor) {
		// get the register
		reg := cpu.registerMap(regName)

		*reg = 0x42

		instr.Execute(cpu)

		if *cpu.registerMap(regName) != 0x41 {
			t.Errorf("Expected register to be 0x41, got 0x%02x", *reg)
		}

		if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) && cpu.isFlagsNotSet(FlagCarry) {
			t.Errorf("Expected flags to be 0, got 0x%02x", cpu.F)
		}

	}
}

func addRegisterTest(regName string) func(*testing.T, Instructor) {
	return func(t *testing.T, instr Instructor) {
		*cpu.registerMap(regName) = 0x22
		cpu.A = 0x42

		instr.Execute(cpu)

		if cpu.A != 0x64 && regName != "A" {
			t.Errorf("Expected A to be 0x64, got 0x%02x", cpu.A)
		} else if regName == "A" && cpu.A != 0x84 {
			t.Errorf("Expected A to be 0x84, got 0x%02x", cpu.A)
		}

		if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) && cpu.isFlagsNotSet(FlagCarry) {
			t.Errorf("Expected flags to be 0, got 0x%02x", cpu.F)
		}

		// reset the flags
		cpu.F = 0

		// test the zero flag
		t.Run("Zero Flag", func(t *testing.T) {
			if regName == "A" {
				// TODO handle this case
				return
			}
			*cpu.registerMap(regName) = 0xFF
			cpu.A = 0x01

			instr.Execute(cpu)

			if cpu.isFlagsNotSet(FlagZero) {
				t.Errorf("Expected flags to be 0x80, got 0x%02x", cpu.F)
			}
		})
		// test the half carry flag
		t.Run("Half Carry Flag", func(t *testing.T) {
			if regName == "A" {
				// TODO handle this case
				return
			}
			*cpu.registerMap(regName) = 0x0F
			cpu.A = 0x10

			instr.Execute(cpu)

			if cpu.isFlagsNotSet(FlagHalfCarry) {
				t.Errorf("Expected flags to be 0x20, got 0x%02x", cpu.F)
			}
		})
		// test the carry flag
		t.Run("Carry Flag", func(t *testing.T) {
			if regName == "A" {
				// TODO handle this case
				return
			}
			*cpu.registerMap(regName) = 0xFF
			cpu.A = 0x01

			instr.Execute(cpu)

			if cpu.isFlagsNotSet(FlagCarry) {
				t.Errorf("Expected flags to be 0x10, got 0x%02x", cpu.F)
			}
		})
	}
}

func addCarryRegisterTest(regName string) func(*testing.T, Instructor) {
	return func(t *testing.T, instr Instructor) {
		*cpu.registerMap(regName) = 0x22
		cpu.A = 0x42
		cpu.setFlag(FlagCarry)

		instr.Execute(cpu)

		if cpu.A != 0x65 && regName != "A" {
			t.Errorf("Expected A to be 0x65, got 0x%02x", cpu.A)
		} else if regName == "A" && cpu.A != 0x85 {
			t.Errorf("Expected A to be 0x85, got 0x%02x", cpu.A)
		}
	}
}

func subtractRegisterTest(regName string) func(*testing.T, Instructor) {
	return func(t *testing.T, instr Instructor) {
		*cpu.registerMap(regName) = 0x22
		cpu.A = 0x42

		instr.Execute(cpu)

		if cpu.A != 0x20 && regName != "A" {
			t.Errorf("Expected A to be 0x20, got 0x%02x", cpu.A)
		} else if regName == "A" && cpu.A != 0x00 {
			t.Errorf("Expected A to be 0x00, got 0x%02x", cpu.A)
		}

		// reset the flags
		cpu.F = 0

		// test the zero flag
		t.Run("Zero Flag", func(t *testing.T) {
			*cpu.registerMap(regName) = 0x42
			cpu.A = 0x42

			instr.Execute(cpu)

			if cpu.isFlagsNotSet(FlagZero) {
				t.Errorf("Expected flags to be 0x80, got 0x%02x", cpu.F)
			}
		})
		// test the half carry flag
		t.Run("Half Carry Flag", func(t *testing.T) {
			if regName == "A" {
				// TODO handle this case
				return
			}
			*cpu.registerMap(regName) = 0x42
			cpu.A = 0x10

			instr.Execute(cpu)

			if cpu.isFlagsNotSet(FlagHalfCarry) {
				t.Errorf("Expected flags to be 0x20, got 0x%02x", cpu.F)
			}
		})
		// test the carry flag
		t.Run("Carry Flag", func(t *testing.T) {
			if regName == "A" {
				// TODO handle this case
				return
			}
			*cpu.registerMap(regName) = 0x42
			cpu.A = 0x10

			instr.Execute(cpu)

			if cpu.isFlagsNotSet(FlagCarry) {
				t.Errorf("Expected flags to be 0x10, got 0x%02x", cpu.F)
			}
		})
	}
}

func subtractCarryRegisterTest(regName string) func(*testing.T, Instructor) {
	return func(t *testing.T, instr Instructor) {
		*cpu.registerMap(regName) = 0x22
		cpu.A = 0x42
		cpu.setFlag(FlagCarry)

		instr.Execute(cpu)

		if cpu.A != 0x1F && regName != "A" {
			t.Errorf("Expected A to be 0x1F, got 0x%02x", cpu.A)
		} else if regName == "A" && cpu.A != 0xFF {
			t.Errorf("Expected A to be 0xFF, got 0x%02x", cpu.A)
		}
	}
}

func incrementRegisterPairTest(regName string) func(*testing.T, Instructor) {
	return func(t *testing.T, instr Instructor) {
		// randomize the flags
		r := randomizeFlags(cpu)

		cpu.registerPairMap(regName).SetUint16(0x4242)
		instr.Execute(cpu)

		if cpu.registerPairMap(regName).Uint16() != 0x4243 {
			t.Errorf("Expected register pair to be 0x4243, got 0x%04x", *cpu.registerPairMap(regName))
		}

		// flags should be unchanged
		if cpu.F != r {
			t.Errorf("Expected flags to be 0x%02x, got 0x%02x", r, cpu.F)
		}
	}
}

func addRegisterPairHLTest(regName string) func(*testing.T, Instructor) {
	return func(t *testing.T, instr Instructor) {
		cpu.HL.SetUint16(0x4242)
		cpu.registerPairMap(regName).SetUint16(0x4242)

		instr.Execute(cpu)

		if cpu.HL.Uint16() != 0x8484 {
			t.Errorf("Expected HL to be 0x8484, got 0x%04x", cpu.HL)
		}
		// ensure correct flags are set
		if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) && cpu.isFlagsNotSet(FlagCarry) {
			t.Errorf("Expected flags to be 0, got 0x%02x", cpu.F)
		}

		// test the half carry flag
		t.Run("Half Carry Flag", func(t *testing.T) {
			cpu.HL.SetUint16(0x0F0F)
			cpu.registerPairMap(regName).SetUint16(0x0F0F)

			instr.Execute(cpu)

			if cpu.isFlagsNotSet(FlagHalfCarry) {
				t.Errorf("Expected flags to be 0x20, got 0x%02x", cpu.F)
			}
		})
		// test the carry flag
		t.Run("Carry Flag", func(t *testing.T) {
			if regName == "HL" {
				// TODO handle this case
				return
			}
			cpu.HL.SetUint16(0xFFFF)
			cpu.registerPairMap(regName).SetUint16(0x0001)

			instr.Execute(cpu)

			if cpu.isFlagsNotSet(FlagCarry) {
				t.Errorf("Expected flags to be 0x10, got 0x%02x", cpu.F)
			}
		})
		// test Zero flag remains unchanged
		t.Run("Zero Flag Unchanged", func(t *testing.T) {
			cpu.setFlag(FlagZero)
			cpu.HL.SetUint16(0x4242)
			cpu.registerPairMap(regName).SetUint16(0x4242)

			instr.Execute(cpu)

			if cpu.isFlagsNotSet(FlagZero) {
				t.Errorf("Expected flags to be 0x80, got 0x%02x", cpu.F)
			}
		})

	}
}

func decrementRegisterPairTest(regName string) func(*testing.T, Instructor) {
	return func(t *testing.T, instr Instructor) {
		// randomize the flags
		r := randomizeFlags(cpu)
		cpu.registerPairMap(regName).SetUint16(0x4242)

		instr.Execute(cpu)

		if cpu.registerPairMap(regName).Uint16() != 0x4241 {
			t.Errorf("Expected register pair to be 0x4241, got 0x%04x", *cpu.registerPairMap(regName))
		}

		// flags should be unchanged
		if cpu.F != r {
			t.Errorf("Expected flags to be 0x%02x, got 0x%02x", r, cpu.F)
		}
	}
}

var registerNames = map[uint8]string{
	0x0: "B",
	0x1: "C",
	0x2: "D",
	0x3: "E",
	0x4: "H",
	0x5: "L",
	0x6: "(HL)",
	0x7: "A",
}

func randomizeFlags(cpu *CPU) uint8 {
	r := uint8(rand.Intn(255))
	cpu.F = r & 0xF0
	return r
}

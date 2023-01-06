package cpu

import "testing"

func TestInstruction_Shift(t *testing.T) {
	// 0x20 - 0x27 SLA r (Shift Left Arithmetic)
	for i, regName := range registerNames {
		if i == 6 {
			continue
		}
		testInstructionCB(t, "SLA "+regName, 0x20+i, func(t *testing.T, instr Instructor) {
			// set register to value that will cause carry flag to be set
			*cpu.registerMap(regName) = 0x80

			// execute instruction
			instr.Execute(cpu, nil)

			// check that register was shifted left into carry
			if *cpu.registerMap(regName) != 0x00 {
				t.Errorf("expected register to be shifted left into carry, got %02X", *cpu.registerMap(regName))
			}

			// check that carry flag was set
			if !cpu.isFlagSet(FlagCarry) {
				t.Errorf("expected carry flag to be set, got unset")
			}

			// test register was shifted left (no carry)
			testInstructionCB(t, "No Carry", 0x20+i, func(t *testing.T, instr Instructor) {
				// set register to value that will not cause carry flag to be set
				*cpu.registerMap(regName) = 0x40

				// execute instruction
				instr.Execute(cpu, nil)

				// check that register was shifted left
				if *cpu.registerMap(regName) != 0x80 {
					t.Errorf("expected register to be shifted left, got %02X", *cpu.registerMap(regName))
				}

				// check that carry flag was unset
				if cpu.isFlagSet(FlagCarry) {
					t.Errorf("expected carry flag to be unset, got set")
				}
			})
		})

	}
	// 0x28 - 0x2F SRA r (Shift Right Arithmetic)
	for i, regName := range registerNames {
		if i == 6 {
			continue
		}
		testInstructionCB(t, "SRA "+regName, 0x28+uint8(i), func(t *testing.T, inst Instructor) {
			// set register to value that will cause carry flag to be set taking into account that MSB is preserved
			*cpu.registerMap(regName) = 0x81

			// execute instruction
			inst.Execute(cpu, nil)

			// check that register was shifted right into carry and MSB was preserved
			if *cpu.registerMap(regName) != 0xC0 {
				t.Errorf("expected register to be shifted right into carry and MSB preserved, got %02X", *cpu.registerMap(regName))
			}

			// check that carry flag was set
			if !cpu.isFlagSet(FlagCarry) {
				t.Errorf("expected carry flag to be set, got unset")
			}

			// test register was shifted right (no carry)
			testInstructionCB(t, "No Carry", 0x28+uint8(i), func(t *testing.T, instr Instructor) {
				// set register to value that will not cause carry flag to be set
				*cpu.registerMap(regName) = 0x40

				// execute instruction
				instr.Execute(cpu, nil)

				// check that register was shifted right
				if *cpu.registerMap(regName) != 0x20 {
					t.Errorf("expected register to be shifted right, got %02X", *cpu.registerMap(regName))
				}

				// check that carry flag was unset
				if cpu.isFlagSet(FlagCarry) {
					t.Errorf("expected carry flag to be unset, got set")
				}
			})
		})
	}
	// 0x38 - 0x3F SRL r (Shift Right Logical)
	for i, regName := range registerNames {
		if i == 6 {
			continue
		}

		testInstructionCB(t, "SRL "+regName, 0x38+uint8(i), func(t *testing.T, inst Instructor) {
			// set register to value that will cause carry flag to be set
			*cpu.registerMap(regName) = 0x81

			// execute instruction
			inst.Execute(cpu, nil)

			// check that register was shifted right into carry
			if *cpu.registerMap(regName) != 0x40 {
				t.Errorf("expected register to be shifted right into carry, got %02X", *cpu.registerMap(regName))
			}

			// check that carry flag was set
			if !cpu.isFlagSet(FlagCarry) {
				t.Errorf("expected carry flag to be set, got unset")
			}

			// test register was shifted right (no carry)
			testInstructionCB(t, "No Carry", 0x38+uint8(i), func(t *testing.T, instr Instructor) {
				// set register to value that will not cause carry flag to be set
				*cpu.registerMap(regName) = 0x40

				// execute instruction
				instr.Execute(cpu, nil)

				// check that register was shifted right
				if *cpu.registerMap(regName) != 0x20 {
					t.Errorf("expected register to be shifted right, got %02X", *cpu.registerMap(regName))
				}

				// check that carry flag was unset
				if cpu.isFlagSet(FlagCarry) {
					t.Errorf("expected carry flag to be unset, got set")
				}
			})
		})
	}
}

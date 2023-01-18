package cpu

import (
	"testing"
)

func TestInstruction_Rotate(t *testing.T) {
	// 0x07 - RLCA
	testInstruction(t, "RLCA", 0x07, func(t *testing.T, instruction Instructor) {
		cpu.A = 0x80
		cpu.setFlag(FlagCarry)
		instruction.Execute(cpu)

		if cpu.A != 0x01 {
			t.Errorf("Expected A to be 0x01, got 0x%02X", cpu.A)
		}
		if !cpu.isFlagSet(FlagCarry) {
			t.Errorf("Expected Carry to be set, got not set")
		}
		// all other flags should be reset
		if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) {
			t.Errorf("Expected all other flags to be reset, got not reset")
		}
	})
	// 0x0F - RRCA
	testInstruction(t, "RRCA", 0x0F, func(t *testing.T, instruction Instructor) {
		cpu.A = 0x01
		cpu.setFlag(FlagCarry)
		instruction.Execute(cpu)

		if cpu.A != 0x80 {
			t.Errorf("Expected A to be 0x80, got 0x%02X", cpu.A)
		}
		if !cpu.isFlagSet(FlagCarry) {
			t.Errorf("Expected Carry to be set, got not set")
		}
		// all other flags should be reset
		if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) {
			t.Errorf("Expected all other flags to be reset, got not reset")
		}
	})
	// 0x17 - RLA
	testInstruction(t, "RLA", 0x17, func(t *testing.T, instruction Instructor) {
		if cpu.isFlagSet(FlagCarry) {
			t.Errorf("Expected Carry to be not set, got set")
		}
		cpu.A = 0b10101010

		instruction.Execute(cpu)

		// ensure A was rotated
		if cpu.A != 0b01010100 {
			t.Errorf("Expected A to be 0b01010100, got 0b%08b", cpu.A)
		}

		// ensure carry flag wasn't set (since bit 7 was 0)
		if !cpu.isFlagSet(FlagCarry) {
			t.Errorf("Expected Carry to be unset, got set")
		}

		// all other flags should be reset
		if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) {
			t.Errorf("Expected all other flags to be reset, got not reset")
		}

		// test through a carry
		t.Run("Through Carry", func(t *testing.T) {
			cpu.setFlag(FlagCarry)
			cpu.A = 0b01000000
			instruction.Execute(cpu)

			// ensure A was rotated
			if cpu.A != 0b10000001 {
				t.Errorf("Expected A to be 0b10000000, got 0b%08b", cpu.A)
			}

			// ensure carry flag was not set
			if cpu.isFlagSet(FlagCarry) {
				t.Errorf("Expected Carry to be not set, got set")
			}

			// all other flags should be reset
			if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) {
				t.Errorf("Expected all other flags to be reset, got not reset")
			}
		})
	})
	// 0x1F - RRA
	testInstruction(t, "RRA", 0x1F, func(t *testing.T, instruction Instructor) {
		cpu.A = 0b10101010

		instruction.Execute(cpu)

		// ensure A was rotated
		if cpu.A != 0b01010101 {
			t.Errorf("Expected A to be 0b11010101, got 0b%08b", cpu.A)
		}

		// ensure carry flag wasn't set
		if !cpu.isFlagSet(FlagCarry) {
			t.Errorf("Expected Carry to be unset, got set")
		}

		// all other flags should be reset
		if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) {
			t.Errorf("Expected all other flags to be reset, got not reset")
		}

		// test no carry
		t.Run("No Carry", func(t *testing.T) {
			cpu.A = 0b00000001
			instruction.Execute(cpu)

			// ensure A was rotated
			if cpu.A != 0b10000000 {
				t.Errorf("Expected A to be 0b00000000, got 0b%08b", cpu.A)
			}

			// ensure carry flag was not set
			if cpu.isFlagSet(FlagCarry) {
				t.Errorf("Expected Carry to be not set, got set")
			}

			// all other flags should be reset
			if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) {
				t.Errorf("Expected all other flags to be reset, got not reset")
			}
		})
	})
}

func TestInstruction_16Bit_Rotate(t *testing.T) {
	// 0x00 - 0x07 - RLC r (Exclude (HL))
	for i, regName := range registerNames {
		if i == 6 {
			continue
		}

		testInstructionCB(t, "RLC "+regName, 0x00+byte(i), func(t *testing.T, instruction Instructor) {
			*cpu.registerMap(regName) = 0x80

			instruction.Execute(cpu)

			// ensure register was rotated
			if *cpu.registerMap(regName) != 0x01 {
				t.Errorf("Expected register to be 0x01, got 0x%02X", *cpu.registerMap(regName))
			}

			// ensure carry flag was set
			if !cpu.isFlagSet(FlagCarry) {
				t.Errorf("Expected Carry to be set, got not set")
			}

			// all other flags should be reset
			if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) {
				t.Errorf("Expected all other flags to be reset, got not reset")
			}

			// test no carry
			t.Run("No Carry", func(t *testing.T) {
				*cpu.registerMap(regName) = 0x40
				instruction.Execute(cpu)

				// ensure register was rotated
				if *cpu.registerMap(regName) != 0x80 {
					t.Errorf("Expected register to be 0x80, got 0x%02X", *cpu.registerMap(regName))
				}

				// ensure carry flag was not set
				if cpu.isFlagSet(FlagCarry) {
					t.Errorf("Expected Carry to be not set, got set")
				}

				// all other flags should be reset
				if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) {
					t.Errorf("Expected all other flags to be reset, got not reset")
				}
			})

			// test zero
			t.Run("Zero", func(t *testing.T) {
				*cpu.registerMap(regName) = 0x00
				instruction.Execute(cpu)

				// ensure register was rotated
				if *cpu.registerMap(regName) != 0x00 {
					t.Errorf("Expected register to be 0x00, got 0x%02X", *cpu.registerMap(regName))
				}

				// ensure zero flag was set
				if !cpu.isFlagSet(FlagZero) {
					t.Errorf("Expected Zero to be set, got not set")
				}

				// all other flags should be reset
				if cpu.isFlagsSet(FlagCarry, FlagSubtract, FlagHalfCarry) {
					t.Errorf("Expected all other flags to be reset, got not reset")
				}
			})
		})
	}
	// 0x08 - 0x0F - RRC r (Exclude (HL))
	for i, regName := range registerNames {
		if i == 6 {
			continue
		}

		testInstructionCB(t, "RRC "+regName, 0x08+byte(i), func(t *testing.T, instruction Instructor) {
			*cpu.registerMap(regName) = 0x01

			instruction.Execute(cpu)

			// ensure register was rotated
			if *cpu.registerMap(regName) != 0x80 {
				t.Errorf("Expected register to be 0x80, got 0x%02X", *cpu.registerMap(regName))
			}

			// ensure carry flag was set
			if !cpu.isFlagSet(FlagCarry) {
				t.Errorf("Expected Carry to be set, got not set")
			}

			// all other flags should be reset
			if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) {
				t.Errorf("Expected all other flags to be reset, got not reset")
			}

			// test no carry
			t.Run("No Carry", func(t *testing.T) {
				*cpu.registerMap(regName) = 0x80
				instruction.Execute(cpu)

				// ensure register was rotated
				if *cpu.registerMap(regName) != 0x40 {
					t.Errorf("Expected register to be 0x40, got 0x%02X", *cpu.registerMap(regName))
				}

				// ensure carry flag was not set
				if cpu.isFlagSet(FlagCarry) {
					t.Errorf("Expected Carry to be not set, got set")
				}

				// all other flags should be reset
				if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) {
					t.Errorf("Expected all other flags to be reset, got not reset")
				}
			})

			// test zero
			t.Run("Zero", func(t *testing.T) {
				*cpu.registerMap(regName) = 0x00
				instruction.Execute(cpu)

				// ensure register was rotated
				if *cpu.registerMap(regName) != 0x00 {
					t.Errorf("Expected register to be 0x00, got 0x%02X", *cpu.registerMap(regName))
				}

				// ensure zero flag was set
				if !cpu.isFlagSet(FlagZero) {
					t.Errorf("Expected Zero to be set, got not set")
				}

				// all other flags should be reset
				if cpu.isFlagsSet(FlagCarry, FlagSubtract, FlagHalfCarry) {
					t.Errorf("Expected all other flags to be reset, got not reset")
				}
			})
		})
	}
	// 0x10 - 0x17 - RL r (Exclude (HL))
	for i, regName := range registerNames {
		if i == 6 {
			continue
		}

		testInstructionCB(t, "RL "+regName, 0x10+byte(i), func(t *testing.T, instruction Instructor) {
			cpu.setFlag(FlagCarry)
			*cpu.registerMap(regName) = 0x80

			instruction.Execute(cpu)

			// ensure register was rotated
			if *cpu.registerMap(regName) != 0x01 {
				t.Errorf("Expected register to be 0x01, got 0x%02X", *cpu.registerMap(regName))
			}

			// ensure carry flag was set
			if !cpu.isFlagSet(FlagCarry) {
				t.Errorf("Expected Carry to be set, got not set")
			}

			// all other flags should be reset
			if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) {
				t.Errorf("Expected all other flags to be reset, got not reset")
			}

			// test no carry
			t.Run("No Carry", func(t *testing.T) {
				cpu.setFlag(FlagCarry)
				*cpu.registerMap(regName) = 0x01
				instruction.Execute(cpu)

				// ensure register was rotated
				if *cpu.registerMap(regName) != 0x02 {
					t.Errorf("Expected register to be 0x02, got 0x%02X", *cpu.registerMap(regName))
				}

				// ensure carry flag was not set
				if cpu.isFlagSet(FlagCarry) {
					t.Errorf("Expected Carry to be not set, got set")
				}

				// all other flags should be reset
				if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) {
					t.Errorf("Expected all other flags to be reset, got not reset")
				}
			})

			// test zero
			t.Run("Zero", func(t *testing.T) {
				*cpu.registerMap(regName) = 0x00
				instruction.Execute(cpu)

				// ensure register was rotated
				if *cpu.registerMap(regName) != 0x00 {
					t.Errorf("Expected register to be 0x00, got 0x%02X", *cpu.registerMap(regName))
				}

				// ensure zero flag was set
				if !cpu.isFlagSet(FlagZero) {
					t.Errorf("Expected Zero to be set, got not set")
				}

				// all other flags should be reset
				if cpu.isFlagsSet(FlagCarry, FlagSubtract, FlagHalfCarry) {
					t.Errorf("Expected all other flags to be reset, got not reset")
				}

			})
		})
	}
	// 0x18 - 0x1F - RR r (Exclude (HL))
	for i, regName := range registerNames {
		if i == 6 {
			continue
		}

		testInstructionCB(t, "RR "+regName, 0x18+byte(i), func(t *testing.T, instruction Instructor) {
			cpu.setFlag(FlagCarry)
			*cpu.registerMap(regName) = 0x01

			instruction.Execute(cpu)

			// ensure register was rotated
			if *cpu.registerMap(regName) != 0x80 {
				t.Errorf("Expected register to be 0x80, got 0x%02X", *cpu.registerMap(regName))
			}

			// ensure carry flag was set
			if !cpu.isFlagSet(FlagCarry) {
				t.Errorf("Expected Carry to be set, got not set")
			}

			// all other flags should be reset
			if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) {
				t.Errorf("Expected all other flags to be reset, got not reset")
			}

			// test no carry
			t.Run("No Carry", func(t *testing.T) {
				*cpu.registerMap(regName) = 0x80
				instruction.Execute(cpu)

				// ensure register was rotated
				if *cpu.registerMap(regName) != 0x40 {
					t.Errorf("Expected register to be 0x40, got 0x%02X", *cpu.registerMap(regName))
				}

				// ensure carry flag was not set
				if cpu.isFlagSet(FlagCarry) {
					t.Errorf("Expected Carry to be not set, got set")
				}

				// all other flags should be reset
				if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry) {
					t.Errorf("Expected all other flags to be reset, got not reset")
				}
			})

			// test zero
			t.Run("Zero", func(t *testing.T) {
				*cpu.registerMap(regName) = 0x00
				instruction.Execute(cpu)

				// ensure register was rotated
				if *cpu.registerMap(regName) != 0x00 {
					t.Errorf("Expected register to be 0x00, got 0x%02X", *cpu.registerMap(regName))
				}

				// ensure zero flag was set
				if !cpu.isFlagSet(FlagZero) {
					t.Errorf("Expected Zero to be set, got not set")
				}

				// all other flags should be reset
				if cpu.isFlagsSet(FlagCarry, FlagSubtract, FlagHalfCarry) {
					t.Errorf("Expected all other flags to be reset, got not reset")
				}

			})
		})
	}
}

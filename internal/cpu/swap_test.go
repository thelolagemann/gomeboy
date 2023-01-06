package cpu

import "testing"

func TestSwap(t *testing.T) {
	t.Run("zeroSwap", func(t *testing.T) {
		for _, test := range []struct {
			name string
			reg  *Register
			want uint8
		}{
			{"swapA", &cpu.A, 0x00},
			{"swapB", &cpu.B, 0x00},
			{"swapC", &cpu.C, 0x00},
			{"swapD", &cpu.D, 0x00},
			{"swapE", &cpu.E, 0x00},
			{"swapH", &cpu.H, 0x00},
			{"swapL", &cpu.L, 0x00},
		} {
			t.Run(test.name, func(t *testing.T) {
				*test.reg = 0x00
				cpu.swap(test.reg)
				if *test.reg != test.want {
					t.Errorf("got %02X, want %02X", *test.reg, test.want)
				}
				if !cpu.isFlagSet(FlagZero) {
					t.Errorf("expected zero flag to be set, got unset")
				}
				cpu.clearFlag(FlagZero)
			})
		}
	})
	t.Run("nonZeroSwap", func(t *testing.T) {
		for _, test := range []struct {
			name string
			reg  *Register
			want uint8
		}{
			{"swapA", &cpu.A, 0x12},
			{"swapB", &cpu.B, 0x12},
			{"swapC", &cpu.C, 0x12},
			{"swapD", &cpu.D, 0x12},
			{"swapE", &cpu.E, 0x12},
			{"swapH", &cpu.H, 0x12},
			{"swapL", &cpu.L, 0x12},
		} {
			t.Run(test.name, func(t *testing.T) {
				*test.reg = 0x21
				cpu.swap(test.reg)
				if *test.reg != test.want {
					t.Errorf("got %02X, want %02X", *test.reg, test.want)
				}
				if cpu.isFlagSet(FlagZero) {
					t.Errorf("expected zero flag to be unset, got set")
				}
				cpu.clearFlag(FlagZero)
			})
		}
	})
}

func TestInstruction_Swap(t *testing.T) {
	// 0x30 - 0x37 - SWAP r (Exclude (HL))
	for i, regName := range registerNames {
		if i == 6 {
			continue
		}
		testInstructionCB(t, "SWAP "+regName, 0x30+i, func(t *testing.T, instr Instructor) {
			randomizeFlags(cpu)
			*cpu.registerMap(regName) = 0x21

			instr.Execute(cpu, nil)

			if *cpu.registerMap(regName) != 0x12 {
				t.Errorf("got %02X, want %02X", *cpu.registerMap(regName), 0x12)
			}

			// all other flags should be reset
			if cpu.isFlagsSet(FlagZero, FlagSubtract, FlagHalfCarry, FlagCarry) {
				t.Errorf("expected all flags to be reset, got %02X", cpu.F)
			}

			// test zero flag
			t.Run("Zero Flag", func(t *testing.T) {
				randomizeFlags(cpu)
				*cpu.registerMap(regName) = 0x00

				instr.Execute(cpu, nil)

				if !cpu.isFlagSet(FlagZero) {
					t.Errorf("expected zero flag to be set, got unset")
				}

				// all other flags should be reset
				if cpu.isFlagsSet(FlagSubtract, FlagHalfCarry, FlagCarry) {
					t.Errorf("expected all flags to be reset, got %02X", cpu.F)
				}
			})
		})
	}
}

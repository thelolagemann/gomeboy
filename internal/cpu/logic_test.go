package cpu

import (
	"math/rand"
	"testing"
)

func TestLogicInstructions(t *testing.T) {
	// 0xA0 - 0xA7 (Exclude 0xA6) - AND B, C, D, E, H, L, A
	for i, regName := range registerNames {
		if i == 6 {
			continue
		}
		testInstruction(t, "AND "+regName, 0xA0+uint8(i), andRegisterTest(regName))
	}
	// 0xA8 - 0xAF (Exclude 0xAE) - XOR B, C, D, E, H, L, A
	for i, regName := range registerNames {
		if i == 6 {
			continue
		}
		testInstruction(t, "XOR "+regName, 0xA8+uint8(i), xorRegisterTest(regName))
	}
	// 0xB0 - 0xB7 (Exclude 0xB6) - OR B, C, D, E, H, L, A
	for i, regName := range registerNames {
		if i == 6 {
			continue
		}
		testInstruction(t, "OR "+regName, 0xB0+uint8(i), orRegisterTest(regName))
	}
}

func andRegisterTest(regName string) func(*testing.T, Instruction) {
	return func(t *testing.T, instruction Instruction) {
		rand1 := uint8(rand.Intn(255))
		rand2 := uint8(rand.Intn(255))

		cpu.A = rand1
		*cpu.registerMap(regName) = rand2

		instruction.Execute(cpu, nil)

		if cpu.A != rand1&rand2 {
			t.Errorf("Expected A to be %d, got %d", rand1&rand2, cpu.A)
		}

		if cpu.isFlagSet(FlagZero) != (cpu.A == 0) {
			t.Errorf("Expected Zero flag to be %t, got %t", cpu.A == 0, cpu.isFlagSet(FlagZero))
		}

		if cpu.isFlagSet(FlagSubtract) {
			t.Errorf("Expected Subtract flag to be false, got true")
		}

		if !cpu.isFlagSet(FlagHalfCarry) {
			t.Errorf("Expected Half Carry flag to be true, got false")
		}

		if cpu.isFlagSet(FlagCarry) {
			t.Errorf("Expected Carry flag to be false, got true")
		}
	}
}

func xorRegisterTest(regName string) func(*testing.T, Instruction) {
	return func(t *testing.T, instruction Instruction) {
		rand1 := uint8(rand.Intn(255))
		rand2 := uint8(rand.Intn(255))

		cpu.A = rand1
		*cpu.registerMap(regName) = rand2

		instruction.Execute(cpu, nil)

		if cpu.A != rand1^rand2 {
			t.Errorf("Expected A to be %d, got %d", rand1^rand2, cpu.A)
		}

		if cpu.isFlagSet(FlagZero) != (cpu.A == 0) {
			t.Errorf("Expected Zero flag to be %t, got %t", cpu.A == 0, cpu.isFlagSet(FlagZero))
		}

		if cpu.isFlagSet(FlagSubtract) {
			t.Errorf("Expected Subtract flag to be false, got true")
		}

		if cpu.isFlagSet(FlagHalfCarry) {
			t.Errorf("Expected Half Carry flag to be false, got true")
		}

		if cpu.isFlagSet(FlagCarry) {
			t.Errorf("Expected Carry flag to be false, got true")
		}
	}
}

func orRegisterTest(regName string) func(*testing.T, Instruction) {
	return func(t *testing.T, instruction Instruction) {
		rand1 := uint8(rand.Intn(255))
		rand2 := uint8(rand.Intn(255))

		cpu.A = rand1
		*cpu.registerMap(regName) = rand2

		instruction.Execute(cpu, nil)

		if cpu.A != rand1|rand2 {
			t.Errorf("Expected A to be %d, got %d", rand1|rand2, cpu.A)
		}

		if cpu.isFlagSet(FlagZero) != (cpu.A == 0) {
			t.Errorf("Expected Zero flag to be %t, got %t", cpu.A == 0, cpu.isFlagSet(FlagZero))
		}

		if cpu.isFlagSet(FlagSubtract) {
			t.Errorf("Expected Subtract flag to be false, got true")
		}

		if cpu.isFlagSet(FlagHalfCarry) {
			t.Errorf("Expected Half Carry flag to be false, got true")
		}

		if cpu.isFlagSet(FlagCarry) {
			t.Errorf("Expected Carry flag to be false, got true")
		}
	}
}

func cpRegisterTest(regName string) func(*testing.T, Instruction) {
	return func(t *testing.T, instruction Instruction) {

	}
}

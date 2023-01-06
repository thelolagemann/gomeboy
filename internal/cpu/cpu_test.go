package cpu

import "testing"

func TestInstruction_Control(t *testing.T) {
	// 0x00 - NOP
	testInstruction(t, "NOP", 0x00, func(t *testing.T, instruction Instructor) {
		instruction.Execute(cpu, nil)
	})
	// 0x10 - STOP
	testInstruction(t, "STOP", 0x10, func(t *testing.T, instruction Instructor) {
		cpu.stopped = false
		instruction.Execute(cpu, nil)

		if !cpu.stopped {
			t.Errorf("Expected CPU to be stopped, got running")
		}
	})
	// 0x76 - HALT
	testInstruction(t, "HALT", 0x76, func(t *testing.T, instruction Instructor) {
		cpu.halted = false
		instruction.Execute(cpu, nil)

		if !cpu.halted {
			t.Errorf("Expected CPU to be halted, got running")
		}
	})
	// 0xF3 - DI
	testInstruction(t, "DI", 0xF3, func(t *testing.T, instruction Instructor) {

	})
	// 0xFB - EI
	testInstruction(t, "EI", 0xFB, func(t *testing.T, instruction Instructor) {

	})
}

package cpu

import (
	"testing"
)

func TestBit(t *testing.T) {
	t.Run("set", func(t *testing.T) {
		cpu.A = cpu.setBit(cpu.A, 0)
		if cpu.A != 0x01 {
			t.Errorf("expected 0x02, got 0x%02x", cpu.A)
		}
	})
	t.Run("clear", func(t *testing.T) {
		cpu.A = cpu.clearBit(cpu.A, 0)
		if cpu.A != 0x00 {
			t.Errorf("expected A to be 0x00, got 0x%02X", cpu.A)
		}
	})
	t.Run("test", func(t *testing.T) {
		cpu.testBit(cpu.A, 0)
		if !cpu.isFlagSet(FlagZero) {
			t.Errorf("expected zero flag to be set, got unset")
		}
		cpu.A = 0x01
		cpu.testBit(cpu.A, 0)
		if cpu.isFlagSet(FlagZero) {
			t.Errorf("expected zero flag to be unset, got set")
		}
	})
}

func TestInstruction_16Bit_Bits(t *testing.T) {
	// 0x40 - 0x7F BIT b,r

}

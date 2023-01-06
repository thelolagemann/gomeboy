package cpu

import "testing"

func TestFlag(t *testing.T) {
	t.Run("clear", func(t *testing.T) {
		for i := FlagZero; i <= FlagCarry; i++ {
			cpu.clearFlag(i)
			if cpu.isFlagSet(i) {
				t.Errorf("expected flag %d to be unset, got set", i)
			}
		}
	})
	t.Run("set", func(t *testing.T) {
		for i := FlagZero; i <= FlagCarry; i++ {
			cpu.setFlag(i)
			if !cpu.isFlagSet(i) {
				t.Errorf("expected flag %d to be set, got unset", i)
			}
		}
	})
	t.Run("isFlagSet", func(t *testing.T) {
		for i := FlagZero; i <= FlagCarry; i++ {
			cpu.clearFlag(i)
			if cpu.isFlagSet(i) {
				t.Errorf("expected flag %d to be unset, got set", i)
			}
			cpu.setFlag(i)
			if !cpu.isFlagSet(i) {
				t.Errorf("expected flag %d to be set, got unset", i)
			}
		}
	})
}

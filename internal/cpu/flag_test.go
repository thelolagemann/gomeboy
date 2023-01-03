package cpu

import "testing"

func TestFlag(t *testing.T) {
	c := NewCPU(nil)
	t.Run("clear", func(t *testing.T) {
		for i := FlagZero; i <= FlagCarry; i++ {
			c.clearFlag(i)
			if c.isFlagSet(i) {
				t.Errorf("expected flag %d to be unset, got set", i)
			}
		}
	})
	t.Run("set", func(t *testing.T) {
		for i := FlagZero; i <= FlagCarry; i++ {
			c.setFlag(i)
			if !c.isFlagSet(i) {
				t.Errorf("expected flag %d to be set, got unset", i)
			}
		}
	})
	t.Run("isFlagSet", func(t *testing.T) {
		for i := FlagZero; i <= FlagCarry; i++ {
			c.clearFlag(i)
			if c.isFlagSet(i) {
				t.Errorf("expected flag %d to be unset, got set", i)
			}
			c.setFlag(i)
			if !c.isFlagSet(i) {
				t.Errorf("expected flag %d to be set, got unset", i)
			}
		}
	})
}

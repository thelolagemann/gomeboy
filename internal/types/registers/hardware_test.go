package registers

import "testing"

func BenchmarkHardware_Value(b *testing.B) {
	// create a new hardware
	h := NewHardware(0x0000)
	h.value = 0x01

	b.Run("func", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			x := h.Value()
			if x != 1 {
				b.Fatal("unexpected value")
			}
		}
	})
	b.Run("field", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			x := h.value
			if x != 1 {
				b.Fatal("unexpected value")
			}
		}
	})
}

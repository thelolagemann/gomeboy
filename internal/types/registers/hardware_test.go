package registers

import "testing"

func BenchmarkHardware_Value(b *testing.B) {
	// create a new hardware
	h := NewHardware(0x0000)

	b.Run("func", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = h.Value()
		}
	})
	b.Run("field", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = h.value
		}
	})
}
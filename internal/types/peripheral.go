package types

// Peripheral is a peripheral device that can be connected to the CPU,
// such as the joypad, the serial port, the timer, etc. The CPU will
// call the Tick method of the peripheral device every time a tick
// occurs.
type Peripheral interface {
	// Tick should be called every time a tick occurs, to allow the
	// peripheral to update its state. Each call to Tick should
	// advance the peripheral by one tick.
	Tick()
	// HasDoubleSpeed returns true if the peripheral is affected by
	// the double speed mode. If this returns true, the CPU will
	// call Tick twice as fast, when the double speed mode is
	// enabled. (CGB only)
	HasDoubleSpeed() bool
}

// Ticker is a function that can be called to advance the state of
// the peripheral by one tick.
type Ticker func()

type Address struct {
	// Read is a function that is called when the CPU reads from
	// the address.
	Read func(address uint16) uint8
	// Write is a function that is called when the CPU writes to
	// the address.
	Write func(address uint16, value uint8)
}

func Unreadable(address uint16) uint8 {
	return 0xFF
}

package types

// Peripheral is a peripheral device that can be connected to the CPU,
// such as the joypad, the serial port, the timer, etc. The CPU will
// call the Step method of the peripheral device every time it executes
// the Fetch-Decode-Execute cycle. This should advance the peripheral
// device by the Cl
type Peripheral interface {
	// Tick is called by the CPU every time it executes the Fetch-Decode-Execute
	// cycle. This should advance the peripheral device by the Clocks
	// specified by the ClocksPerStep method.
	Tick()
}

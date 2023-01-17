package types

// Peripheral is a peripheral device that can be connected to the CPU,
// such as the joypad, the serial port, the timer, etc. The CPU will
// call the Step method of the peripheral device every time it executes
// the Fetch-Decode-Execute cycle. This should advance the peripheral
// device by the Cl
type Peripheral interface {
	// Step advances the peripheral device by the given number of cycles.
	Step(cycles uint8)
	// Clock returns the clock speed of the hardware. This is used to
	// calculate the number of cycles that have passed since the last
	// call to Step.
	Clock() uint8
}

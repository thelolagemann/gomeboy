package emulator

// Status represents the status of the emulator's
// CPU. It can be one of the following:
//
//   - Running
//   - Halted
//   - Errored
type Status int

const (
	// Running represents the status of the
	// CPU when it is running.
	Running Status = iota
	// Halted represents the status of the
	// CPU when it has halted.
	Halted
	// Errored represents the status of the
	// CPU when it has encountered an unexpected
	// error.
	Errored
)

func (s Status) String() string {
	switch s {
	case Running:
		return "Running"
	case Halted:
		return "Halted"
	case Errored:
		return "Errored"
	default:
		return "Unknown"
	}
}

func (s Status) IsRunning() bool {
	return s == Running
}

func (s Status) IsHalted() bool {
	return s == Halted
}

func (s Status) IsErrored() bool {
	return s == Errored
}

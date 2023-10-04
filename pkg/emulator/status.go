package emulator

// State represents the status of the emulator.
// It can be one of the following:
//
//   - Running
//   - Paused
type State int

const (
	// Running represents the status of the
	// emulator when it is running.
	Running State = iota
	// Paused represents the status of the
	// emulator when it is paused.
	Paused
	// Stopped represents the status of the
	// emulator when it is stopped.
	Stopped
)

func (s State) String() string {
	switch s {
	case Running:
		return "Running"
	case Paused:
		return "Paused"
	case Stopped:
		return "Stopped"
	default:
		return "Unknown"
	}
}

func (s State) IsRunning() bool {
	return s == Running
}

func (s State) IsPaused() bool {
	return s == Paused
}

func (s State) IsStopped() bool {
	return s == Stopped
}

// Status represents the status of the emulator's
// CPU. It can be one of the following:
//
//   - Execution
//   - Halted
//   - Errored
type Status int

const (
	// Execution represents the status of the
	// CPU when it is executing instructions.
	Execution Status = iota
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
	case Execution:
		return "Executing"
	case Halted:
		return "Halted"
	case Errored:
		return "Errored"
	default:
		return "Unknown"
	}
}

func (s Status) IsExecuting() bool {
	return s == Execution
}

func (s Status) IsHalted() bool {
	return s == Halted
}

func (s Status) IsErrored() bool {
	return s == Errored
}

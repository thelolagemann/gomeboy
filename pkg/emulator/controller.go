package emulator

// Controller defines the interface contract for an Emulator to
// implement in order for a display.Driver to be able to control
// it.
type Controller interface {
	LoadROM(string) error
	Pause()
	Resume()
	Paused() bool
	Initialised() bool
}

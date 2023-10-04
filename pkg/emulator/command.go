package emulator

// CommandPacket is a command packet that is sent to the
// emulator to control it.
type CommandPacket struct {
	Command Command
	Data    []byte
}

// Command is a command that is sent to the emulator to
// control it.
type Command int

// ResponsePacket is a response packet that is sent
// from the emulator to the client.
type ResponsePacket struct {
	Command Command
	Data    []byte
	Error   error
}

const (
	// CommandPause pauses the emulator.
	CommandPause Command = iota
	// CommandResume resumes the emulator.
	CommandResume
	// CommandClose closes the emulator.
	CommandClose
	// CommandReset resets the emulator.
	CommandReset
	// CommandLoadROM loads a ROM into the emulator.
	CommandLoadROM
	// CommandLoadSave loads a save file into the emulator.
	CommandLoadSave
	// CommandSetSpeed sets the speed of the emulator.
	CommandSetSpeed
)

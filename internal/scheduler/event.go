//go:generate go run golang.org/x/tools/cmd/stringer -type=EventType -output=event_string.go
package scheduler

type EventType uint8

const (
	APUFrameSequencer EventType = iota
	APUFrameSequencer2
	APUChannel1
	APUChannel2
	APUChannel3
	APUSample

	EIPending
	EIHaltDelay

	PPUStartHBlank
	PPUHBlank
	PPUHBlankInterrupt
	PPUBeginFIFO
	PPUFIFOTransfer
	PPUStartOAMSearch
	PPUEndFrame
	PPUContinueOAMSearch
	PPUPrepareEndOAMSearch
	PPUEndOAMSearch
	PPULine153Continue
	PPULine153End
	PPUStartVBlank
	PPUContinueVBlank
	PPUVRAMTransfer
	PPUStartGlitchedLine0
	PPUMiddleGlitchedLine0
	PPUContinueGlitchedLine0
	PPUEndGlitchedLine0
	PPUOAMInterrupt

	DMAStartTransfer
	DMAEndTransfer
	DMATransfer

	TimerTIMAReload
	TimerTIMAFinishReload
	TimerTIMAIncrement

	SerialBitTransfer
	SerialBitInterrupt

	CameraShoot
)

type Event struct {
	cycle     uint64
	eventType EventType
	next      *Event
	handler   func()
}

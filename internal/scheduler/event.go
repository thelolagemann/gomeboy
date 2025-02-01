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

	PPUHandleVisualLine
	PPUHandleGlitchedLine0
	PPUHandleOffscreenLine

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

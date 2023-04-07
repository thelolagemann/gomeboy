package scheduler

type EventType uint8

const (
	APUFrameSequencer EventType = iota
	APUChannel1
	APUChannel2
	APUChannel3
	APUChannel3WaveRAMWriteCorruption
	APUChannel3WaveRAMWriteCorruptionEnd
	APUChannel4
	APUSample

	PPUHBlank
	PPUHBlankInterrupt
	PPUVBlank
	PPUVBlankInterrupt
	PPUVBlankLast
	PPUOAMSearch
	PPUVRAMTransfer
	PPULYReset
	PPUGlitchedLine0

	DMAStartTransfer
	DMAEndTransfer
	DMATransfer

	TimerInterrupt
	TimerTIMAReload
	TimerTIMAFinishReload
	TimerTIMAIncrement

	EIPending
	HaltDI
	EIHaltDelay
)

const (
	eventTypes = 28
)

type Event struct {
	cycle     uint64
	eventType EventType
	next      *Event
	handler   func()
}

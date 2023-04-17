package scheduler

type EventType uint8

const (
	APUFrameSequencer EventType = iota
	APUChannel1
	APUChannel2
	APUChannel3
	APUChannel3WaveRAMReadCorruption
	APUChannel3WaveRAMWriteCorruption
	APUChannel3WaveRAMWriteCorruptionEnd
	APUChannel4
	APUSample

	PPUHBlank
	PPUHBlankInterrupt
	PPUVBlank
	PPUVBlankInterrupt
	PPUVBlankLast
	PPUStartOAMSearch
	PPUEndFrame
	PPUContinueOAMSearch
	PPUEndOAMSearch
	PPULine153Start
	PPULine153Continue
	PPULine153End
	PPUStartVBlank
	PPUContinueVBlank
	PPUVRAMUnlocked
	PPUVRAMLocked
	PPUVRAMTransfer
	PPUOAMLocked
	PPUOAMUnlocked
	PPULYReset
	PPUGlitchedLine0
	PPUStartGlitchedLine0
	PPUContinueGlitchedLine0
	PPUGlitchedLine0End
	PPUOAMInterrupt

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

	SerialBitTransfer
)

const (
	eventTypes = 45
)

var eventTypeNames = []string{
	"APUFrameSequencer",
	"APUChannel1",
	"APUChannel2",
	"APUChannel3",
	"APUChannel3WaveRAMWriteCorruption",
	"APUChannel3WaveRAMWriteCorruptionEnd",
	"APUChannel4",
	"APUSample",
	"PPUHBlank",
	"PPUHBlankInterrupt",
	"PPUVBlank",
	"PPUVBlankInterrupt",
	"PPUVBlankLast",
	"PPUStartOAMSearch",
	"PPUContinueOAMSearch",
	"PPUEndOAMSearch",
	"PPULine153Start",
	"PPULine153Continue",
	"PPULine153End",
	"PPUStartVBlank",
	"PPUContinueVBlank",
	"PPUVRAMTransfer",
	"PPULYReset",
	"PPUGlitchedLine0",
	"PPUStartGlitchedLine0",
	"PPUContinueGlitchedLine0",
	"PPUGlitchedLine0End",
	"PPUOAMInterrupt",
	"DMAStartTransfer",
	"DMAEndTransfer",
	"DMATransfer",
	"TimerInterrupt",
	"TimerTIMAReload",
	"TimerTIMAFinishReload",
	"TimerTIMAIncrement",
	"EIPending",
	"HaltDI",
	"EIHaltDelay",
	"SerialBitTransfer",
}

type Event struct {
	cycle     uint64
	eventType EventType
	next      *Event
	handler   func()
}

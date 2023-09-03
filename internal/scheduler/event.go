package scheduler

type EventType uint

func (e EventType) String() string {
	return eventTypeNames[e]
}

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

	EIPending
	HaltDI
	EIHaltDelay

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
	PPUVRAMReadLocked
	PPUVRAMReadUnlocked
	PPUVRAMWriteLocked
	PPUVRAMWriteUnlocked
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

	SerialBitTransfer
	SerialBitInterrupt
)

const (
	eventTypes = 48
)

var eventTypeNames = []string{
	"APUFrameSequencer",
	"APUChannel1",
	"APUChannel2",
	"APUChannel3",
	"APUChannel3WaveRAMReadCorruption",
	"APUChannel3WaveRAMWriteCorruption",
	"APUChannel3WaveRAMWriteCorruptionEnd",
	"APUChannel4",
	"APUSample",

	"EIPending",
	"HaltDI",
	"EIHaltDelay",

	"PPUHBlank",
	"PPUHBlankInterrupt",
	"PPUVBlank",
	"PPUVBlankInterrupt",
	"PPUVBlankLast",
	"PPUStartOAMSearch",
	"PPUEndFrame",
	"PPUContinueOAMSearch",
	"PPUEndOAMSearch",
	"PPULine153Start",
	"PPULine153Continue",
	"PPULine153End",
	"PPUStartVBlank",
	"PPUContinueVBlank",
	"PPUVRAMReadLocked",
	"PPUVRAMReadUnlocked",
	"PPUVRAMWriteLocked",
	"PPUVRAMWriteUnlocked",
	"PPUVRAMTransfer",
	"PPUOAMLocked",
	"PPUOAMUnlocked",
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

	"SerialBitTransfer",
	"SerialBitInterrupt",
}

type Event struct {
	cycle     uint64
	eventType EventType
	next      *Event
	handler   func()
}

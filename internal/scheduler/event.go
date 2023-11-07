package scheduler

type EventType uint8

func (e EventType) String() string {
	return eventTypeNames[e]
}

const (
	APUFrameSequencer EventType = iota
	APUChannel1
	APUChannel2
	APUChannel3
	APUSample

	EIPending
	EIHaltDelay

	PPUStartHBlank
	PPUHBlank
	PPUHBlankInterrupt
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
	HDMA

	SerialBitTransfer
	SerialBitInterrupt

	JoypadA
	JoypadB
	JoypadSelect
	JoypadStart
	JoypadRight
	JoypadLeft
	JoypadUp
	JoypadDown

	JoypadARelease
	JoypadBRelease
	JoypadSelectRelease
	JoypadStartRelease
	JoypadRightRelease
	JoypadLeftRelease
	JoypadUpRelease
	JoypadDownRelease
)

const (
	eventTypes = 50
)

var eventTypeNames = []string{
	"APUFrameSequencer",
	"APUChannel1",
	"APUChannel2",
	"APUChannel3",
	"APUSample",

	"EIPending",
	"EIHaltDelay",

	"PPUStartHBlank",
	"PPUHBlank",
	"PPUHBlankInterrupt",
	"PPUStartOAMSearch",
	"PPUEndFrame",
	"PPUContinueOAMSearch",
	"PPUPrepareEndOAMSearch",
	"PPUEndOAMSearch",
	"PPULine153Continue",
	"PPULine153End",
	"PPUStartVBlank",
	"PPUContinueVBlank",
	"PPUVRAMTransfer",
	"PPUStartGlitchedLine0",
	"PPUMiddleGlitchedLine0",
	"PPUContinueGlitchedLine0",
	"PPUEndGlitchedLine0",
	"PPUOAMInterrupt",

	"DMAStartTransfer",
	"DMAEndTransfer",
	"DMATransfer",

	"TimerTIMAReload",
	"TimerTIMAFinishReload",
	"TimerTIMAIncrement",
	"HDMA",

	"SerialBitTransfer",
	"SerialBitInterrupt",

	"JoypadA",
	"JoypadB",
	"JoypadSelect",
	"JoypadStart",
	"JoypadRight",
	"JoypadLeft",
	"JoypadUp",
	"JoypadDown",

	"JoypadARelease",
	"JoypadBRelease",
	"JoypadSelectRelease",
	"JoypadStartRelease",
	"JoypadRightRelease",
	"JoypadLeftRelease",
	"JoypadUpRelease",
	"JoypadDownRelease",
}

type Event struct {
	cycle     uint64
	eventType EventType
	next      *Event
	handler   func()
}

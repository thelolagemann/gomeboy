package display

// EventType defines the various event types
// that can be sent to a view. The event type
// indicates to the view what action should be
// taken.
type EventType int

const (
	// EventTypeQuit is the event type for when the user quits the application
	EventTypeQuit EventType = iota
	EventTypeSample
	// EventTypeFrame is the event type for when a frame should be drawn
	EventTypeFrame
	EventTypeFrameTime
	// EventTypeTitle is the event type for when the title of the window should be changed
	EventTypeTitle
	// EventTypePrint is the event type for when a print job should be performed
	EventTypePrint
)

type Event struct {
	// Type is the type of event
	Type EventType
	// State is the state of the event
	State struct {
		// CPU is the state of the CPU
		CPU CPUState
	}
	// Data is the data of the event
	Data interface{}
}

type CPUState struct {
	Registers struct {
		// AF is the state of the AF register
		AF uint16
		// BC is the state of the BC register
		BC uint16
		// DE is the state of the DE register
		DE uint16
		// HL is the state of the HL register
		HL uint16
		// SP is the state of the SP register
		SP uint16
		// PC is the state of the PC register
		PC uint16
	}
}

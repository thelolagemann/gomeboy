// Package event defines the various event types that can
// be sent to a display.Driver. This package is separate from
// the display package to avoid circular dependencies.
package event

// Type defines the various event types
// that can be sent to a display.Driver. The event type
// indicates to the display.Driver what action should be
// taken.
type Type int

const (
	// Quit is sent when the user requests that the
	// application be closed.
	Quit Type = iota
	// Sample is periodically sent to the display.Driver
	// to indicate that the display.Driver should update
	// its audio visualiser view (if any).
	Sample
	// FrameTime is periodically sent to the display.Driver
	// to indicate the average time between frames.
	FrameTime
	// Title is sent to the display.Driver to change the
	// title of the window. This can be used to display
	// custom information in the title bar, such as the
	// current game, or FPS.
	Title
	// Print is sent when the accessories.Printer
	// receives a print job from the Game Boy, indicating to the
	// display.Driver that it should update the printer display.
	Print
)

// Event is the data structure that is sent to the display.Driver
// to indicate an event has occurred. Data may or may not
// contain any data, depending on the event type.
type Event struct {
	// Type is the type of event
	Type Type
	// Data is the data of the event
	Data interface{}
}

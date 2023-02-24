package display

// View defines the interface contract for a view.
type View interface {
	// Run runs the view and blocks until the view is closed,
	// or an error occurs. The event channel is used to send events
	// to the view.
	Run(window Window) error
}

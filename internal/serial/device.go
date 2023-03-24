package serial

// Device is a device that can be attached to the Controller.
type Device interface {
	Receive(bool)
	Send() bool
}

// nullDevice is an implementation of Device that
// simply returns true on Send and does nothing on
// Receive. This is most commonly used for when no
// device is attached to the Controller.
type nullDevice struct{}

// Receive does nothing.
func (n nullDevice) Receive(bool) {}

// Send always returns true.
func (n nullDevice) Send() bool { return true }

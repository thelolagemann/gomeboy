package display

import (
	"github.com/thelolagemann/gomeboy/internal/joypad"
)

// Driver is the interface that wraps the basic methods for a
// display driver.
type Driver interface {
	// Start the display driver.
	Start(fb <-chan []byte, events <-chan Event, pressed, released chan<- joypad.Button) error
	// Stop the display driver.
	Stop() error
}

// InstalledDriver is a driver that has been installed. This is
// used to allow drivers to register their name.
type InstalledDriver struct {
	Name string
	Driver
}

// InstalledDrivers is a list of all the installed drivers. This
// variable is exported so that it can be used by the main
// program to determine which drivers can be used. Drivers should
// call display.Install in their init() function.
var InstalledDrivers []*InstalledDriver

// GetDriver returns the driver with the given name, or nil if
// no driver with that name is installed.
func GetDriver(name string) Driver {
	if name == "auto" {
		return InstalledDrivers[0]
	}
	for _, driver := range InstalledDrivers {
		if driver.Name == name {
			return driver.Driver
		}
	}

	return nil
}

// Install registers a display driver with the given name.
func Install(name string, driver Driver) {
	if InstalledDrivers == nil {
		InstalledDrivers = make([]*InstalledDriver, 0)
	}

	InstalledDrivers = append(InstalledDrivers, &InstalledDriver{
		Name:   name,
		Driver: driver,
	})
}

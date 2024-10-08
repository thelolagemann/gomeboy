package display

import (
	"flag"
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/pkg/emulator"
	"github.com/thelolagemann/gomeboy/pkg/log"
	"strconv"
)

// Init initializes the display drivers and ensures
// at least one has been installed.
func Init() {
	// make sure at least 1 driver is installed
	if len(InstalledDrivers) == 0 {
		log.Fatal("No display drivers installed. Please compile with at least one display driver")
	}

	// register flags for each driver
	RegisterFlags()
}

// Driver is the interface that wraps the basic methods for a
// display driver.
type Driver interface {
	// Start the display driver.
	Start(c emulator.Controller, fb <-chan []byte, pressed, released chan<- io.Button) error
	// Stop the display driver.
	Stop() error
}

type DriverDebugger interface {
	AttachGameboy(*gameboy.GameBoy) // find a better way to do this
}

// DriverOption is a display driver option. This is used to
// configure a display driver.
type DriverOption struct {
	Name        string // name of the option
	Default     any    // default value of the option
	Value       any    // pointer to the value of the option
	Description string // description of the option
	Type        string // "int", "bool", "string", "float"
}

// InstalledDriver is a driver that has been installed. This is
// used to allow drivers to register their name.
type InstalledDriver struct {
	Name    string
	Options []DriverOption
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
		return InstalledDrivers[0].Driver
	}
	for _, driver := range InstalledDrivers {
		if driver.Name == name {
			return driver.Driver
		}
	}

	return nil
}

// Install registers a display driver with the given name.
func Install(name string, driver Driver, options []DriverOption) {
	if InstalledDrivers == nil {
		InstalledDrivers = make([]*InstalledDriver, 0)
	}

	InstalledDrivers = append(InstalledDrivers, &InstalledDriver{
		Name:    name,
		Options: options,
		Driver:  driver,
	})
}

// RegisterFlags iterates through all the display driver
// options and registers them with the flag package.
func RegisterFlags() {
	optionCounts := make(map[string]int)
	opts := make(map[string][]DriverOption)
	prefixes := make(map[DriverOption]string)

	for _, driver := range InstalledDrivers {
		for _, opt := range driver.Options {
			// track how many times an option is used
			optionCounts[opt.Name]++
			opts[opt.Name] = append(opts[opt.Name], opt)
			prefixes[opt] = driver.Name
		}
	}

	for o, count := range optionCounts {
		// this requires an option merge
		if count > 1 {
			// grab the first option
			opt := opts[o][0]
			switch opt.Type {
			case "string":
				multi := &multiValue{values: make([]any, 0)}
				for _, mOpt := range opts[o] {
					multi.values = append(multi.values, opt.Value.(*string))
					*mOpt.Value.(*string) = multi.defaultValue.(string)
				}
				flag.Var(multi, o, opt.Description)
			case "bool":
				multi := &multiValue{make([]any, 0), opt.Default.(bool)}
				for _, mOpt := range opts[o] {
					multi.values = append(multi.values, mOpt.Value.(*bool))
					*mOpt.Value.(*bool) = multi.defaultValue.(bool)
				}
				flag.Var(multi, o, opt.Description)
			case "float":
				multi := &multiValue{make([]any, 0), opt.Default.(float64)}
				for _, mOpt := range opts[o] {
					multi.values = append(multi.values, mOpt.Value.(*float64))
					*mOpt.Value.(*float64) = multi.defaultValue.(float64)
				}
				flag.Var(multi, o, opt.Description)
			}
		} else {
			// this option is unique and should be prefixed
			opt := opts[o][0]
			optName := fmt.Sprintf("%s-%s", prefixes[opt], opt.Name)
			switch opt.Type {
			case "string":
				flag.StringVar(opt.Value.(*string), optName, opt.Default.(string), opt.Description)
			case "bool":
				flag.BoolVar(opt.Value.(*bool), optName, opt.Default.(bool), opt.Description)
			case "float":
				flag.Float64Var(opt.Value.(*float64), optName, opt.Default.(float64), opt.Description)
			}
		}
	}
}

type multiValue struct {
	values       []any
	defaultValue any
}

func (m *multiValue) String() string {
	switch m.defaultValue.(type) {
	case string:
		return m.defaultValue.(string)
	case bool:
		return fmt.Sprintf("%t", m.defaultValue.(bool))
	case float64:
		return fmt.Sprintf("%f", m.defaultValue.(float64))
	default:
		return ""
	}
}

func (m *multiValue) Set(value string) error {
	// update all the pointers with the provided value
	for _, ptr := range m.values {
		switch ptr.(type) {
		case *string:
			*ptr.(*string) = value
		case *bool:
			*ptr.(*bool) = true
		case *float64:
			f, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return err
			}
			*ptr.(*float64) = f
		default:
			return fmt.Errorf("unknown type: %T", ptr) // should never happen, but just in case...
		}
	}

	return nil
}

func (m *multiValue) IsBoolFlag() bool {
	_, isBool := m.defaultValue.(bool)
	return isBool
}

package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/go-gameboy/internal/cartridge"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"strconv"
	"strings"
)

var (
	_ display.View = (*MMU)(nil)
)

// MMU is a view that displays the memory map of the gameboy
type MMU struct {
	*mmu.MMU
}

func (M *MMU) Title() string {
	return "MMU"
}

func (M *MMU) Run(w fyne.Window, events <-chan display.Event) error {
	// create the base grid
	grid := container.NewVBox()

	// TODO change to textgrid

	// set the content of the window
	w.SetContent(grid)

	// boot rom information
	bootRomGrid := container.NewVBox()
	grid.Add(bootRomGrid)

	// boot rom information
	bootRomGrid.Add(widget.NewLabel("Boot ROM Information"))
	bootRomGrid.Add(widget.NewTextGridFromString(`Enabled		` + strconv.FormatBool(M.BootROM != nil) + `
Model		` + M.BootROM.Model() + `
Checksum	` + strings.ToUpper(M.BootROM.Checksum())))

	// cartridge informaton
	cartridgeGrid := container.NewVBox()
	grid.Add(cartridgeGrid)

	cartridgeGrid.Add(widget.NewLabel("Cartridge Information"))
	// cartridge information
	// cartridgeGrid.Add(widget.NewLabel("Cartridge Information"))
	cartridgeGrid.Add(widget.NewTextGridFromString(`Title		` + M.Cart.Header().Title + `
Type		` + M.Cart.Header().CartridgeType.String() + `
ROM Size	` + humanReadable(M.Cart.Header().ROMSize) + `
RAM Size	` + humanReadable(M.Cart.Header().RAMSize) + `
Licensee	` + M.Cart.Header().Licensee() + `
Checksum	` + fmt.Sprintf("0x%02x", M.Cart.Header().HeaderChecksum) + `
Destination	` + M.Cart.Header().Destination() + `
`))

	// feature grid
	featureGrid := container.NewGridWithRows(2)
	cartridgeGrid.Add(featureGrid)

	// checkbox grid
	checkboxGrid := container.NewGridWithColumns(3)

	// 1 label, 3 checkboxes for the SGB and CGB flags
	featureGrid.Add(widget.NewLabel("Features:"))
	sgbCheckbox := widget.NewCheck("SGB", func(b bool) {})
	sgbCheckbox.SetChecked(M.Cart.Header().SGBFlag)
	cgbCheckbox := widget.NewCheck("Requires CGB", func(b bool) {})
	cgbCheckbox.SetChecked(M.Cart.Header().CartridgeGBMode == cartridge.FlagOnlyCGB)
	cgbSupportCheckbox := widget.NewCheck("Supports CGB", func(b bool) {})
	cgbSupportCheckbox.SetChecked(M.Cart.Header().CartridgeGBMode == cartridge.FlagSupportsCGB)

	// disable the checkboxes as they cannot be changed and are only for information
	//sgbCheckbox.Disable()
	//cgbCheckbox.Disable()
	//cgbSupportCheckbox.Disable()

	// add the checkboxes to the grid
	checkboxGrid.Add(sgbCheckbox)
	checkboxGrid.Add(cgbCheckbox)
	checkboxGrid.Add(cgbSupportCheckbox)

	// add the checkbox grid to the feature grid
	featureGrid.Add(checkboxGrid)

	go func() {
		for {
			<-events // MMU view does not react to events (yet)
		}
	}()

	return nil
}

// NewMMU creates a new MMU view
func NewMMU(m *mmu.MMU) *MMU {
	return &MMU{MMU: m}
}

// humanReadable returns a human readable string in bytes for the given size
func humanReadable(s uint) string {
	if s < 1024 {
		return strconv.Itoa(int(s)) + " B"
	}
	if s < 1024*1024 {
		return strconv.Itoa(int(s)/1024) + " KiB"
	}
	return strconv.Itoa(int(s)/(1024*1024)) + " MiB"
}

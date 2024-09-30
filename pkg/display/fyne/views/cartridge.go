package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/io"
	"strconv"
)

type Cartridge struct {
	widget.BaseWidget
	*io.Cartridge
}

func NewCartridge(cart *io.Cartridge) *Cartridge {
	c := &Cartridge{Cartridge: cart}
	c.ExtendBaseWidget(c)
	return c
}

func (c *Cartridge) CreateRenderer() fyne.WidgetRenderer {
	// create cartridge info container
	cartridgeInfo := container.NewVBox(widget.NewLabelWithStyle("Cartridge Information", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

	// create textgrid for cartridge info
	cartridgeInfoGrid := widget.NewTextGrid()
	cartridgeInfoGrid.SetText(`Title				` + c.Cartridge.Title + `
Manufacturer		` + c.ManufacturerCode + `
CGB Support			` + strconv.FormatBool(c.IsCGBCartridge()) + `
SGB Support			` + strconv.FormatBool(c.SGBFlag) + `
Cartridge Type		` + c.CartridgeType.String() + `
ROM Size			` + humanReadable(uint(c.ROMSize)) + `
RAM Size			` + humanReadable(uint(c.RAMSize)) + `
Destination			` + c.Destination() + `
Licensee			` + c.Licensee() + `
ROM Version			` + strconv.Itoa(int(c.MaskROMVersion)) + `
Header Checksum		` + fmt.Sprintf("0x%02X", c.HeaderChecksum) + `
Global Checksum		` + fmt.Sprintf("0x%04X", c.GlobalChecksum))
	cartridgeInfo.Add(cartridgeInfoGrid)

	// create features checklist
	type feature struct {
		name  string
		value bool
	}
	features := []feature{{"Accelerometer", c.Features.Accelerometer}, {"Battery", c.Features.Battery}, {"RAM", c.Features.RAM}, {"RTC", c.Features.RTC}, {"Rumble", c.Features.Rumble}}
	checkList := container.NewVBox(widget.NewLabelWithStyle("Features", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	for _, k := range features {
		checkList.Add(newStaticCheckbox(k.name, k.value))
	}
	cartridgeInfo.Add(checkList)

	return widget.NewSimpleRenderer(cartridgeInfo)
}

// humanReadable returns a human-readable string in bytes for the given size
func humanReadable(s uint) string {
	if s < 1024 {
		return strconv.Itoa(int(s)) + " B"
	}
	if s < 1024*1024 {
		return strconv.Itoa(int(s)/1024) + " KiB"
	}
	return strconv.Itoa(int(s)/(1024*1024)) + " MiB"
}

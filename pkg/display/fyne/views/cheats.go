package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/pkg/display/fyne/themes"
	"image/color"
	"strings"
)

type Cheats struct {
	widget.BaseWidget

	bus *io.Bus
}

func NewCheats(b *io.Bus) *Cheats {
	c := &Cheats{bus: b}
	c.ExtendBaseWidget(c)
	return c
}

func (c *Cheats) CreateRenderer() fyne.WidgetRenderer {
	actions := container.NewVBox()
	cheatName := widget.NewEntry()
	cheatName.PlaceHolder = "Name"

	cheatCode := widget.NewEntry()
	cheatCode.PlaceHolder = "Code"
	cheatCode.TextStyle.Monospace = true
	cheatCode.MultiLine = true
	cheatCode.SetMinRowsVisible(6)

	warningBox := newInfoBox(themeColor(theme.ColorNameError))

	actions.Add(cheatName)
	actions.Add(cheatCode)

	cheatList := widget.NewList(
		func() int {
			return len(c.bus.LoadedCheats)
		},
		func() fyne.CanvasObject { return newCheatListItem(findWindow("Cheats")) },
		nil)
	cheatList.UpdateItem = func(id widget.ListItemID, object fyne.CanvasObject) {
		cont := object.(*cheatListItem)
		cont.cheat = &c.bus.LoadedCheats[id]
		cont.refresher = func() { cheatList.RefreshItem(id) }
		cont.deleter = func() {
			if err := c.bus.UnloadCheat(id, true); err != nil {
				fmt.Println("eee")
			}
			cheatList.Refresh()
		}
		cont.editor = func() {
			cheatName.Text = c.bus.LoadedCheats[id].Name
			cheatCode.Text = strings.Join(c.bus.LoadedCheats[id].Codes, "\n")

			cheatName.Refresh()
			cheatCode.Refresh()
		}
		cont.cheatName.Text = c.bus.LoadedCheats[id].Name
		cont.cheatName.Refresh()
		cont.subheading.Text = fmt.Sprintf("%d codes", len(c.bus.LoadedCheats[id].Codes))
		cont.subheading.Refresh()

		if !c.bus.LoadedCheats[id].Enabled {
			c.bus.UnloadCheat(id, false)
			cont.background.FillColor = themeColor(theme.ColorNameHover)
			cont.enabledCheck.SetChecked(false) // to handle on initial load
		} else {
			cont.background.FillColor = themeColor(themes.ColorNameBackgroundOnBackground)
			c.bus.LoadCheat(c.bus.LoadedCheats[id], false)
		}
	}
	addCheat := widget.NewButton("Add", func() {
		warningBox.setText("")
		if cheatName.Text == "" {
			warningBox.setText("Please enter a name")
			return
		}
		ch := io.Cheat{Name: cheatName.Text, Enabled: true}
		warningComment := ""
		for _, code := range strings.Split(cheatCode.Text, "\n") {
			code = strings.Trim(strings.TrimSpace(code), "-")
			if code != "" { // skip empty lines
				// try to load the cheat
				if _, err := io.ParseGameGenieCode(code); err == nil { // valid game genie code
					ch.Codes = append(ch.Codes, code)
				} else if _, err := io.ParseGameSharkCode(code); err == nil { // valid game shark code
					ch.Codes = append(ch.Codes, code)
				} else {
					warningComment += fmt.Sprintf("invalid code: %s\n", code)
				}

			}
		}
		if warningComment != "" {
			warningBox.setText("Error parsing codes:\n" + warningComment[:len(warningComment)-1])
		}
		cheatName.SetText("")
		cheatCode.SetText("")
		c.bus.LoadCheat(ch, true)
		cheatList.Refresh()
	})
	addCheat.Resize(fyne.NewSize(400, addCheat.MinSize().Height))
	actions.Add(addCheat)
	actions.Add(warningBox)

	spaceTaker := canvas.NewText("                                                                        ", themeColor(theme.ColorNameForeground))
	actions.Add(spaceTaker) // force width

	return widget.NewSimpleRenderer(container.NewBorder(nil, nil, actions, nil, cheatList))
}

type cheatListItem struct {
	widget.BaseWidget

	cheat                      *io.Cheat
	deleter, editor, refresher func()
	cheatName, subheading      *canvas.Text
	enabledCheck               *widget.Check
	background                 *canvas.Rectangle

	w fyne.Window
}

func newCheatListItem(w fyne.Window) *cheatListItem {
	c := &cheatListItem{w: w}
	c.ExtendBaseWidget(c)
	return c
}

func (c *cheatListItem) CreateRenderer() fyne.WidgetRenderer {
	c.cheatName = canvas.NewText("", themeColor(theme.ColorNameForeground))
	c.cheatName.SetMinSize(fyne.NewSize(800, c.cheatName.MinSize().Height))
	c.subheading = canvas.NewText("", themeColor(theme.ColorNameForeground))
	c.subheading.TextSize = 12

	textBox := container.NewVBox(c.cheatName, c.subheading)

	actionContainer := container.NewHBox()
	editButton := widget.NewButton("", func() {
		c.editor()
	})
	editButton.SetIcon(theme.DocumentCreateIcon())
	deleteButton := widget.NewButton("", func() {
		d := dialog.NewConfirm(fmt.Sprintf("Delete %s?", c.cheat.Name), fmt.Sprintf("Are you sure you want to delete %s and all of it's codes?", c.cheat.Name), func(b bool) {
			c.deleter()
		}, c.w)
		d.SetConfirmImportance(widget.WarningImportance)
		d.Show()
	})
	deleteButton.SetIcon(theme.DeleteIcon())
	actionContainer.Add(editButton)
	actionContainer.Add(deleteButton)
	c.background = canvas.NewRectangle(themeColor(theme.ColorNameBackground))
	c.background.CornerRadius = 5
	c.background.SetMinSize(fyne.NewSize(300, c.background.MinSize().Height))
	c.enabledCheck = widget.NewCheck("", func(b bool) {
		c.cheat.Enabled = b
		c.refresher()
	})
	c.enabledCheck.Checked = true

	content := container.NewStack(c.background, container.NewPadded(container.NewBorder(nil, nil, c.enabledCheck, actionContainer, textBox)))
	return widget.NewSimpleRenderer(content)
}

type infoBox struct {
	widget.BaseWidget

	text            *widget.Label
	backgroundColor color.Color
	content         *fyne.Container
}

func newInfoBox(bg color.Color) *infoBox {
	w := &infoBox{backgroundColor: bg}
	w.ExtendBaseWidget(w)
	return w
}

func (w *infoBox) CreateRenderer() fyne.WidgetRenderer {
	w.text = widget.NewLabel("")
	w.text.Wrapping = fyne.TextWrapWord
	background := canvas.NewRectangle(w.backgroundColor)

	background.CornerRadius = 5
	bIcon := widget.NewIcon(theme.ErrorIcon())

	w.content = container.NewStack(background, container.NewPadded(container.NewBorder(nil, nil, bIcon, nil, w.text)))
	w.content.Hide()

	return widget.NewSimpleRenderer(w.content)
}

func (w *infoBox) setText(text string) {
	if text == "" {
		w.content.Hide()
	} else {
		w.text.SetText(text)
		w.content.Show()
	}
}

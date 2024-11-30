package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/pkg/display/fyne/themes"
	"image"
	"image/color"
	"image/png"
)

// bold is a small utility function for creating a bold label.
func bold(s string) *widget.Label { return widget.NewLabelWithStyle(s, 0, fyne.TextStyle{Bold: true}) }

// mono is a small utility function for creating a monospaced text element.
func mono(s string, c color.Color) *canvas.Text {
	t := canvas.NewText(s, c)
	t.TextStyle.Monospace = true

	return t
}

// newBadge wraps content in a stack, with optional rounded edges.
func newBadge(backgroundColor color.Color, cornerRadius float32, content fyne.CanvasObject) fyne.CanvasObject {
	bgRect := canvas.NewRectangle(backgroundColor)
	bgRect.CornerRadius = cornerRadius
	bgRect.Resize(content.MinSize())

	return container.NewStack(bgRect, content)
}

func newCard(title string, content fyne.CanvasObject) fyne.CanvasObject {
	return newBadge(themeColor(themes.ColorNameBackgroundOnBackground), 5, container.NewVBox(
		newBadge(themeColor(theme.ColorNameInputBackground), 5, container.NewPadded(mono(title, themeColor(theme.ColorNameForeground)))),
		content))
}

type staticCheckbox struct {
	widget.Check
}

func newStaticCheckbox(label string, checked bool) *staticCheckbox {
	cb := &staticCheckbox{}
	cb.Text = label
	cb.Checked = checked
	cb.ExtendBaseWidget(cb)
	return cb
}

func (c *staticCheckbox) CreateRenderer() fyne.WidgetRenderer { return c.Check.CreateRenderer() }
func (c *staticCheckbox) FocusGained()                        {}
func (c *staticCheckbox) MouseIn(_ *desktop.MouseEvent)       {}
func (c *staticCheckbox) MouseOut()                           {}
func (c *staticCheckbox) MouseMoved(_ *desktop.MouseEvent)    {}
func (c *staticCheckbox) Tapped(_ *fyne.PointEvent)           {}

type tappable struct {
	obj fyne.CanvasObject
	widget.BaseWidget
	tapHandler func()
}

func newWrappedTappable(onTap func(), obj fyne.CanvasObject) *tappable {
	if onTap == nil {
		onTap = func() {}
	}
	w := &tappable{obj: obj, tapHandler: onTap}
	w.ExtendBaseWidget(w)
	return w
}

func (t *tappable) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(t.obj) }
func (t *tappable) Cursor() desktop.Cursor              { return desktop.PointerCursor }
func (t *tappable) Tapped(*fyne.PointEvent)             { t.tapHandler() }
func (t *tappable) TappedSecondary(*fyne.PointEvent)    {}

func themeColor(name fyne.ThemeColorName) color.Color {
	return theme.Current().Color(name, fyne.CurrentApp().Settings().ThemeVariant())
}

func findWindow(name string) fyne.Window {
	for _, w := range fyne.CurrentApp().Driver().AllWindows() {
		if w.Title() == name {
			return w
		}
	}

	return nil
}

func saveImage(img image.Image, filename, name string) {
	d := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if writer == nil {
			return // user cancelled
		}
		if err := png.Encode(writer, img); err != nil {
			showError(err, name)
			return
		}
	}, findWindow(name))
	d.SetFileName(filename)
	d.Show()
}

func showError(err error, name string) {
	if err != nil {
		d := dialog.NewError(err, findWindow(name))
		d.Show()
	}
}

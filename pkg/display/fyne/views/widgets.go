package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"image"
)

type MenuOption func(*fyne.MenuItem)

func Checked(b bool, refresher func()) MenuOption {
	return func(item *fyne.MenuItem) {
		tempFn := item.Action
		item.Action = func() {
			tempFn()
			item.Checked = !item.Checked
			refresher()
		}
		item.Checked = b
	}
}

func Gated(b bool) MenuOption {
	return func(item *fyne.MenuItem) {
		item.Disabled = b
	}
}

func NewCustomizedMenuItem(label string, fn func(), opts ...MenuOption) *fyne.MenuItem {
	m := fyne.NewMenuItem(label, fn)
	for _, o := range opts {
		o(m)
	}
	return m
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

func (c *staticCheckbox) FocusGained()                     {}
func (c *staticCheckbox) MouseIn(_ *desktop.MouseEvent)    {}
func (c *staticCheckbox) MouseOut()                        {}
func (c *staticCheckbox) MouseMoved(_ *desktop.MouseEvent) {}
func (c *staticCheckbox) Tapped(_ *fyne.PointEvent)        {}

func (c *staticCheckbox) CreateRenderer() fyne.WidgetRenderer {
	return c.Check.CreateRenderer()
}

type customPaddedButton struct {
	widget.BaseWidget

	Button *widget.Button
}

func (c *customPaddedButton) MinSize() fyne.Size {
	return fyne.NewSize(26, 26)
}

func (c *customPaddedButton) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.Button)
}

func newCustomPaddedButton(label string, tapped func()) *customPaddedButton {
	c := &customPaddedButton{}
	c.ExtendBaseWidget(c)
	c.Button = widget.NewButton(label, tapped)
	return c
}

type tappableImage struct {
	widget.BaseWidget
	c          *canvas.Raster
	img        *image.RGBA
	tapHandler func(event *fyne.PointEvent)
}

func newTappableImage(img *image.RGBA, c *canvas.Raster, tapHandler func(event *fyne.PointEvent)) *tappableImage {
	t := &tappableImage{img: img, tapHandler: tapHandler, c: c}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tappableImage) Cursor() desktop.Cursor           { return desktop.PointerCursor }
func (t *tappableImage) Tapped(at *fyne.PointEvent)       { t.tapHandler(at) }
func (t *tappableImage) TappedSecondary(*fyne.PointEvent) {}

func (t *tappableImage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.c)
}

type wrappedTappable struct {
	obj fyne.CanvasObject
	widget.BaseWidget
	tapHandler func()
}

func newWrappedTappable(onTap func(), obj fyne.CanvasObject) *wrappedTappable {
	if onTap == nil {
		onTap = func() {}
	}
	w := &wrappedTappable{obj: obj, tapHandler: onTap}
	w.ExtendBaseWidget(w)
	return w
}

func (w *wrappedTappable) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(w.obj)
}

func (w *wrappedTappable) Cursor() desktop.Cursor           { return desktop.PointerCursor }
func (w *wrappedTappable) Tapped(*fyne.PointEvent)          { w.tapHandler() }
func (w *wrappedTappable) TappedSecondary(*fyne.PointEvent) {}

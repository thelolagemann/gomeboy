package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"image"
	"image/png"
)

// Windowed i
type Windowed interface {
	AttachWindow(fyne.Window)
}

// WindowedView is a helper struct to allow widgets to be embedded
// with a Window.
type WindowedView struct {
	Window fyne.Window
}

// error displays err with a dialog on the embedded Window. If err
// is nil, nothing happens.
func (w *WindowedView) error(err error) {
	if err != nil {
		d := dialog.NewError(err, w.Window)
		d.Show()
	}
}

func (w *WindowedView) saveImage(img image.Image, name string) {
	d := dialog.NewFileSave(func(closer fyne.URIWriteCloser, err error) {
		if closer == nil {
			return // user cancelled
		}
		if err := png.Encode(closer, img); err != nil {
			w.error(err)
			return
		}
	}, w.Window)
	d.SetFileName(name)
	d.Show()
}

func (w *WindowedView) saveFile(b []byte) {
	d := dialog.NewFileSave(func(closer fyne.URIWriteCloser, err error) {
		if closer == nil {
			return // user cancelled
		}
		if _, err := closer.Write(b); err != nil {
			w.error(err)
			return
		}
	}, w.Window)
	d.Show()
}

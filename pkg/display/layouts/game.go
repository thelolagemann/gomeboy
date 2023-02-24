package layouts

import "fyne.io/fyne/v2"

type Game struct{}

func NewGame() *Game {
	return &Game{}
}

func (g *Game) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	pos := fyne.NewPos(0, 0)
	for _, o := range objects {
		// resize to take up the whole space
		w, h := size.Width, size.Height
		o.Resize(fyne.NewSize(w, h))
		o.Move(pos)
	}
}

func (g *Game) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(160, 144)
}

//go:build !test

package utils

import "github.com/sqweek/dialog"

func AskForFile(title, startingDir string) (string, error) {
	builder := dialog.File().SetStartDir(startingDir).Title(title)

	// show the dialog
	return builder.Load()
}

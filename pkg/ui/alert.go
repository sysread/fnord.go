package ui

import (
	"github.com/rivo/tview"
)

func (ui *UI) alert(message string, callback func()) tview.Primitive {
	modal := tview.NewModal()

	modal.SetText(message)

	modal.AddButtons([]string{"OK"})

	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if callback != nil {
			callback()
		}
	})

	return modal
}

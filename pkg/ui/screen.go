package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type keyBinding struct {
	key   string
	label string
}

type screenArgs struct {
	title string
	keys  []keyBinding
}

func (ui *UI) newScreen(widget tview.Primitive, args screenArgs) *tview.Frame {
	keyLabels := ""
	for i, key := range args.keys {
		label := "[blue]" + key.key + "[-] " + key.label

		if i > 0 {
			keyLabels += " | "
		}

		keyLabels += label
	}

	innerFlex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false). // Left padding
		AddItem(widget, 0, 50, true).
		AddItem(nil, 0, 1, false) // Right padding

	outerFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false). // Top padding
		AddItem(innerFlex, 0, 50, true).
		AddItem(nil, 0, 1, false) // Bottom padding

	frame := tview.NewFrame(outerFlex)
	frame.AddText(args.title, true, tview.AlignCenter, tcell.ColorWhite)
	frame.AddText(keyLabels, false, tview.AlignCenter, tcell.ColorWhite)
	frame.SetBorders(1, 1, 0, 0, 1, 1)
	frame.SetBorder(true)

	return frame
}

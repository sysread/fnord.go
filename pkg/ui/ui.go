package ui

import (
	"github.com/rivo/tview"
)

type UI struct {
	app *tview.Application
}

func New() *UI {
	app := tview.NewApplication()
	app.EnableMouse(true)

	ui := &UI{
		app: app,
	}

	return ui
}

func (ui *UI) Run() {
	ui.OpenHome()

	if err := ui.app.Run(); err != nil {
		panic(err)
	}
}

func (ui *UI) Quit() {
	ui.app.Stop()
}

func (ui *UI) Open(view tview.Primitive, fullScreen bool) {
	ui.app.SetRoot(view, fullScreen).SetFocus(view)
}

func (ui *UI) OpenHome() {
	home := ui.newHomeView()
	ui.Open(home, true)
}

func (ui *UI) OpenHelp() {
	helpView := ui.newHelpView()
	ui.Open(helpView, true)
}

func (ui *UI) OpenChat() {
	chatView := ui.newChatView()
	ui.Open(chatView, true)
}

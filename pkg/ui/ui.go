package ui

import (
	"github.com/rivo/tview"
)

type UI struct {
	app   *tview.Application
	pages *tview.Pages

	// Pages
	home       tview.Primitive
	help       tview.Primitive
	chat       *chatView
	filePicker *filePicker
}

func New() *UI {
	app := tview.NewApplication()
	app.EnableMouse(true)

	ui := &UI{
		app:   app,
		pages: tview.NewPages(),
	}

	ui.home = ui.newHomeView()
	ui.help = ui.newHelpView()
	ui.chat = ui.newChatView()
	ui.filePicker = ui.newFilePicker()

	ui.pages.AddPage("home", ui.home, true, true)
	ui.pages.AddPage("help", ui.help, true, true)
	ui.pages.AddPage("chat", ui.chat, true, true)
	ui.pages.AddPage("filePicker", ui.filePicker, true, true)

	ui.app.SetRoot(ui.pages, true).SetFocus(ui.pages)

	return ui
}

func (ui *UI) Run() {
	//ui.OpenHome()
	ui.OpenChat()

	if err := ui.app.Run(); err != nil {
		panic(err)
	}
}

func (ui *UI) Quit() {
	ui.app.Stop()
}

func (ui *UI) CurrentPage() string {
	page, _ := ui.pages.GetFrontPage()
	return page
}

func (ui *UI) Open(pageName string) {
	ui.pages.SwitchToPage(pageName)
}

func (ui *UI) OpenHome() {
	ui.Open("home")
}

func (ui *UI) OpenHelp() {
	ui.Open("help")
}

func (ui *UI) OpenChat() {
	ui.Open("chat")
	ui.app.SetFocus(ui.chat.GetInitialFocus())
}

func (ui *UI) OpenFilePicker(prompt string, path string, callback func(string)) {
	ui.Open("filePicker")
	ui.filePicker.Setup(prompt, path, callback)
	ui.app.SetFocus(ui.filePicker.GetInitialFocus())
}

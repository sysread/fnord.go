package ui

import (
	"github.com/rivo/tview"

	"github.com/sysread/fnord/pkg/fnord"
)

type UI struct {
	Fnord *fnord.Fnord

	app    *tview.Application
	frame  *tview.Flex
	status *tview.TextView
	pages  *tview.Pages

	// Pages
	home       tview.Primitive
	help       tview.Primitive
	chat       *chatView
	filePicker *filePicker
}

func New() *UI {
	app := tview.NewApplication()
	app.EnableMouse(true)

	frame := tview.NewFlex().
		SetDirection(tview.FlexRow)

	status := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	status.SetText("Loading...")

	ui := &UI{
		Fnord:  fnord.NewFnord(),
		app:    app,
		frame:  frame,
		status: status,
		pages:  tview.NewPages(),
	}

	ui.home = ui.newHomeView()
	ui.help = ui.newHelpView()
	ui.chat = ui.newChatView()
	ui.filePicker = ui.newFilePicker()

	ui.pages.AddPage("home", ui.home, true, true)
	ui.pages.AddPage("help", ui.help, true, true)
	ui.pages.AddPage("chat", ui.chat, true, true)
	ui.pages.AddPage("filePicker", ui.filePicker, true, true)

	ui.frame.AddItem(ui.pages, 0, 1, true)
	ui.frame.AddItem(ui.status, 1, 0, false)

	ui.app.SetRoot(ui.frame, true).SetFocus(ui.pages)

	return ui
}

func (ui *UI) SetStatus(status string) {
	ui.status.SetText(status)
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

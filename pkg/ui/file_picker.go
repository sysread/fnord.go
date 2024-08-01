package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type filePicker struct {
	*tview.Frame
	filePickerInput *tview.InputField
}

func (ui *UI) newFilePicker() *filePicker {
	frame := tview.NewFrame(nil)
	frame.SetBorder(true)
	frame.SetBorderColor(tcell.ColorLightYellow)
	frame.SetBorders(0, 0, 1, 1, 0, 0)

	return &filePicker{
		Frame: frame,
	}
}

func (fp *filePicker) GetInitialFocus() tview.Primitive {
	return fp.filePickerInput
}

func (fp *filePicker) Setup(prompt string, path string, callback func(string)) {
	fp.filePickerInput = newFilePickerInput(path, callback)

	fp.Clear()
	fp.AddText("File Picker", true, tview.AlignCenter, tcell.ColorLightYellow)
	fp.AddText(prompt, true, tview.AlignCenter, tcell.ColorWhite)
	fp.AddText("ESC cancels", false, tview.AlignCenter, tcell.ColorLightYellow)
	fp.SetPrimitive(fp.filePickerInput)
}

func newFilePickerInput(path string, callback func(string)) *tview.InputField {
	files := listFiles(path)

	inputField := tview.NewInputField()
	inputField.SetLabel("Select a file: ")
	inputField.SetFieldWidth(30)

	inputField.
		SetDoneFunc(func(key tcell.Key) {
			switch key {
			case tcell.KeyEnter:
				callback(inputField.GetText())
			case tcell.KeyEscape:
				callback("")
			}
		}).
		SetAutocompletedFunc(func(text string, index, source int) bool {
			if source != tview.AutocompletedNavigate {
				inputField.SetText(text)
				callback(text)
			}

			return source == tview.AutocompletedEnter || source == tview.AutocompletedClick
		}).
		SetAutocompleteFunc(func(currentText string) []string {
			if len(currentText) == 0 {
				return nil
			}

			entries := []string{}
			for _, file := range files {
				if strings.HasPrefix(file, ".") {
					continue
				}

				if strings.Contains(strings.ToLower(file), strings.ToLower(currentText)) {
					entries = append(entries, file)
				}
			}

			return entries
		})

	return inputField
}

func listFiles(root string) []string {
	files := []string{}

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(root, path)

		if strings.HasPrefix(relPath, ".") {
			return nil
		}

		if err != nil {
			return err
		}

		files = append(files, relPath)

		return nil
	})

	return files
}

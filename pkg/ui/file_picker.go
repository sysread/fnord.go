package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rivo/tview"
)

type filePicker struct {
	path  string
	files []string

	*tview.Flex

	list  *tview.List
	input *tview.InputField
}

func (ui *UI) newFilePicker(prompt string, path string) *filePicker {
	fp := &filePicker{
		path:  path,
		files: listFiles(path),

		Flex: tview.NewFlex(),
		list: tview.NewList(),
		input: tview.NewInputField().
			SetLabel(prompt + ": ").
			SetFieldWidth(30),
	}

	fp.input.SetChangedFunc(fp.updateList)

	fp.SetDirection(tview.FlexRow)
	fp.AddItem(fp.input, 3, 1, true)
	fp.AddItem(fp.list, 0, 1, false)

	fp.updateList("")

	return fp
}

func (fp *filePicker) updateList(text string) {
	fp.list.Clear()

	for _, item := range fp.files {
		if strings.Contains(strings.ToLower(item), strings.ToLower(text)) {
			fp.list.AddItem(item, "", 0, func() {
				fp.input.SetText(item)
				fp.list.Clear()
			})
		}
	}
}

func listFiles(root string) []string {
	var files []string

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(root, path)

			if err != nil {
				return err
			}

			files = append(files, relPath)
		}

		return nil
	})

	return files
}

package console

import (
	"fmt"

	"github.com/sysread/fnord/pkg/storage"
)

// Function to handle list boxes command
func ListBoxes() {
	boxes, err := storage.GetBoxes()
	if err != nil {
		fmt.Printf("Error listing boxes: %v\n", err)
		return
	}

	if len(boxes) == 0 {
		fmt.Println("No boxes have been created yet.")
		return
	}

	fmt.Println("Available boxes:")
	for _, box := range boxes {
		fmt.Println("  - ", box)
	}
}

// Function to handle list projects command
func ListProjects() {
	projects, err := storage.GetProjects()
	if err != nil {
		fmt.Printf("Error listing projects: %v\n", err)
		return
	}

	if len(projects) == 0 {
		fmt.Println("No projects have been added yet.")
		return
	}

	fmt.Println("Available projects:")
	for _, project := range projects {
		fmt.Println("  - ", project)
	}
}

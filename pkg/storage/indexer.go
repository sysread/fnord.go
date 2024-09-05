package storage

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/h2non/filetype"
	"github.com/philippgille/chromem-go"
	gitignore "github.com/sabhiram/go-gitignore"

	"github.com/sysread/fnord/pkg/debug"
)

var ProjectGitIgnored *gitignore.GitIgnore

func startIndexer() {
	debug.Log("Indexing project directory %s", ProjectPath)

	if ProjectPath == "" {
		debug.Log("ProjectPath not set")
		return
	}

	if ProjectFiles == nil {
		debug.Log("ProjectFiles collection not initialized")
		return
	}

	var err error

	// Load the .gitignore file
	ProjectGitIgnored, err = gitignore.CompileIgnoreFile(filepath.Join(ProjectPath, ".gitignore"))
	if err != nil {
		panic(fmt.Errorf("error loading .gitignore: %v", err))
	}

	// On init, ensure that everything that is not git-ignored in the
	// project directory is indexed.
	var toIndex []string

	// Queue files for indexing
	walkProjectDir(func(path string) {
		toIndex = append(toIndex, path)
	})

	// Queue them for indexing
	indexPaths(toIndex)

	// Start the directory watcher, and index new files or files that have changed.
	err = watchProjectDir()
	if err != nil {
		panic(fmt.Errorf("failed to start directory watcher: %v", err))
	}
}

func walkProjectDir(fn func(path string)) error {
	err := filepath.WalkDir(ProjectPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		if !canIndex(path) {
			return nil
		}


		fn(path)

		return nil
	})

	if err != nil {
		return fmt.Errorf("error performing initial index of project directory: %v", err)
	}

	return nil
}

// watchProjectDir sets up a recursive watcher for the project directory
/*func watchProjectDir() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	//defer watcher.Close()

	// Add initial directory and subdirectories
	err = addDirRecursive(watcher, ProjectPath)
	if err != nil {
		return fmt.Errorf("failed to add directories to watcher: %v", err)
	}

	// Start watching in a goroutine
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Handle file create, write, and rename events
				if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) != 0 {
					// Ensure it's not a directory
					info, err := os.Stat(event.Name)

					if err != nil {
						debug.Log("Error getting file info: %v", err)
						continue
					}

					if info.IsDir() {
						// If it's a directory, add it to the watcher recursively
						if err = addDirRecursive(watcher, event.Name); err != nil {
							debug.Log("Failed to add new directory to watcher: %v", err)
						}
					} else {
						// Queue file for indexing
						indexPath(event.Name)
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}

				debug.Log("Watcher error: %v", err)
			}
		}
	}()

	// Block until the program exits
	done := make(chan bool)
	<-done

	return nil
}*/

// watchProjectDir sets up a recursive watcher for the project directory
func watchProjectDir() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	// defer watcher.Close()

	// Add initial directory and subdirectories
	err = addDirRecursive(watcher, ProjectPath)
	if err != nil {
		return fmt.Errorf("failed to add directories to watcher: %v", err)
	}

	// Start watching in a goroutine
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Handle file create, write, rename, and remove events
				switch {
				case event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) != 0:
					// Ensure it's not a directory
					info, err := os.Stat(event.Name)

					if err != nil {
						debug.Log("Error getting file info: %v", err)
						continue
					}

					if info.IsDir() {
						// If it's a directory, add it to the watcher recursively
						if err = addDirRecursive(watcher, event.Name); err != nil {
							debug.Log("Failed to add new directory to watcher: %v", err)
						}
					} else {
						// Queue file for indexing
						indexPath(event.Name)
					}

				case event.Op&fsnotify.Remove != 0:
					// Handle file/directory removal
					debug.Log("File or directory removed: %s", event.Name)

					info, err := os.Stat(event.Name)
					if err != nil {
						// Remove from index
						removeFromIndex(event.Name)
						continue
					}

					if info.IsDir() {
						watcher.Remove(event.Name)
						debug.Log("Deleted directory removed from watcher: %s", event.Name)
					} else {
						// Remove from index
						removeFromIndex(event.Name)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}

				debug.Log("Watcher error: %v", err)
			}
		}
	}()

	// Block until the program exits
	done := make(chan bool)
	<-done

	return nil
}

// Recursively add directories to the watcher
func addDirRecursive(watcher *fsnotify.Watcher, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if strings.HasSuffix(path, "/.git") {
				return filepath.SkipDir
			}

			err = watcher.Add(path)
			if err != nil {
				return err
			}

			debug.Log("Watching directory: %s", path)
		}

		return nil
	})
}

func indexPath(path string) {
	if !canIndex(path) {
		return
	}

	doc, err := toChromemDocument(path)
	if err != nil {
		debug.Log("Error converting file to document: %v", err)
		return
	}

	debug.Log("  - index: %s", path)
	ProjectFiles.AddDocuments(context.Background(), []chromem.Document{doc}, 2)
}

func removeFromIndex(path string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		debug.Log("Error getting absolute path: %v", err)
		return
	}

	debug.Log("  - remove from index: %s", absPath)
	ProjectFiles.Delete(context.Background(), nil, nil, absPath)
}

func indexPaths(paths []string) {
	var toIndex []chromem.Document

	for _, path := range paths {
		if !canIndex(path) {
			continue
		}

		doc, err := toChromemDocument(path)
		if err != nil {
			debug.Log("Error converting file to document: %v", err)
			continue
		}

		debug.Log("  - index: %s", path)
		toIndex = append(toIndex, doc)
	}

	ProjectFiles.AddDocuments(context.Background(), toIndex, 4)
}

func toChromemDocument(path string) (chromem.Document, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return chromem.Document{}, fmt.Errorf("error reading file %s: %v", path, err)
	}

	return chromem.Document{
		ID:      path,
		Content: string(buf),
		Metadata: map[string]string{
			"hash": path,
		},
	}, nil
}

func canIndex(path string) bool {
	if isGitIgnored(path) {
		return false
	}

	// Check if the file is binary
	isBinary, err := isBinaryFile(path)
	if err != nil {
		debug.Log("Error checking file type for %s: %v", path, err)
		return false
	}
	if isBinary {
		return false
	}

	return true
}

// isBinaryFile checks if a file is a binary file using its magic number.
func isBinaryFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Read the first 512 bytes or less, depending on the file size
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}

	// Use only the bytes that were read
	buf = buf[:n]

	// Use filetype package to check the MIME type of the file
	kind, _ := filetype.Match(buf)

	// Treat anything that filetype does not recognize as binary
	if kind != filetype.Unknown {
		return true, nil
	}

	// Alternatively, you can check for specific MIME types, like images or executables
	return false, nil
}

// isGitIgnored checks if a file is ignored by git
func isGitIgnored(path string) bool {
	// If the path contains the .git directory, ignore it
	if strings.Contains(path, "/.git/") {
		return true
	}

	relpath, err := filepath.Rel(ProjectPath, path)
	if err != nil {
		debug.Log("Error getting relative path: %v", err)
		return false
	}

	return ProjectGitIgnored.MatchesPath(relpath)
}

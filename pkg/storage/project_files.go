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

	"github.com/sysread/fnord/pkg/config"
	"github.com/sysread/fnord/pkg/debug"
)

// Project path is the path to a project directory, selected by the user via
// the --project flag, to be indexed by the service. This is optional. If
// unset, the service will not index a project directory.
var ProjectPath string

// ProjectFiles is the chromem collection of files in the project directory
var ProjectFiles *chromem.Collection

// ProjectGitIgnored is the gitignore parser for the project directory
var ProjectGitIgnored *gitignore.GitIgnore

func InitializeProjectFilesCollection(config *config.Config) error {
	debug.Log("[storage] [project] Initializing project files collection from root path %s", config.ProjectPath)
	var err error

	gitPath := filepath.Join(config.ProjectPath, ".git")
	if _, err := os.Stat(gitPath); err != nil {
		debug.Log("[storage] [project] Project path %s is not a git repository", config.ProjectPath)
		return fmt.Errorf("project path %s is not a git repository", config.ProjectPath)
	}

	ProjectPath = config.ProjectPath

	collectionName := fmt.Sprintf("project_files:%s", ProjectPath)
	ProjectFiles, err = DB.GetOrCreateCollection(collectionName, nil, nil)
	if err != nil {
		debug.Log("[storage] [project] Error creating %s collection: %v", collectionName, err)
	} else {
		// Load the .gitignore file
		ProjectGitIgnored, err = gitignore.CompileIgnoreFile(filepath.Join(ProjectPath, ".gitignore"))
		if err != nil {
			return err
		}

		go startIndexer()
	}

	return nil
}

// Function to list all projects' collections
func GetProjects() ([]string, error) {
	collections := DB.ListCollections()
	var projects []string

	for name := range collections {
		// We exclude project files' collections based on their naming pattern
		if strings.HasPrefix(name, "project_files:") {
			name = strings.TrimPrefix(name, "project_files:")
			projects = append(projects, name)
		}
	}

	return projects, nil
}

// Searches the project file index for the given query and returns the results.
func SearchProject(query string, numResults int) ([]Result, error) {
	debug.Log("[storage] [project] Searching project files for %d results using query: '%s'", numResults, query)

	if ProjectFiles == nil {
		return []Result{}, nil
	}

	maxResults := ProjectFiles.Count()
	if numResults > maxResults {
		numResults = maxResults
	}

	if numResults == 0 {
		debug.Log("[storage] [project] No indexed project files to search!")
		return []Result{}, nil
	}

	results, err := ProjectFiles.Query(context.Background(), query, numResults, nil, nil)
	if err != nil {
		debug.Log("[storage] [project] Error querying project files: %v", err)
		return nil, err
	}

	var found []Result
	for _, doc := range results {
		debug.Log("[storage] [project] Found project file: %s", doc.ID)

		found = append(found, Result{
			ID:      doc.ID,
			Content: doc.Content,
		})
	}

	return found, nil
}

// startIndexer initializes the project file indexer and starts watching the
// project directory for changes.
func startIndexer() {
	debug.Log("[storage] [project] Indexing project directory %s", ProjectPath)

	if ProjectPath == "" {
		debug.Log("[storage] [project] ProjectPath not set")
		return
	}

	if ProjectFiles == nil {
		debug.Log("[storage] [project] ProjectFiles collection not initialized")
		return
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
	err := watchProjectDir()
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
						debug.Log("[storage] [project] Error getting file info: %v", err)
						continue
					}

					if info.IsDir() {
						// If it's a directory, add it to the watcher recursively
						if err = addDirRecursive(watcher, event.Name); err != nil {
							debug.Log("[storage] [project] Failed to add new directory to watcher: %v", err)
						}
					} else {
						// Queue file for indexing
						indexPath(event.Name)
					}

				case event.Op&fsnotify.Remove != 0:
					// Handle file/directory removal
					debug.Log("[storage] [project] File or directory removed: %s", event.Name)

					info, err := os.Stat(event.Name)
					if err != nil {
						// Remove from index
						removeFromIndex(event.Name)
						continue
					}

					if info.IsDir() {
						watcher.Remove(event.Name)
						debug.Log("[storage] [project] Deleted directory removed from watcher: %s", event.Name)
					} else {
						// Remove from index
						removeFromIndex(event.Name)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}

				debug.Log("[storage] [project] Watcher error: %v", err)
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

			if isGitIgnored(path) {
				return filepath.SkipDir
			}

			err = watcher.Add(path)
			if err != nil {
				return err
			}

			debug.Log("[storage] [project] Watching directory: %s", path)
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
		debug.Log("[storage] [project] Error converting file to document: %v", err)
		return
	}

	debug.Log("[storage] [project]   - index: %s", path)
	ProjectFiles.AddDocuments(context.Background(), []chromem.Document{doc}, 2)
}

func removeFromIndex(path string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		debug.Log("[storage] [project] Error getting absolute path: %v", err)
		return
	}

	debug.Log("[storage] [project]   - remove from index: %s", absPath)
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
			debug.Log("[storage] [project] Error converting file to document: %v", err)
			continue
		}

		debug.Log("[storage] [project]   - index: %s", path)
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
		debug.Log("[storage] [project] Error checking file type for %s: %v", path, err)
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
		debug.Log("[storage] [project] Error getting relative path: %v", err)
		return false
	}

	return ProjectGitIgnored.MatchesPath(relpath)
}

// Result represents a search result
func (r *Result) ProjectFileString() string {
	path := r.ID
	content := r.Content
	return fmt.Sprintf("Project file: %s\n%s\n\n", path, content)
}

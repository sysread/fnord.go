package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
)

const (
	DefaultBox = "default"
)

type Config struct {
	Help         bool
	Testing      bool
	Home         string
	Box          string
	BoxPath      string
	OpenAIApiKey string
	OpenAIAsstId string
}

func Getopts() *Config {
	config := &Config{
		Testing: false,
	}

	config.
		SetEnvOptions().
		ReadCommandLineOptions().
		SetTestingOverrides().
		SetBoxPath()

	if config.Help {
		fmt.Println("Usage: fnord [options]")
		pflag.PrintDefaults()
		os.Exit(0)
	}

	return config.
		validateOpenAIApiKey().
		validateOpenAIAsstId().
		validateBox().
		validateBoxPath()
}

//------------------------------------------------------------------------------
// Setters
//------------------------------------------------------------------------------

func (c *Config) ReadCommandLineOptions() *Config {
	pflag.BoolVarP(&c.Help, "help", "h", false, "display this help message")
	pflag.BoolVarP(&c.Testing, "testing", "t", false, "enable testing mode (forces --box to be 'testing')")
	pflag.StringVarP(&c.Box, "box", "b", DefaultBox, "boxes are isolated workspaces; conversations held within a box are isolated from other boxes")
	pflag.Parse()
	return c
}

func (c *Config) SetEnvOptions() *Config {
	c.OpenAIApiKey = os.Getenv("FNORD_OPENAI_API_KEY")
	c.OpenAIAsstId = os.Getenv("FNORD_OPENAI_ASST_ID")

	if os.Getenv("FNORD_TESTING") == "true" || os.Getenv("FNORD_TESTING") == "1" {
		c.Testing = true
	}

	// Determine the base directory.
	basePath := os.Getenv("FNORD_HOME")
	if basePath == "" {
		basePath = filepath.Join(os.Getenv("HOME"), ".config", "fnord")
	}

	// Ensure the base directory exists.
	if !makeDir(basePath) {
		die("Could not create base directory (%s)", basePath)
	}

	c.Home = basePath

	return c
}

func (c *Config) SetTestingOverrides() *Config {
	if c.Testing {
		c.Box = "testing"
	}

	return c
}

func (c *Config) SetBoxPath() *Config {
	if c.Box != "" {
		c.BoxPath = path.Join(c.Home, c.Box)
	}

	return c
}

//------------------------------------------------------------------------------
// Validation
//------------------------------------------------------------------------------

func (c *Config) validateOpenAIApiKey() *Config {
	if c.OpenAIApiKey == "" {
		die("FNORD_OPENAI_API_KEY must be set in the shell environment")
	}

	return c
}

func (c *Config) validateOpenAIAsstId() *Config {
	if c.OpenAIAsstId == "" {
		die("FNORD_OPENAI_ASST_ID must be set in the shell environment")
	}

	return c
}

func (c *Config) validateBox() *Config {
	if c.Box == "" {
		die("Box name cannot be empty")
	}

	if strings.Contains(c.Box, "/") {
		die("Box name cannot contain '/'")
	}

	return c
}

func (c *Config) validateBoxPath() *Config {
	// BoxPath should always be set by SetBoxPath if Box itself is set.
	if c.BoxPath == "" {
		die("Box name cannot be empty")
	}

	// Ensure the box directory exists.
	if !makeDir(c.BoxPath) {
		die("Could not create box directory (%s)", c.BoxPath)
	}

	return c
}

//------------------------------------------------------------------------------
// Helper functions
//------------------------------------------------------------------------------

func die(fmtString string, args ...interface{}) {
	panic(fmt.Sprintf(fmtString, args...))
}

func pathExists(path string) bool {
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		return false
	}

	return err == nil
}

func makeDir(path string) bool {
	if err := os.MkdirAll(path, 0700); err != nil {
		die("Could not create directory (%s): %s", path, err)
	}

	return pathExists(path)
}

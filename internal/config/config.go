// Package config loads the user's brewupdater settings from a YAML file.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// Config is the user's settings, as read from config.yaml.
type Config struct {
	Favorites  []string `yaml:"favorites"`
	Greedy     bool     `yaml:"greedy"`
	AutoUpdate bool     `yaml:"auto_update"`
}

// Default is what applies when the file or a key is absent: auto-update on, not greedy.
func Default() Config {
	return Config{AutoUpdate: true}
}

// Path returns the config file location: XDG_CONFIG_HOME if set to an absolute path, else ~/.config.
func Path() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if !filepath.IsAbs(base) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}

	return filepath.Join(base, "brewupdater", "config.yaml"), nil
}

// Load reads path; a missing file returns Default() and no error.
func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("reading config %s: %w", path, err)
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
		return Config{}, fmt.Errorf("parsing config %s: %w", path, err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Config{}, fmt.Errorf("parsing config %s: multiple documents are not supported", path)
	}

	return cfg, nil
}

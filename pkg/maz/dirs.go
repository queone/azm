package maz

import (
	"errors"
	"os"
	"path/filepath"
)

const appDirName = "maz" // subdirectory under each XDG base directory

// Dirs holds the resolved configuration and cache directories.
type Dirs struct {
	Config string // credentials and token cache
	Cache  string // cached object snapshots
}

// ResolveDirs picks the configuration and cache directories following the XDG
// Base Directory convention with a fallback to the legacy ~/.maz directory.
// Each directory resolves independently: the XDG location wins when it exists,
// else an existing legacy directory is used, else the XDG location is chosen
// so that the caller creates it. getenv looks up environment variables and
// exists reports whether a path is an existing directory, so the rules can be
// tested without touching the real environment or filesystem.
func ResolveDirs(home string, getenv func(string) string, exists func(string) bool) (Dirs, error) {
	if home == "" {
		return Dirs{}, errors.New("home directory is empty")
	}
	legacy := filepath.Join(home, ConfigBaseDir)
	config := filepath.Join(xdgBase(getenv("XDG_CONFIG_HOME"), home, ".config"), appDirName)
	cache := filepath.Join(xdgBase(getenv("XDG_CACHE_HOME"), home, ".cache"), appDirName)
	return Dirs{
		Config: pickDir(config, legacy, exists),
		Cache:  pickDir(cache, legacy, exists),
	}, nil
}

// xdgBase returns the XDG base directory value, or the default under home
// when the variable is unset or empty.
func xdgBase(value, home, defaultName string) string {
	if value != "" {
		return value
	}
	return filepath.Join(home, defaultName)
}

// pickDir returns preferred when it exists, else legacy when that exists,
// else preferred so that it gets created.
func pickDir(preferred, legacy string, exists func(string) bool) string {
	if exists(preferred) {
		return preferred
	}
	if exists(legacy) {
		return legacy
	}
	return preferred
}

// dirExists reports whether path is an existing directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

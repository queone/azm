package maz

import (
	"path/filepath"
	"testing"
)

// fakeHome is a relative stand-in for the home directory so the tests touch no real path.
const fakeHome = "fake-home"

func fakeEnv(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func fakeExists(paths ...string) func(string) bool {
	set := map[string]bool{}
	for _, p := range paths {
		set[p] = true
	}
	return func(p string) bool { return set[p] }
}

func TestResolveDirsUsesXdgVariablesWhenSet(t *testing.T) {
	env := fakeEnv(map[string]string{"XDG_CONFIG_HOME": "/x/cfg", "XDG_CACHE_HOME": "/x/cache"})
	got, err := ResolveDirs(fakeHome, env, fakeExists())
	if err != nil {
		t.Fatal(err)
	}
	want := Dirs{Config: filepath.Join("/x/cfg", "maz"), Cache: filepath.Join("/x/cache", "maz")}
	if got != want {
		t.Fatalf("ResolveDirs() = %+v, want %+v", got, want)
	}
}

func TestResolveDirsDefaultsUnderHomeWithoutLegacy(t *testing.T) {
	got, err := ResolveDirs(fakeHome, fakeEnv(nil), fakeExists())
	if err != nil {
		t.Fatal(err)
	}
	want := Dirs{Config: filepath.Join(fakeHome, ".config", "maz"), Cache: filepath.Join(fakeHome, ".cache", "maz")}
	if got != want {
		t.Fatalf("ResolveDirs() = %+v, want %+v", got, want)
	}
}

func TestResolveDirsFallsBackToLegacyDirectory(t *testing.T) {
	legacy := filepath.Join(fakeHome, ".maz")
	got, err := ResolveDirs(fakeHome, fakeEnv(nil), fakeExists(legacy))
	if err != nil {
		t.Fatal(err)
	}
	want := Dirs{Config: legacy, Cache: legacy}
	if got != want {
		t.Fatalf("ResolveDirs() = %+v, want %+v", got, want)
	}
}

func TestResolveDirsPrefersXdgDirectoryOverLegacyPerDirectory(t *testing.T) {
	legacy := filepath.Join(fakeHome, ".maz")
	xdgConfig := filepath.Join(fakeHome, ".config", "maz")
	got, err := ResolveDirs(fakeHome, fakeEnv(nil), fakeExists(legacy, xdgConfig))
	if err != nil {
		t.Fatal(err)
	}
	want := Dirs{Config: xdgConfig, Cache: legacy}
	if got != want {
		t.Fatalf("ResolveDirs() = %+v, want %+v", got, want)
	}
}

func TestResolveDirsRejectsEmptyHome(t *testing.T) {
	if _, err := ResolveDirs("", fakeEnv(nil), fakeExists()); err == nil {
		t.Fatal("ResolveDirs(\"\") returned no error, want one")
	}
}

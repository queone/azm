package main

import (
	"regexp"
	"testing"
)

// strictStableSemver mirrors the canonical build contract for programVersion:
// MAJOR.MINOR.PATCH with no leading zeroes, prerelease, or build metadata.
var strictStableSemver = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func TestProgramVersionIsStrictStableSemver(t *testing.T) {
	if !strictStableSemver.MatchString(programVersion) {
		t.Fatalf("programVersion %q must match MAJOR.MINOR.PATCH", programVersion)
	}
}

func TestVersionLineMatchesDeclaredVersion(t *testing.T) {
	want := "raf v" + programVersion
	if got := versionLine(); got != want {
		t.Fatalf("versionLine() = %q, want %q", got, want)
	}
}

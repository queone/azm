package utl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJsonAndYamlFilesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	obj := map[string]any{"name": "azm", "count": float64(2)}

	jsonPath := filepath.Join(dir, "obj.json")
	SaveFileJson(obj, jsonPath, false)
	loaded, format, err := LoadFileAuto(jsonPath)
	if err != nil || format != "json" || Map(loaded)["name"] != "azm" {
		t.Fatalf("LoadFileAuto(json) = %v, %q, %v", loaded, format, err)
	}

	yamlPath := filepath.Join(dir, "obj.yaml")
	if err := SaveFileAuto(yamlPath, "yaml", obj, false, 0); err != nil {
		t.Fatal(err)
	}
	y, err := LoadFileYaml(yamlPath)
	if err != nil || Map(y)["name"] != "azm" {
		t.Fatalf("LoadFileYaml() = %v, %v", y, err)
	}
	info, _ := os.Stat(yamlPath)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("SaveFileAuto() perm = %o, want 600", info.Mode().Perm())
	}
}

func TestFileStateHelpers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if FileExist(path) || FileUsable(path) || FileAge(path) != -1 {
		t.Fatal("helpers must report a missing file")
	}
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !FileExist(path) || !FileUsable(path) || FileAge(path) < 0 {
		t.Fatal("helpers must report an existing non-empty file")
	}
	if err := RemoveFile(path); err != nil || FileExist(path) {
		t.Fatalf("RemoveFile() = %v, exists=%v", err, FileExist(path))
	}
	if err := RemoveFile(path); err != nil {
		t.Fatalf("RemoveFile() on a missing file = %v, want nil", err)
	}
}

func TestJsonToBytesIndentUsesRequestedIndent(t *testing.T) {
	b, err := JsonToBytesIndent(map[string]any{"a": 1}, 4)
	if err != nil || string(b) != "{\n    \"a\": 1\n}" {
		t.Fatalf("JsonToBytesIndent() = %q, %v", b, err)
	}
}

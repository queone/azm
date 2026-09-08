package utl

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	prev := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = prev
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintYamlColorRendersEveryKeyAndValue(t *testing.T) {
	prev := enabled
	enabled = false
	defer func() { enabled = prev }()
	out := captureStdout(t, func() { PrintYamlColor(map[string]any{"name": "azm", "count": 2}) })
	for _, want := range []string{"name: azm", "count: 2"} {
		if !strings.Contains(out, want) {
			t.Fatalf("PrintYamlColor() output %q lacks %q", out, want)
		}
	}
}

func TestPrintJsonColorRendersIndentedJson(t *testing.T) {
	prev := enabled
	enabled = false
	defer func() { enabled = prev }()
	out := captureStdout(t, func() { PrintJsonColor(map[string]any{"name": "azm"}) })
	if !strings.Contains(out, "\"name\": \"azm\"") {
		t.Fatalf("PrintJsonColor() output %q lacks the key", out)
	}
}

func TestPrintYamlColorEmitsEscapesWhenEnabled(t *testing.T) {
	prev := enabled
	enabled = true
	defer func() { enabled = prev }()
	out := captureStdout(t, func() { PrintYamlColor(map[string]any{"name": "azm"}) })
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("PrintYamlColor() emitted no escapes while enabled: %q", out)
	}
}

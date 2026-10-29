package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInputBytesFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	content := []byte(`{"hello":"world"}`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	got, err := inputBytes([]string{path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestInputBytesMissingFile(t *testing.T) {
	_, err := inputBytes([]string{filepath.Join(t.TempDir(), "does-not-exist.json")})
	if err == nil {
		t.Error("expected error reading a missing file, got nil")
	}
}

func TestInputBytesTooManyArgs(t *testing.T) {
	_, err := inputBytes([]string{"a.json", "b.json"})
	if err == nil {
		t.Fatal("expected usage error for too many args, got nil")
	}
}

func TestInputBytesEndToEnd(t *testing.T) {
	// A file read through inputBytes should parse and format cleanly,
	// exercising the main pipeline wiring (lexer -> parser -> formatter).
	withColor(t, false)

	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	if err := os.WriteFile(path, []byte(`{"name":"Ada","age":36}`), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	input, err := inputBytes([]string{path})
	if err != nil {
		t.Fatalf("inputBytes error: %v", err)
	}

	value, err := NewParser(NewLexer(input)).Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	got := FormatTerminal(value)
	want := "age = 36\nname = Ada"
	if got != want {
		t.Errorf("formatted output = %q, want %q", got, want)
	}
}

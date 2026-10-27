package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParserValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  any
	}{
		{name: "null", input: "null", want: nil},
		{name: "true", input: "true", want: true},
		{name: "false", input: "false", want: false},
		{name: "string", input: `"hello"`, want: "hello"},
		{name: "escaped string", input: `"a\nb"`, want: "a\nb"},
		{name: "integer", input: "-42", want: json.Number("-42")},
		{name: "decimal", input: "3.125", want: json.Number("3.125")},
		{name: "exponent", input: "6.02e23", want: json.Number("6.02e23")},
		{name: "empty array", input: "[]", want: []any{}},
		{name: "mixed array", input: `[1, "two", false, null]`, want: []any{json.Number("1"), "two", false, nil}},
		{name: "nested object", input: `{"user":{"name":"Ada"}}`, want: map[string]any{"user": map[string]any{"name": "Ada"}}},
		{name: "empty object", input: "{}", want: map[string]any{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewParser(NewLexer([]byte(test.input))).Parse()
			if err != nil {
				t.Fatalf("Parse returned error: %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Parse = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestValidJSONFixtures(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("testdata", "valid", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) < 20 {
		t.Fatalf("expected at least 20 valid fixtures, found %d", len(paths))
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			input, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := NewParser(NewLexer(input)).Parse(); err != nil {
				t.Fatalf("valid fixture rejected: %v", err)
			}
		})
	}
}

func TestInvalidJSONFixtures(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("testdata", "invalid", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) < 20 {
		t.Fatalf("expected at least 20 invalid fixtures, found %d", len(paths))
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			input, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := NewParser(NewLexer(input)).Parse(); err == nil {
				t.Fatal("invalid fixture was accepted")
			}
		})
	}
}

func TestLexerTokenSequence(t *testing.T) {
	lexer := NewLexer([]byte(`{"a":[true, 12, null]}`))
	want := []TokenType{
		TokenLeftBrace, TokenString, TokenColon, TokenLeftBracket,
		TokenTrue, TokenComma, TokenNumber, TokenComma, TokenNull,
		TokenRightBracket, TokenRightBrace, TokenEOF,
	}
	for i, wantType := range want {
		tok, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("token %d: %v", i, err)
		}
		if tok.Type != wantType {
			t.Fatalf("token %d type = %s, want %s", i, tok.Type, wantType)
		}
	}
}

func TestFormatTerminal(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{
			name: "object has equals and expands nested objects",
			value: map[string]any{
				"z":      "last",
				"a":      "first",
				"items":  []any{"one", json.Number("2")},
				"nested": map[string]any{"ok": true},
			},
			want: "a = first\nitems = [\n  one,\n  2\n]\nnested\n  ok = true\nz = last",
		},
		{name: "root array", value: []any{json.Number("1"), "two"}, want: "[\n  1,\n  two\n]"},
		{name: "root string has no quotes", value: "hello", want: "hello"},
		{name: "strings escape terminal newlines", value: "first\nsecond", want: "first\\nsecond"},
		{name: "array of objects has no braces", value: []any{map[string]any{"name": "Ada"}, map[string]any{"name": "Lin"}}, want: "[\n  name = Ada,\n  name = Lin\n]"},
		{name: "nested arrays", value: []any{[]any{json.Number("1"), json.Number("2")}, []any{json.Number("3")}}, want: "[\n  [\n    1,\n    2\n  ],\n  [\n    3\n  ]\n]"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := FormatTerminal(test.value); got != test.want {
				t.Fatalf("FormatTerminal = %q, want %q", got, test.want)
			}
		})
	}
}

package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

// parse is a helper that lexes and parses the given input in one step.
func parse(t *testing.T, input string) (any, error) {
	t.Helper()
	return NewParser(NewLexer([]byte(input))).Parse()
}

func TestNewParser(t *testing.T) {
	p := NewParser(NewLexer([]byte("null")))
	if p == nil {
		t.Fatal("NewParser returned nil")
	}
	if p.loaded {
		t.Error("new parser should not have a token loaded yet")
	}
}

func TestParseScalars(t *testing.T) {
	cases := []struct {
		input string
		want  any
	}{
		{`"hello"`, "hello"},
		{`42`, json.Number("42")},
		{`-3.14`, json.Number("-3.14")},
		{`true`, true},
		{`false`, false},
		{`null`, nil},
	}
	for _, c := range cases {
		got, err := parse(t, c.input)
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", c.input, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q: got %#v, want %#v", c.input, got, c.want)
		}
	}
}

func TestParseEmptyArray(t *testing.T) {
	got, err := parse(t, "[]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := got.([]any)
	if !ok {
		t.Fatalf("got %T, want []any", got)
	}
	if len(arr) != 0 {
		t.Errorf("len = %d, want 0", len(arr))
	}
}

func TestParseArray(t *testing.T) {
	got, err := parse(t, `[1, "two", true, null]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []any{json.Number("1"), "two", true, nil}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestParseNestedArray(t *testing.T) {
	got, err := parse(t, `[[1, 2], [3]]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []any{
		[]any{json.Number("1"), json.Number("2")},
		[]any{json.Number("3")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestParseEmptyObject(t *testing.T) {
	got, err := parse(t, "{}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("got %T, want map[string]any", got)
	}
	if len(obj) != 0 {
		t.Errorf("len = %d, want 0", len(obj))
	}
}

func TestParseObject(t *testing.T) {
	got, err := parse(t, `{"name": "Ada", "age": 36, "active": true}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{
		"name":   "Ada",
		"age":    json.Number("36"),
		"active": true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestParseNestedObject(t *testing.T) {
	got, err := parse(t, `{"outer": {"inner": [1, 2]}}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{
		"outer": map[string]any{
			"inner": []any{json.Number("1"), json.Number("2")},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"empty input", ""},
		{"trailing content", `{} extra`},
		{"trailing comma array", `[1,]`},
		{"trailing comma object", `{"a":1,}`},
		{"missing colon", `{"a" 1}`},
		{"non-string key", `{1: 2}`},
		{"unclosed array", `[1, 2`},
		{"unclosed object", `{"a": 1`},
		{"missing comma array", `[1 2]`},
		{"bare value then junk", `true false`},
		{"just a comma", `,`},
		{"lexer error propagates", `[01]`},
	}
	for _, c := range cases {
		if _, err := parse(t, c.input); err == nil {
			t.Errorf("%s (%q): expected error, got nil", c.name, c.input)
		}
	}
}

func TestParseWhitespaceTolerant(t *testing.T) {
	got, err := parse(t, "\n\t  {  \"k\" : \n 1 }  \n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"k": json.Number("1")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

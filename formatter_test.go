package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// withColor temporarily enables colorEnabled for a test and restores it after.
func withColor(t *testing.T, on bool) {
	t.Helper()
	prev := colorEnabled
	colorEnabled = on
	t.Cleanup(func() { colorEnabled = prev })
}

func TestRender(t *testing.T) {
	withColor(t, false)
	if got := render(keyStyle, "plain"); got != "plain" {
		t.Errorf("render with color off = %q, want %q", got, "plain")
	}

	withColor(t, true)
	if got := render(keyStyle, "styled"); !strings.Contains(got, "styled") {
		t.Errorf("render with color on = %q, want to contain %q", got, "styled")
	}
}

func TestFormatTerminalEmptyObject(t *testing.T) {
	withColor(t, false)
	if got := FormatTerminal(map[string]any{}); got != "(empty object)" {
		t.Errorf("got %q, want %q", got, "(empty object)")
	}
}

func TestFormatTerminalScalarObject(t *testing.T) {
	withColor(t, false)
	value := map[string]any{"name": "Ada", "age": json.Number("36")}
	got := FormatTerminal(value)
	// Keys are sorted, so "age" precedes "name".
	want := "age = 36\nname = Ada"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatTerminalNestedObject(t *testing.T) {
	withColor(t, false)
	value := map[string]any{
		"user": map[string]any{"id": json.Number("1")},
	}
	got := FormatTerminal(value)
	want := "user\n  id = 1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatTerminalNestedEmptyObject(t *testing.T) {
	withColor(t, false)
	value := map[string]any{"meta": map[string]any{}}
	got := FormatTerminal(value)
	want := "meta\n  (empty object)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatTerminalTopLevelScalar(t *testing.T) {
	withColor(t, false)
	if got := FormatTerminal("hello"); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
	if got := FormatTerminal(nil); got != "null" {
		t.Errorf("got %q, want %q", got, "null")
	}
}

func TestFormatArray(t *testing.T) {
	withColor(t, false)
	got := formatValue([]any{json.Number("1"), json.Number("2")}, 0)
	want := "[\n  1,\n  2\n]"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatEmptyArray(t *testing.T) {
	withColor(t, false)
	if got := formatValue([]any{}, 0); got != "[]" {
		t.Errorf("got %q, want %q", got, "[]")
	}
}

func TestFormatArrayOfObjects(t *testing.T) {
	withColor(t, false)
	value := []any{
		map[string]any{"a": json.Number("1"), "b": json.Number("2")},
	}
	got := formatValue(value, 0)
	// First field shares the bullet line, second aligns beneath.
	want := "[\n  a = 1\n  b = 2\n]"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatArrayOfObjectsNested(t *testing.T) {
	withColor(t, false)
	// An object inside an array that itself contains a nested object and an
	// empty object exercises the formatArrayObject nested branches.
	value := []any{
		map[string]any{
			"inner": map[string]any{"x": json.Number("1")},
			"empty": map[string]any{},
			"name":  "z",
		},
	}
	got := formatValue(value, 0)
	for _, want := range []string{"inner", "x = 1", "empty", "(empty object)", "name = z"} {
		if !strings.Contains(got, want) {
			t.Errorf("array-of-objects output missing %q; got:\n%s", want, got)
		}
	}
}

func TestFormatObjectWithArrayAndNested(t *testing.T) {
	withColor(t, false)
	// A top-level object whose fields include an array and a nested object
	// exercises formatValue's []any and map branches.
	value := map[string]any{
		"items":  []any{json.Number("1"), map[string]any{"k": "v"}},
		"nested": map[string]any{"deep": map[string]any{"d": json.Number("9")}},
	}
	got := FormatTerminal(value)
	for _, want := range []string{"items", "nested", "deep", "d = 9", "k = v"} {
		if !strings.Contains(got, want) {
			t.Errorf("object output missing %q; got:\n%s", want, got)
		}
	}
}

func TestFormatInline(t *testing.T) {
	withColor(t, false)
	cases := []struct {
		value any
		want  string
	}{
		{nil, "null"},
		{true, "true"},
		{false, "false"},
		{"text", "text"},
		{json.Number("42"), "42"},
		{map[string]any{"a": json.Number("1"), "b": "x"}, "a = 1; b = x"},
	}
	for _, c := range cases {
		if got := formatInline(c.value); got != c.want {
			t.Errorf("formatInline(%#v) = %q, want %q", c.value, got, c.want)
		}
	}
}

func TestSortedKeys(t *testing.T) {
	object := map[string]any{"banana": 1, "apple": 2, "cherry": 3}
	got := sortedKeys(object)
	want := []string{"apple", "banana", "cherry"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if len(sortedKeys(map[string]any{})) != 0 {
		t.Error("sortedKeys of empty map should be empty")
	}
}

func TestCleanTerminalText(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"plain", "plain"},
		{"line\nbreak", `line\nbreak`},
		{"tab\there", `tab\there`},
		{"carriage\rreturn", `carriage\rreturn`},
		{`back\slash`, `back\\slash`},
	}
	for _, c := range cases {
		if got := cleanTerminalText(c.in); got != c.want {
			t.Errorf("cleanTerminalText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

package main

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
)

const indentUnit = "  "

// colorEnabled controls whether lipgloss styling is applied. It stays off by
// default so piped output and tests compare against plain text; main.go turns
// it on when stdout is an interactive terminal.
var colorEnabled bool

var (
	keyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	stringStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	numberStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	boolStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
	nullStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	bracketStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

// render applies a lipgloss style only when color output is enabled.
func render(style lipgloss.Style, text string) string {
	if !colorEnabled {
		return text
	}
	return style.Render(text)
}

// FormatTerminal renders a parsed JSON value for the terminal. Scalar fields
// render as "key = value"; a field whose value is an object expands into an
// indented block beneath its key.
func FormatTerminal(value any) string {
	if object, ok := value.(map[string]any); ok {
		if len(object) == 0 {
			return "(empty object)"
		}
		return formatObjectBlock(object, 0)
	}
	return formatValue(value, 0)
}

// formatObjectBlock renders each field of object at the given indent depth. A
// scalar or array field stays on the key's line ("key = value"); an object
// field prints its key as a header and expands its contents one level deeper.
func formatObjectBlock(object map[string]any, depth int) string {
	keys := sortedKeys(object)
	indent := strings.Repeat(indentUnit, depth)
	lines := make([]string, 0, len(keys))

	for _, key := range keys {
		renderedKey := render(keyStyle, cleanTerminalText(key))

		if nested, ok := object[key].(map[string]any); ok {
			lines = append(lines, indent+renderedKey)
			if len(nested) == 0 {
				lines = append(lines, strings.Repeat(indentUnit, depth+1)+render(nullStyle, "(empty object)"))
				continue
			}
			lines = append(lines, formatObjectBlock(nested, depth+1))
			continue
		}

		rendered := formatValue(object[key], depth)
		fieldLines := strings.Split(rendered, "\n")
		fieldLines[0] = indent + renderedKey + " = " + fieldLines[0]
		lines = append(lines, strings.Join(fieldLines, "\n"))
	}

	return strings.Join(lines, "\n")
}

func formatValue(value any, depth int) string {
	switch typed := value.(type) {
	case []any:
		return formatArray(typed, depth)
	case map[string]any:
		if len(typed) == 0 {
			return render(nullStyle, "(empty object)")
		}
		return "\n" + formatObjectBlock(typed, depth+1)
	default:
		return formatInline(value)
	}
}

func formatArray(array []any, depth int) string {
	if len(array) == 0 {
		return render(bracketStyle, "[]")
	}

	lines := []string{render(bracketStyle, "[")}
	for i, item := range array {
		var rendered string
		if object, isObject := item.(map[string]any); isObject {
			// Objects in arrays expand so each field lands on its own line.
			rendered = formatArrayObject(object, depth+1)
		} else {
			rendered = formatValue(item, depth+1)
		}
		itemLines := strings.Split(rendered, "\n")
		itemLines[0] = strings.Repeat(indentUnit, depth+1) + itemLines[0]
		if i < len(array)-1 {
			itemLines[len(itemLines)-1] += ","
		}
		lines = append(lines, itemLines...)
	}
	lines = append(lines, strings.Repeat(indentUnit, depth)+render(bracketStyle, "]"))
	return strings.Join(lines, "\n")
}

// formatArrayObject renders an object that lives inside an array. The first
// field shares the array's bullet indent; the rest align beneath it.
func formatArrayObject(object map[string]any, depth int) string {
	keys := sortedKeys(object)
	lines := make([]string, 0, len(keys))
	for i, key := range keys {
		renderedKey := render(keyStyle, cleanTerminalText(key))

		if nested, ok := object[key].(map[string]any); ok {
			header := renderedKey
			if i > 0 {
				header = strings.Repeat(indentUnit, depth) + header
			}
			lines = append(lines, header)
			if len(nested) == 0 {
				lines = append(lines, strings.Repeat(indentUnit, depth+1)+render(nullStyle, "(empty object)"))
				continue
			}
			lines = append(lines, formatObjectBlock(nested, depth+1))
			continue
		}

		rendered := formatValue(object[key], depth)
		fieldLines := strings.Split(rendered, "\n")
		fieldLines[0] = renderedKey + " = " + fieldLines[0]
		if i > 0 {
			fieldLines[0] = strings.Repeat(indentUnit, depth) + fieldLines[0]
		}
		lines = append(lines, fieldLines...)
	}
	return strings.Join(lines, "\n")
}

func formatInline(value any) string {
	switch value := value.(type) {
	case nil:
		return render(nullStyle, "null")
	case bool:
		return render(boolStyle, fmt.Sprint(value))
	case string:
		return render(stringStyle, cleanTerminalText(value))
	case map[string]any:
		keys := sortedKeys(value)
		items := make([]string, len(keys))
		for i, key := range keys {
			items[i] = render(keyStyle, cleanTerminalText(key)) + " = " + formatInline(value[key])
		}
		return strings.Join(items, "; ")
	default:
		return render(numberStyle, fmt.Sprint(value))
	}
}

func sortedKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func cleanTerminalText(value string) string {
	return strings.NewReplacer("\\", "\\\\", "\n", "\\n", "\r", "\\r", "\t", "\\t").Replace(value)
}

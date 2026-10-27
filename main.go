package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"charm.land/bubbles/v2/filepicker"
	tea "charm.land/bubbletea/v2"
)

var errFileSelectionCancelled = errors.New("file selection cancelled")

func main() {
	input, err := inputBytes(os.Args[1:])
	if err != nil {
		if errors.Is(err, errFileSelectionCancelled) {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	value, err := parser.Parse()
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse JSON:", err)
		os.Exit(1)
	}

	interactive := false
	if info, statErr := os.Stdout.Stat(); statErr == nil {
		interactive = info.Mode()&os.ModeCharDevice != 0
	}

	if interactive {
		if err := runViewer(value); err != nil {
			fmt.Fprintln(os.Stderr, "run viewer:", err)
			os.Exit(1)
		}
		return
	}

	if _, err := fmt.Fprintln(os.Stdout, FormatTerminal(value)); err != nil {
		fmt.Fprintln(os.Stderr, "write response:", err)
		os.Exit(1)
	}
}

func inputBytes(args []string) ([]byte, error) {
	if len(args) > 1 {
		return nil, fmt.Errorf("usage: %s [file.json]", filepath.Base(os.Args[0]))
	}
	if len(args) == 1 {
		return os.ReadFile(args[0])
	}

	info, err := os.Stdin.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect stdin: %w", err)
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		return input, nil
	}

	path, err := selectJSONFile()
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, errFileSelectionCancelled
	}
	return os.ReadFile(path)
}

type filePickerModel struct {
	picker    filepicker.Model
	selected  string
	cancelled bool
}

func (m filePickerModel) Init() tea.Cmd {
	return m.picker.Init()
}

func (m filePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		if key.String() == "esc" || key.String() == "ctrl+c" {
			m.cancelled = true
			return m, tea.Quit
		}
	}
	if selected, path := m.picker.DidSelectFile(msg); selected {
		m.selected = path
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	return m, cmd
}

func (m filePickerModel) View() tea.View {
	view := tea.NewView("Select a JSON file (enter to open, esc to cancel):\n\n" + m.picker.View())
	view.AltScreen = true
	return view
}

func selectJSONFile() (string, error) {
	currentDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current directory: %w", err)
	}

	picker := filepicker.New()
	picker.AllowedTypes = []string{".json"}
	picker.CurrentDirectory = currentDirectory
	picker.DirAllowed = true
	picker.FileAllowed = true
	picker.ShowSize = false

	finalModel, err := tea.NewProgram(filePickerModel{picker: picker}).Run()
	if err != nil {
		return "", fmt.Errorf("run file picker: %w", err)
	}
	result, ok := finalModel.(filePickerModel)
	if !ok {
		return "", fmt.Errorf("file picker returned an unexpected model")
	}
	if result.cancelled {
		return "", errFileSelectionCancelled
	}
	return result.selected, nil
}

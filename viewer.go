package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// node is one row in the navigable JSON tree. Container nodes (objects and
// arrays) own children and can be expanded or collapsed; leaf nodes render
// their scalar value inline.
type node struct {
	key        string // field name, or array index label like "[0]"
	value      any
	children   []*node
	container  bool
	expandable bool
	expanded   bool
	depth      int
}

// buildTree converts a parsed JSON value into the root list of nodes. The root
// itself is represented by its children so that a top-level object or array
// lists its entries directly.
func buildTree(value any) []*node {
	root := newNode("", value, 0)
	return root.children
}

func newNode(key string, value any, depth int) *node {
	n := &node{key: key, value: value, depth: depth}
	switch typed := value.(type) {
	case map[string]any:
		n.container = true
		n.expandable = len(typed) > 0
		for _, childKey := range sortedKeys(typed) {
			n.children = append(n.children, newNode(childKey, typed[childKey], depth+1))
		}
	case []any:
		n.container = true
		n.expandable = len(typed) > 0
		for i, item := range typed {
			n.children = append(n.children, newNode(fmt.Sprintf("[%d]", i), item, depth+1))
		}
	}
	return n
}

// visible walks the tree depth-first and returns the nodes that should be
// drawn given the current expansion state.
func visible(nodes []*node) []*node {
	var out []*node
	for _, n := range nodes {
		out = append(out, n)
		if n.container && n.expanded {
			out = append(out, visible(n.children)...)
		}
	}
	return out
}

var (
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("12"))
	markerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	countStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type viewerModel struct {
	roots  []*node
	cursor int
}

func newViewerModel(value any) viewerModel {
	return viewerModel{roots: buildTree(value)}
}

func (m viewerModel) Init() tea.Cmd { return nil }

func (m viewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	rows := visible(m.roots)
	switch key.String() {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(rows)-1 {
			m.cursor++
		}
	case "home", "g":
		m.cursor = 0
	case "end", "G":
		m.cursor = len(rows) - 1
	case "enter":
		// Enter toggles the container open or closed based on its current
		// expanded status.
		if len(rows) > 0 {
			current := rows[m.cursor]
			if current.expandable {
				current.expanded = !current.expanded
			}
		}
	case "right", "l":
		if len(rows) > 0 {
			current := rows[m.cursor]
			if current.expandable {
				current.expanded = true
			}
		}
	case "left", "h":
		if len(rows) > 0 {
			current := rows[m.cursor]
			if current.expandable && current.expanded {
				// Collapse the container the cursor is sitting on.
				current.expanded = false
			} else {
				// On a leaf or an already-collapsed node, jump to the parent
				// and collapse it in the same keypress so a single left always
				// closes the enclosing level.
				parent := parentIndex(rows, m.cursor)
				m.cursor = parent
				if rows[parent].expandable {
					rows[parent].expanded = false
				}
			}
		}
	}
	return m, nil
}

// parentIndex finds the nearest preceding row at a shallower depth.
func parentIndex(rows []*node, index int) int {
	if index <= 0 {
		return 0
	}
	depth := rows[index].depth
	for i := index - 1; i >= 0; i-- {
		if rows[i].depth < depth {
			return i
		}
	}
	return index
}

func (m viewerModel) View() tea.View {
	rows := visible(m.roots)
	var b strings.Builder
	b.WriteString("JSON viewer — ")
	b.WriteString(render(helpStyle, "↑/↓ move · enter toggle · → expand · ← collapse · q quit"))
	b.WriteString("\n\n")

	for i, n := range rows {
		line := renderRow(n)
		if i == m.cursor {
			line = render(cursorStyle, line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	view := tea.NewView(b.String())
	view.AltScreen = true
	return view
}

// renderRow builds the single-line representation of a node: indentation, an
// expand/collapse marker for containers, the key, and (for leaves) the value.
func renderRow(n *node) string {
	indent := strings.Repeat("  ", n.depth)
	label := render(keyStyle, cleanTerminalText(n.key))

	if n.container {
		marker := "  "
		if n.expandable {
			if n.expanded {
				marker = render(markerStyle, "▼ ")
			} else {
				marker = render(markerStyle, "▶ ")
			}
		}
		return indent + marker + label + " " + render(countStyle, containerSummary(n))
	}

	return indent + "  " + label + " = " + formatInline(n.value)
}

// containerSummary describes a container's type and child count, e.g.
// "{3}" for an object with three fields or "[2]" for a two-element array.
func containerSummary(n *node) string {
	switch n.value.(type) {
	case map[string]any:
		return fmt.Sprintf("{%d}", len(n.children))
	case []any:
		return fmt.Sprintf("[%d]", len(n.children))
	default:
		return ""
	}
}

// runViewer starts the interactive tree viewer for the parsed value.
func runViewer(value any) error {
	colorEnabled = true // Inside the TUI we always render styled output.
	_, err := tea.NewProgram(newViewerModel(value)).Run()
	return err
}

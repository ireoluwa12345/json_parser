package main

import (
	"encoding/json"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// keyMsg builds a KeyPressMsg for a single printable rune (e.g. 'q', 'j').
func keyMsg(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// specialKey builds a KeyPressMsg for a non-printable key by its code.
func specialKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

// send applies a key to a model and returns the updated viewerModel.
func send(t *testing.T, m viewerModel, msg tea.Msg) viewerModel {
	t.Helper()
	updated, _ := m.Update(msg)
	vm, ok := updated.(viewerModel)
	if !ok {
		t.Fatalf("Update returned %T, want viewerModel", updated)
	}
	return vm
}

func sampleValue() any {
	return map[string]any{
		"name": "Ada",
		"nested": map[string]any{
			"a": json.Number("1"),
			"b": json.Number("2"),
		},
		"list": []any{json.Number("1"), json.Number("2")},
	}
}

func TestBuildTree(t *testing.T) {
	roots := buildTree(map[string]any{"a": json.Number("1"), "b": "x"})
	if len(roots) != 2 {
		t.Fatalf("got %d roots, want 2 (sorted keys)", len(roots))
	}
	// Keys are sorted.
	if roots[0].key != "a" || roots[1].key != "b" {
		t.Errorf("roots keys = %q,%q want a,b", roots[0].key, roots[1].key)
	}
}

func TestBuildTreeScalarRoot(t *testing.T) {
	// A scalar root has no children.
	if roots := buildTree("just a string"); len(roots) != 0 {
		t.Errorf("scalar root children = %d, want 0", len(roots))
	}
}

func TestNewNodeObject(t *testing.T) {
	n := newNode("obj", map[string]any{"k": json.Number("1")}, 0)
	if !n.container {
		t.Error("object node should be a container")
	}
	if !n.expandable {
		t.Error("non-empty object should be expandable")
	}
	if len(n.children) != 1 {
		t.Errorf("children = %d, want 1", len(n.children))
	}
}

func TestNewNodeEmptyContainers(t *testing.T) {
	obj := newNode("o", map[string]any{}, 0)
	if !obj.container || obj.expandable {
		t.Error("empty object should be a non-expandable container")
	}
	arr := newNode("a", []any{}, 0)
	if !arr.container || arr.expandable {
		t.Error("empty array should be a non-expandable container")
	}
}

func TestNewNodeArray(t *testing.T) {
	n := newNode("arr", []any{json.Number("1"), json.Number("2")}, 0)
	if !n.container || !n.expandable {
		t.Error("non-empty array should be an expandable container")
	}
	if n.children[0].key != "[0]" || n.children[1].key != "[1]" {
		t.Errorf("array child keys = %q,%q want [0],[1]", n.children[0].key, n.children[1].key)
	}
}

func TestNewNodeScalar(t *testing.T) {
	n := newNode("x", json.Number("5"), 0)
	if n.container || n.expandable {
		t.Error("scalar node should not be a container")
	}
}

func TestVisible(t *testing.T) {
	roots := buildTree(sampleValue())
	// Collapsed: only top-level nodes (3 keys) are visible.
	if got := len(visible(roots)); got != 3 {
		t.Fatalf("collapsed visible = %d, want 3", got)
	}

	// Expand the "nested" object (sorted order: list, name, nested).
	for _, n := range roots {
		if n.key == "nested" {
			n.expanded = true
		}
	}
	// 3 top-level + 2 children of nested.
	if got := len(visible(roots)); got != 5 {
		t.Errorf("after expanding nested, visible = %d, want 5", got)
	}
}

func TestParentIndex(t *testing.T) {
	roots := buildTree(sampleValue())
	for _, n := range roots {
		if n.container {
			n.expanded = true
		}
	}
	rows := visible(roots)
	// Find a child node (depth 2) and confirm parentIndex points to a
	// shallower row.
	for i, n := range rows {
		if n.depth == 2 {
			parent := parentIndex(rows, i)
			if rows[parent].depth >= n.depth {
				t.Errorf("parentIndex(%d) depth %d not shallower than %d", i, rows[parent].depth, n.depth)
			}
			break
		}
	}
	// Index 0 and negative index clamp to 0.
	if parentIndex(rows, 0) != 0 {
		t.Error("parentIndex of 0 should be 0")
	}
	if parentIndex(rows, -5) != 0 {
		t.Error("parentIndex of negative should be 0")
	}
}

func TestNewViewerModel(t *testing.T) {
	m := newViewerModel(sampleValue())
	if len(m.roots) != 3 {
		t.Errorf("roots = %d, want 3", len(m.roots))
	}
	if m.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", m.cursor)
	}
}

func TestViewerInit(t *testing.T) {
	m := newViewerModel(sampleValue())
	if cmd := m.Init(); cmd != nil {
		t.Error("Init should return nil cmd")
	}
}

func TestUpdateIgnoresNonKeyMsg(t *testing.T) {
	m := newViewerModel(sampleValue())
	m2 := send(t, m, struct{}{})
	if m2.cursor != m.cursor {
		t.Error("non-key message should not move cursor")
	}
}

func TestUpdateNavigation(t *testing.T) {
	m := newViewerModel(sampleValue()) // 3 visible rows when collapsed

	// Down moves cursor forward.
	m = send(t, m, keyMsg('j'))
	if m.cursor != 1 {
		t.Errorf("after 'j' cursor = %d, want 1", m.cursor)
	}
	m = send(t, m, specialKey(tea.KeyDown))
	if m.cursor != 2 {
		t.Errorf("after down cursor = %d, want 2", m.cursor)
	}
	// Down at the bottom stays clamped.
	m = send(t, m, keyMsg('j'))
	if m.cursor != 2 {
		t.Errorf("down at bottom cursor = %d, want 2", m.cursor)
	}

	// Up moves back.
	m = send(t, m, keyMsg('k'))
	if m.cursor != 1 {
		t.Errorf("after 'k' cursor = %d, want 1", m.cursor)
	}
	m = send(t, m, specialKey(tea.KeyUp))
	if m.cursor != 0 {
		t.Errorf("after up cursor = %d, want 0", m.cursor)
	}
	// Up at the top stays clamped.
	m = send(t, m, keyMsg('k'))
	if m.cursor != 0 {
		t.Errorf("up at top cursor = %d, want 0", m.cursor)
	}

	// End / Home jumps.
	m = send(t, m, keyMsg('G'))
	if m.cursor != 2 {
		t.Errorf("after 'G' cursor = %d, want 2", m.cursor)
	}
	m = send(t, m, keyMsg('g'))
	if m.cursor != 0 {
		t.Errorf("after 'g' cursor = %d, want 0", m.cursor)
	}
	m = send(t, m, specialKey(tea.KeyEnd))
	if m.cursor != 2 {
		t.Errorf("after end cursor = %d, want 2", m.cursor)
	}
	m = send(t, m, specialKey(tea.KeyHome))
	if m.cursor != 0 {
		t.Errorf("after home cursor = %d, want 0", m.cursor)
	}
}

func TestUpdateExpandCollapse(t *testing.T) {
	m := newViewerModel(sampleValue())
	// Sorted roots: list[0], name[1], nested[2]. Move to "nested".
	m = send(t, m, keyMsg('G')) // cursor -> last ("nested")
	rows := visible(m.roots)
	nested := rows[m.cursor]
	if nested.key != "nested" {
		t.Fatalf("expected cursor on 'nested', got %q", nested.key)
	}

	// Enter toggles expansion open.
	m = send(t, m, specialKey(tea.KeyEnter))
	if !nested.expanded {
		t.Error("enter should expand the container")
	}
	// Enter again collapses.
	m = send(t, m, specialKey(tea.KeyEnter))
	if nested.expanded {
		t.Error("second enter should collapse the container")
	}

	// Right expands.
	m = send(t, m, specialKey(tea.KeyRight))
	if !nested.expanded {
		t.Error("right should expand the container")
	}
	// Left collapses the expanded container under the cursor.
	m = send(t, m, specialKey(tea.KeyLeft))
	if nested.expanded {
		t.Error("left should collapse the expanded container")
	}
}

func TestUpdateLeftJumpsToParent(t *testing.T) {
	m := newViewerModel(sampleValue())
	// Expand "nested" and move cursor onto one of its children.
	m = send(t, m, keyMsg('G')) // on "nested"
	m = send(t, m, specialKey(tea.KeyRight))
	m = send(t, m, specialKey(tea.KeyDown)) // onto first child of nested
	rows := visible(m.roots)
	if rows[m.cursor].depth != 2 {
		t.Fatalf("expected a child node at depth 2, got depth %d", rows[m.cursor].depth)
	}
	// Left on a leaf jumps to the parent and collapses it.
	m = send(t, m, specialKey(tea.KeyLeft))
	rows = visible(m.roots)
	if rows[m.cursor].key != "nested" {
		t.Errorf("after left on child, cursor key = %q, want nested", rows[m.cursor].key)
	}
	if rows[m.cursor].expanded {
		t.Error("parent should be collapsed after left from child")
	}
}

func TestUpdateQuit(t *testing.T) {
	m := newViewerModel(sampleValue())
	for _, r := range []rune{'q'} {
		if _, cmd := m.Update(keyMsg(r)); cmd == nil {
			t.Errorf("%q should return a quit command", r)
		}
	}
	if _, cmd := m.Update(specialKey(tea.KeyEscape)); cmd == nil {
		t.Error("esc should return a quit command")
	}
}

func TestContainerSummary(t *testing.T) {
	obj := newNode("o", map[string]any{"a": json.Number("1"), "b": json.Number("2")}, 0)
	if got := containerSummary(obj); got != "{2}" {
		t.Errorf("object summary = %q, want {2}", got)
	}
	arr := newNode("a", []any{json.Number("1")}, 0)
	if got := containerSummary(arr); got != "[1]" {
		t.Errorf("array summary = %q, want [1]", got)
	}
	leaf := newNode("x", "scalar", 0)
	if got := containerSummary(leaf); got != "" {
		t.Errorf("scalar summary = %q, want empty", got)
	}
}

func TestRenderRow(t *testing.T) {
	withColor(t, false)

	leaf := newNode("name", "Ada", 1)
	got := renderRow(leaf)
	if !strings.Contains(got, "name = Ada") {
		t.Errorf("leaf row = %q, want to contain 'name = Ada'", got)
	}

	collapsed := newNode("nested", map[string]any{"k": json.Number("1")}, 0)
	got = renderRow(collapsed)
	if !strings.Contains(got, "▶") {
		t.Errorf("collapsed container row = %q, want to contain collapse marker", got)
	}
	if !strings.Contains(got, "{1}") {
		t.Errorf("collapsed container row = %q, want to contain '{1}'", got)
	}

	collapsed.expanded = true
	got = renderRow(collapsed)
	if !strings.Contains(got, "▼") {
		t.Errorf("expanded container row = %q, want to contain expand marker", got)
	}

	empty := newNode("e", map[string]any{}, 0)
	got = renderRow(empty)
	if strings.Contains(got, "▶") || strings.Contains(got, "▼") {
		t.Errorf("empty container row = %q, should have no toggle marker", got)
	}
}

func TestViewerView(t *testing.T) {
	m := newViewerModel(sampleValue())
	view := m.View()
	// The view string should mention the help line and all top-level keys.
	s := view.Content
	for _, want := range []string{"JSON viewer", "list", "name", "nested"} {
		if !strings.Contains(s, want) {
			t.Errorf("view missing %q; got:\n%s", want, s)
		}
	}
}

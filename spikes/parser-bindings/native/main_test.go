package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

func parseFixture(
	t *testing.T,
	filename string,
	language *tree_sitter.Language,
) *tree_sitter.Tree {
	t.Helper()

	path := filepath.Join("..", "fixtures", filename)

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	parser := tree_sitter.NewParser()
	t.Cleanup(parser.Close)

	if err := parser.SetLanguage(language); err != nil {
		t.Fatalf("set language for %s: %v", filename, err)
	}

	tree := parser.Parse(source, nil)
	if tree == nil {
		t.Fatalf("parser returned no tree for %s", filename)
	}

	t.Cleanup(tree.Close)
	return tree
}

func countNodes(node *tree_sitter.Node, kind string) int {
	if node == nil {
		return 0
	}

	count := 0

	if node.Kind() == kind {
		count++
	}

	for index := uint(0); index < node.ChildCount(); index++ {
		child := node.Child(index)
		count += countNodes(child, kind)
	}

	return count
}

func findFirstNativeNode(
	node *tree_sitter.Node,
	kind string,
) (*tree_sitter.Node, bool) {
	if node == nil {
		return nil, false
	}

	if node.Kind() == kind {
		return node, true
	}

	for index := uint(0); index < node.ChildCount(); index++ {
		if found, ok := findFirstNativeNode(
			node.Child(index),
			kind,
		); ok {
			return found, true
		}
	}

	return nil, false
}

func TestJavaScriptFixture(t *testing.T) {
	language := tree_sitter.NewLanguage(
		tree_sitter_javascript.Language(),
	)

	tree := parseFixture(t, "basic.js", language)
	root := tree.RootNode()

	if root.HasError() {
		t.Fatal("valid JavaScript fixture contains parse errors")
	}

	if imports := countNodes(root, "import_statement"); imports < 1 {
		t.Fatalf("expected at least one import, got %d", imports)
	}

	if calls := countNodes(root, "call_expression"); calls < 3 {
		t.Fatalf("expected at least three calls, got %d", calls)
	}
}

func TestTypeScriptFixture(t *testing.T) {
	language := tree_sitter.NewLanguage(
		tree_sitter_typescript.LanguageTypescript(),
	)

	tree := parseFixture(t, "basic.ts", language)
	root := tree.RootNode()

	if root.HasError() {
		t.Fatal("valid TypeScript fixture contains parse errors")
	}

	if imports := countNodes(root, "import_statement"); imports < 2 {
		t.Fatalf("expected at least two imports, got %d", imports)
	}

	if calls := countNodes(root, "call_expression"); calls < 1 {
		t.Fatalf("expected at least one call, got %d", calls)
	}
}

func TestTSXFixture(t *testing.T) {
	language := tree_sitter.NewLanguage(
		tree_sitter_typescript.LanguageTSX(),
	)

	tree := parseFixture(t, "basic.tsx", language)
	root := tree.RootNode()

	if root.HasError() {
		t.Fatal("valid TSX fixture contains parse errors")
	}

	if imports := countNodes(root, "import_statement"); imports < 1 {
		t.Fatalf("expected at least one import, got %d", imports)
	}

	if elements := countNodes(root, "jsx_element"); elements < 1 {
		t.Fatalf("expected at least one JSX element, got %d", elements)
	}
}

func TestInvalidJavaScriptIsReported(t *testing.T) {
	language := tree_sitter.NewLanguage(
		tree_sitter_javascript.Language(),
	)

	tree := parseFixture(t, "invalid.js", language)

	if !tree.RootNode().HasError() {
		t.Fatal("invalid JavaScript was not reported as erroneous")
	}
}

func TestCallExpressionLocation(t *testing.T) {
	path := filepath.Join("..", "fixtures", "basic.js")

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	language := tree_sitter.NewLanguage(
		tree_sitter_javascript.Language(),
	)

	tree := parseFixture(t, "basic.js", language)

	node, found := findFirstNativeNode(
		tree.RootNode(),
		"call_expression",
	)
	if !found {
		t.Fatal("call_expression node not found")
	}

	startByte := node.StartByte()
	endByte := node.EndByte()
	startPoint := node.StartPosition()
	endPoint := node.EndPosition()

	if endByte <= startByte {
		t.Fatalf(
			"invalid byte range: start=%d end=%d",
			startByte,
			endByte,
		)
	}

	if endByte > uint(len(source)) {
		t.Fatalf(
			"node end byte %d exceeds source length %d",
			endByte,
			len(source),
		)
	}

	expectedRow := uint(
		bytes.Count(source[:startByte], []byte{'\n'}),
	)

	lineStart := bytes.LastIndex(
		source[:startByte],
		[]byte{'\n'},
	) + 1

	expectedColumn := startByte - uint(lineStart)

	if startPoint.Row != expectedRow {
		t.Errorf(
			"row mismatch: parser=%d calculated=%d",
			startPoint.Row,
			expectedRow,
		)
	}

	if startPoint.Column != expectedColumn {
		t.Errorf(
			"column mismatch: parser=%d calculated=%d",
			startPoint.Column,
			expectedColumn,
		)
	}

	nodeText := node.Utf8Text(source)

	if nodeText != `require("node:path")` {
		t.Errorf(
			"unexpected call expression text: %q",
			nodeText,
		)
	}

	t.Logf(
		"native call=%q bytes=%d-%d line=%d column=%d endLine=%d endColumn=%d",
		nodeText,
		startByte,
		endByte,
		startPoint.Row+1,
		startPoint.Column+1,
		endPoint.Row+1,
		endPoint.Column+1,
	)
}

func benchmarkNativeFixture(
	b *testing.B,
	filename string,
	language *tree_sitter.Language,
) {
	path := filepath.Join("..", "fixtures", filename)

	source, err := os.ReadFile(path)
	if err != nil {
		b.Fatalf("read %s: %v", path, err)
	}

	parser := tree_sitter.NewParser()
	defer parser.Close()

	if err := parser.SetLanguage(language); err != nil {
		b.Fatalf("set language for %s: %v", filename, err)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(source)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tree := parser.Parse(source, nil)
		if tree == nil {
			b.Fatal("parser returned nil tree")
		}

		// Native Tree-sitter allocates C resources, so each tree
		// must be closed during the measured lifecycle.
		tree.Close()
	}
}

func BenchmarkJavaScriptParse(b *testing.B) {
	benchmarkNativeFixture(
		b,
		"basic.js",
		tree_sitter.NewLanguage(
			tree_sitter_javascript.Language(),
		),
	)
}

func BenchmarkTypeScriptParse(b *testing.B) {
	benchmarkNativeFixture(
		b,
		"basic.ts",
		tree_sitter.NewLanguage(
			tree_sitter_typescript.LanguageTypescript(),
		),
	)
}

func BenchmarkTSXParse(b *testing.B) {
	benchmarkNativeFixture(
		b,
		"basic.tsx",
		tree_sitter.NewLanguage(
			tree_sitter_typescript.LanguageTSX(),
		),
	)
}

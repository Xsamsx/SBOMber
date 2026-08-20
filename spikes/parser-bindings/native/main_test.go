package main

import (
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

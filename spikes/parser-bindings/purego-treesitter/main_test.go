package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ts "github.com/dcosson/treesitter-go"
	"github.com/dcosson/treesitter-go/languages/javascript"
	"github.com/dcosson/treesitter-go/languages/tsx"
	"github.com/dcosson/treesitter-go/languages/typescript"
	tsparser "github.com/dcosson/treesitter-go/parser"
)

func parseTestFixture(
	t testing.TB,
	filename string,
	language *ts.Language,
) ([]byte, *ts.Tree) {
	t.Helper()

	path := filepath.Join("..", "fixtures", filename)

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}

	parser := tsparser.NewParser()
	parser.SetLanguage(language)

	tree := parser.ParseString(context.Background(), source)
	if tree == nil {
		t.Fatalf("parse %s: returned nil tree", filename)
	}

	return source, tree
}

func countNodeTypes(node ts.Node, counts map[string]int) {
	if node.IsNull() {
		return
	}

	counts[node.Type()]++

	for i := 0; i < int(node.ChildCount()); i++ {
		countNodeTypes(node.Child(i), counts)
	}
}

func findFirstNode(node ts.Node, nodeType string) (ts.Node, bool) {
	if node.IsNull() {
		return ts.Node{}, false
	}

	if node.Type() == nodeType {
		return node, true
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		if found, ok := findFirstNode(node.Child(i), nodeType); ok {
			return found, true
		}
	}

	return ts.Node{}, false
}

func findFirstNodeWithCursor(
	root ts.Node,
	nodeType string,
) (ts.Node, bool) {
	cursor := ts.NewTreeCursor(root)

	var walk func() (ts.Node, bool)

	walk = func() (ts.Node, bool) {
		current := cursor.CurrentNode()

		if current.Type() == nodeType {
			return current, true
		}

		if cursor.GotoFirstChild() {
			for {
				if found, ok := walk(); ok {
					return found, true
				}

				if !cursor.GotoNextSibling() {
					break
				}
			}

			cursor.GotoParent()
		}

		return ts.Node{}, false
	}

	return walk()
}

func assertNodeCount(
	t *testing.T,
	counts map[string]int,
	nodeType string,
	minimum int,
) {
	t.Helper()

	if counts[nodeType] < minimum {
		t.Errorf(
			"expected at least %d %q node, got %d",
			minimum,
			nodeType,
			counts[nodeType],
		)
	}
}

func TestJavaScriptASTNodes(t *testing.T) {
	_, tree := parseTestFixture(t, "basic.js", javascript.Language())
	root := tree.RootNode()

	if root.Type() != "program" {
		t.Fatalf("expected program root, got %q", root.Type())
	}

	if strings.Contains(root.String(), "(ERROR") {
		t.Fatal("valid JavaScript fixture contains an ERROR node")
	}

	counts := make(map[string]int)
	countNodeTypes(root, counts)

	assertNodeCount(t, counts, "import_statement", 1)
	assertNodeCount(t, counts, "call_expression", 1)

	t.Logf(
		"JavaScript AST: imports=%d calls=%d",
		counts["import_statement"],
		counts["call_expression"],
	)
}

func TestTypeScriptASTNodes(t *testing.T) {
	_, tree := parseTestFixture(t, "basic.ts", typescript.Language())
	root := tree.RootNode()

	if root.Type() != "program" {
		t.Fatalf("expected program root, got %q", root.Type())
	}

	if strings.Contains(root.String(), "(ERROR") {
		t.Fatal("valid TypeScript fixture contains an ERROR node")
	}

	counts := make(map[string]int)
	countNodeTypes(root, counts)

	assertNodeCount(t, counts, "import_statement", 1)
	assertNodeCount(t, counts, "call_expression", 1)

	t.Logf(
		"TypeScript AST: imports=%d calls=%d",
		counts["import_statement"],
		counts["call_expression"],
	)
}

func TestTSXASTNodes(t *testing.T) {
	_, tree := parseTestFixture(t, "basic.tsx", tsx.Language())
	root := tree.RootNode()

	if root.Type() != "program" {
		t.Fatalf("expected program root, got %q", root.Type())
	}

	if strings.Contains(root.String(), "(ERROR") {
		t.Fatal("valid TSX fixture contains an ERROR node")
	}

	counts := make(map[string]int)
	countNodeTypes(root, counts)

	assertNodeCount(t, counts, "import_statement", 1)
	assertNodeCount(t, counts, "jsx_element", 1)

	t.Logf(
		"TSX AST: imports=%d JSX elements=%d",
		counts["import_statement"],
		counts["jsx_element"],
	)
}

func TestInvalidJavaScriptContainsError(t *testing.T) {
	_, tree := parseTestFixture(t, "invalid.js", javascript.Language())
	treeText := tree.RootNode().String()

	hasError := strings.Contains(treeText, "(ERROR") ||
		strings.Contains(treeText, "(MISSING")

	if !hasError {
		t.Fatal("invalid JavaScript did not produce ERROR or MISSING node")
	}

	t.Log("invalid JavaScript correctly produced a parser error")
}

func TestCallExpressionLocation(t *testing.T) {
	source, tree := parseTestFixture(t, "basic.js", javascript.Language())

	node, found := findFirstNodeWithCursor(
		tree.RootNode(),
		"call_expression",
	)
	if !found {
		t.Fatal("call_expression node not found using TreeCursor")
	}

	startByte := node.StartByte()
	endByte := node.EndByte()
	startPoint := node.StartPoint()
	endPoint := node.EndPoint()

	if endByte <= startByte {
		t.Fatalf(
			"invalid byte range: start=%d end=%d",
			startByte,
			endByte,
		)
	}

	if int(endByte) > len(source) {
		t.Fatalf(
			"node end byte %d exceeds source length %d",
			endByte,
			len(source),
		)
	}

	expectedRow := uint32(
		bytes.Count(source[:startByte], []byte{'\n'}),
	)

	lineStart := bytes.LastIndex(
		source[:startByte],
		[]byte{'\n'},
	) + 1

	expectedColumn := uint32(int(startByte) - lineStart)

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

	nodeText := string(source[startByte:endByte])
	if strings.TrimSpace(nodeText) == "" {
		t.Fatal("call expression source text is empty")
	}

	t.Logf(
		"cursor call=%q bytes=%d-%d line=%d column=%d endLine=%d endColumn=%d",
		nodeText,
		startByte,
		endByte,
		startPoint.Row+1,
		startPoint.Column+1,
		endPoint.Row+1,
		endPoint.Column+1,
	)
}

var benchmarkTree *ts.Tree

func benchmarkFixture(
	b *testing.B,
	filename string,
	language *ts.Language,
) {
	source, _ := parseTestFixture(b, filename, language)

	parser := tsparser.NewParser()
	parser.SetLanguage(language)

	b.ReportAllocs()
	b.SetBytes(int64(len(source)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchmarkTree = parser.ParseString(context.Background(), source)

		if benchmarkTree == nil {
			b.Fatal("parser returned nil tree")
		}
	}
}

func BenchmarkJavaScriptParse(b *testing.B) {
	benchmarkFixture(b, "basic.js", javascript.Language())
}

func BenchmarkTypeScriptParse(b *testing.B) {
	benchmarkFixture(b, "basic.ts", typescript.Language())
}

func BenchmarkTSXParse(b *testing.B) {
	benchmarkFixture(b, "basic.tsx", tsx.Language())
}

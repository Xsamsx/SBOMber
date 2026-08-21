package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ts "github.com/dcosson/treesitter-go"
	"github.com/dcosson/treesitter-go/languages/javascript"
	"github.com/dcosson/treesitter-go/languages/tsx"
	"github.com/dcosson/treesitter-go/languages/typescript"
	tsparser "github.com/dcosson/treesitter-go/parser"
)

type fixture struct {
	filename string
	language *ts.Language
	valid    bool
}

func testFixture(ctx context.Context, item fixture) {
	path := filepath.Join("..", "fixtures", item.filename)

	source, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("%s: READ FAILED: %v\n", item.filename, err)
		return
	}

	parser := tsparser.NewParser()
	parser.SetLanguage(item.language)

	tree := parser.ParseString(ctx, source)
	if tree == nil {
		fmt.Printf("%s: PARSE FAILED: nil tree\n", item.filename)
		return
	}

	root := tree.RootNode()
	treeText := root.String()

	hasError := strings.Contains(treeText, "(ERROR") ||
		strings.Contains(treeText, "(MISSING")

	result := "PASS"
	if item.valid && hasError {
		result = "FAIL: valid source contains parser errors"
	}

	if !item.valid && !hasError {
		result = "FAIL: invalid source was accepted without an error node"
	}

	fmt.Printf(
		"%s: %s root=%s hasError=%t children=%d\n",
		item.filename,
		result,
		root.Type(),
		hasError,
		root.ChildCount(),
	)
}

func main() {
	ctx := context.Background()

	fixtures := []fixture{
		{
			filename: "basic.js",
			language: javascript.Language(),
			valid:    true,
		},
		{
			filename: "basic.ts",
			language: typescript.Language(),
			valid:    true,
		},
		{
			filename: "basic.tsx",
			language: tsx.Language(),
			valid:    true,
		},
		{
			filename: "invalid.js",
			language: javascript.Language(),
			valid:    false,
		},
	}

	for _, item := range fixtures {
		testFixture(ctx, item)
	}
}

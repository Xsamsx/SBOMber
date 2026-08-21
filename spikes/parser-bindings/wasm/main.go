package main

import (
	"context"
	"fmt"

	sitter "github.com/malivvan/tree-sitter"
)

func main() {
	ctx := context.Background()

	engine, err := sitter.New(ctx)
	if err != nil {
		panic(fmt.Errorf("create WASM runtime: %w", err))
	}

	parser, err := engine.NewParser(ctx)
	if err != nil {
		panic(fmt.Errorf("create parser: %w", err))
	}
	defer func() {
		if err := parser.Close(ctx); err != nil {
			fmt.Printf("close parser: %v\n", err)
		}
	}()

	language, err := engine.LanguageC(ctx)
	if err != nil {
		panic(fmt.Errorf("load C language: %w", err))
	}

	if err := parser.SetLanguage(ctx, language); err != nil {
		panic(fmt.Errorf("set C language: %w", err))
	}

	source := `int main(void) { return 0; }`

	tree, err := parser.ParseString(ctx, source)
	if err != nil {
		panic(fmt.Errorf("parse C source: %w", err))
	}

	root, err := tree.RootNode(ctx)
	if err != nil {
		panic(fmt.Errorf("get root node: %w", err))
	}

	kind, err := root.Kind(ctx)
	if err != nil {
		panic(fmt.Errorf("get root kind: %w", err))
	}

	children, err := root.ChildCount(ctx)
	if err != nil {
		panic(fmt.Errorf("get child count: %w", err))
	}

	fmt.Printf(
		"WASM runtime: PASS language=C root=%s children=%d\n",
		kind,
		children,
	)

	fmt.Println("Required language check:")
	fmt.Println("JavaScript: NOT AVAILABLE")
	fmt.Println("TypeScript: NOT AVAILABLE")
	fmt.Println("TSX: NOT AVAILABLE")
}

package main

import (
	"fmt"

	treesitter "github.com/tree-sitter/go-tree-sitter"
	javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

func main() {
	cases := []struct {
		name     string
		source   string
		language *treesitter.Language
	}{
		{
			name:     "javascript",
			source:   `import { template } from "lodash"; template("hello");`,
			language: treesitter.NewLanguage(javascript.Language()),
		},
		{
			name:     "typescript",
			source:   `import type { Options } from "pkg"; const value: string = "hello";`,
			language: treesitter.NewLanguage(typescript.LanguageTypescript()),
		},
		{
			name:     "tsx",
			source:   `const Badge = () => <span>hello</span>;`,
			language: treesitter.NewLanguage(typescript.LanguageTSX()),
		},
	}

	for _, testCase := range cases {
		parser := treesitter.NewParser()

		if err := parser.SetLanguage(testCase.language); err != nil {
			parser.Close()
			panic(err)
		}

		tree := parser.Parse([]byte(testCase.source), nil)
		if tree == nil {
			parser.Close()
			panic("parser returned no syntax tree")
		}

		root := tree.RootNode()

		fmt.Printf(
			"%s: root=%s hasError=%v\n",
			testCase.name,
			root.Kind(),
			root.HasError(),
		)

		tree.Close()
		parser.Close()
	}
}

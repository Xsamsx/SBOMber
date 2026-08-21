package main

import (
	"fmt"
	"os"
	"path/filepath"

	gofastparser "github.com/t14raptor/go-fast/parser"
)

func testFixture(filename string) {
	path := filepath.Join("..", "fixtures", filename)

	source, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("%s: READ FAILED: %v\n", filename, err)
		return
	}

	program, parseErr := gofastparser.Parse(string(source))
	if parseErr != nil {
		fmt.Printf("%s: PARSE FAILED: %v\n", filename, parseErr)
		return
	}

	if program == nil {
		fmt.Printf("%s: PARSE FAILED: parser returned nil AST\n", filename)
		return
	}

	fmt.Printf("%s: PASS\n", filename)
}

func main() {
	fixtures := []string{
		"basic.js",
		"basic.ts",
		"basic.tsx",
		"invalid.js",
	}

	for _, fixture := range fixtures {
		testFixture(fixture)
	}
}

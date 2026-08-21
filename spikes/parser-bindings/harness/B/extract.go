package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	gotreesitter "github.com/odvcencio/gotreesitter"
)

func resultLanguageForPath(path string) (string, error) {
	lowerPath := strings.ToLower(path)

	switch {
	case strings.HasSuffix(lowerPath, ".tsx"):
		return "tsx", nil
	case strings.HasSuffix(lowerPath, ".ts"):
		return "typescript", nil
	case strings.HasSuffix(lowerPath, ".js"),
		strings.HasSuffix(lowerPath, ".mjs"),
		strings.HasSuffix(lowerPath, ".cjs"):
		return "javascript", nil
	default:
		return "", fmt.Errorf(
			"unsupported fixture extension %q",
			filepath.Ext(path),
		)
	}
}

func captureNode(
	match gotreesitter.QueryMatch,
	name string,
) *gotreesitter.Node {
	for _, capture := range match.Captures {
		if capture.Name == name {
			return capture.Node
		}
	}

	return nil
}

func nodeLocation(node *gotreesitter.Node) (int, int) {
	point := node.StartPoint()

	return int(point.Row) + 1, int(point.Column)
}

func stringPointer(value string) *string {
	return &value
}

func unquoteJavaScriptString(value string) (string, error) {
	if len(value) < 2 {
		return "", fmt.Errorf("invalid JavaScript string %q", value)
	}

	unquoted, err := strconv.Unquote(value)
	if err != nil {
		return "", fmt.Errorf(
			"unquote JavaScript string %q: %w",
			value,
			err,
		)
	}

	return unquoted, nil
}

func collectNodesByType(
	node *gotreesitter.Node,
	language *gotreesitter.Language,
	nodeType string,
) []*gotreesitter.Node {
	if node == nil {
		return nil
	}

	nodes := make([]*gotreesitter.Node, 0)

	if node.Type(language) == nodeType {
		nodes = append(nodes, node)
	}

	for index := 0; index < node.ChildCount(); index++ {
		nodes = append(
			nodes,
			collectNodesByType(
				node.Child(index),
				language,
				nodeType,
			)...,
		)
	}

	return nodes
}

func nodeHasDirectChildType(
	node *gotreesitter.Node,
	language *gotreesitter.Language,
	nodeType string,
) bool {
	if node == nil {
		return false
	}

	for index := 0; index < node.ChildCount(); index++ {
		child := node.Child(index)
		if child != nil && child.Type(language) == nodeType {
			return true
		}
	}

	return false
}

func appendESMImports(
	result *Result,
	statement *gotreesitter.Node,
	sourceNode *gotreesitter.Node,
	language *gotreesitter.Language,
	source []byte,
) error {
	if statement == nil || sourceNode == nil {
		return fmt.Errorf("ESM import match is missing required captures")
	}

	specifier, err := unquoteJavaScriptString(sourceNode.Text(source))
	if err != nil {
		return err
	}

	statementTypeOnly := nodeHasDirectChildType(
		statement,
		language,
		"type",
	)

	importSpecifiers := collectNodesByType(
		statement,
		language,
		"import_specifier",
	)

	for _, importSpecifier := range importSpecifiers {
		nameNode := importSpecifier.ChildByFieldName("name", language)
		if nameNode == nil {
			return fmt.Errorf(
				"import specifier at bytes %d-%d has no name",
				importSpecifier.StartByte(),
				importSpecifier.EndByte(),
			)
		}

		aliasNode := importSpecifier.ChildByFieldName("alias", language)
		localNode := nameNode

		if aliasNode != nil {
			localNode = aliasNode
		}

		line, column := nodeLocation(localNode)

		result.Imports = append(
			result.Imports,
			Import{
				Specifier: specifier,
				Kind:      "esm_named",
				Local:     localNode.Text(source),
				Imported:  nameNode.Text(source),
				TypeOnly: statementTypeOnly ||
					nodeHasDirectChildType(
						importSpecifier,
						language,
						"type",
					),
				Line:   line,
				Column: column,
			},
		)
	}

	return nil
}

func appendDirectCall(
	result *Result,
	site *gotreesitter.Node,
	name *gotreesitter.Node,
	receiver *gotreesitter.Node,
	property *gotreesitter.Node,
	source []byte,
) error {
	if site == nil {
		return fmt.Errorf("call match has no call.site capture")
	}

	line, column := nodeLocation(site)

	call := Call{
		Line:   line,
		Column: column,
	}

	switch {
	case name != nil:
		call.Callee = stringPointer(name.Text(source))
	case receiver != nil && property != nil:
		call.Receiver = stringPointer(receiver.Text(source))
		call.Callee = stringPointer(property.Text(source))
	default:
		return fmt.Errorf(
			"call at bytes %d-%d has no supported callee",
			site.StartByte(),
			site.EndByte(),
		)
	}

	result.Calls = append(result.Calls, call)
	return nil
}

func appendIIFECall(
	result *Result,
	site *gotreesitter.Node,
) error {
	if site == nil {
		return fmt.Errorf("IIFE match has no call.iife_site capture")
	}

	line, column := nodeLocation(site)

	result.Calls = append(
		result.Calls,
		Call{
			Callee:   nil,
			Receiver: nil,
			Line:     line,
			Column:   column,
			Note:     "immediately-invoked result",
		},
	)

	return nil
}

func appendFunction(
	result *Result,
	declaration *gotreesitter.Node,
	name *gotreesitter.Node,
	language *gotreesitter.Language,
	source []byte,
) error {
	if declaration == nil || name == nil {
		return fmt.Errorf(
			"function match is missing required captures",
		)
	}

	line, column := nodeLocation(name)

	exported := false
	parent := declaration.Parent()

	if parent != nil &&
		parent.Type(language) == "export_statement" {
		exported = true
	}

	result.Functions = append(
		result.Functions,
		Function{
			Name:     name.Text(source),
			Line:     line,
			Column:   column,
			Exported: exported,
		},
	)

	return nil
}

func sortResult(result *Result) {
	sort.SliceStable(
		result.Imports,
		func(left, right int) bool {
			a := result.Imports[left]
			b := result.Imports[right]

			if a.Line != b.Line {
				return a.Line < b.Line
			}

			if a.Column != b.Column {
				return a.Column < b.Column
			}

			if a.Local != b.Local {
				return a.Local < b.Local
			}

			return a.Imported < b.Imported
		},
	)

	sort.SliceStable(
		result.Calls,
		func(left, right int) bool {
			a := result.Calls[left]
			b := result.Calls[right]

			if a.Line != b.Line {
				return a.Line < b.Line
			}

			if a.Column != b.Column {
				return a.Column < b.Column
			}

			// The ground-truth convention lists the inner named call
			// before the outer immediately-invoked result.
			if (a.Note == "") != (b.Note == "") {
				return a.Note == ""
			}

			aName := ""
			bName := ""

			if a.Callee != nil {
				aName = *a.Callee
			}

			if b.Callee != nil {
				bName = *b.Callee
			}

			return aName < bName
		},
	)

	sort.SliceStable(
		result.Functions,
		func(left, right int) bool {
			a := result.Functions[left]
			b := result.Functions[right]

			if a.Line != b.Line {
				return a.Line < b.Line
			}

			if a.Column != b.Column {
				return a.Column < b.Column
			}

			return a.Name < b.Name
		},
	)

	sort.SliceStable(
		result.Unresolved,
		func(left, right int) bool {
			a := result.Unresolved[left]
			b := result.Unresolved[right]

			if a.Line != b.Line {
				return a.Line < b.Line
			}

			if a.Column != b.Column {
				return a.Column < b.Column
			}

			return a.Expression < b.Expression
		},
	)
}

func extractFixture(fixturePath string) (Result, error) {
	resultLanguage, err := resultLanguageForPath(fixturePath)
	if err != nil {
		return Result{}, err
	}

	result := newResult(
		filepath.Base(fixturePath),
		resultLanguage,
	)

	source, err := os.ReadFile(fixturePath)
	if err != nil {
		return Result{}, fmt.Errorf("read fixture: %w", err)
	}

	language, err := languageForFixture(fixturePath)
	if err != nil {
		return Result{}, err
	}

	parser := gotreesitter.NewParser(language)
	if parser == nil {
		return Result{}, fmt.Errorf("parser creation returned nil")
	}

	tree, err := parser.Parse(source)
	if err != nil {
		return Result{}, fmt.Errorf("parse fixture: %w", err)
	}

	if tree == nil {
		return Result{}, fmt.Errorf("parser returned nil tree")
	}
	defer tree.Release()

	root := tree.RootNode()
	if root == nil {
		return Result{}, fmt.Errorf("tree returned nil root node")
	}

	result.HasError = root.HasError()

	querySource, err := os.ReadFile("queries/usage.scm")
	if err != nil {
		return Result{}, fmt.Errorf("read usage query: %w", err)
	}

	query, err := gotreesitter.NewQuery(
		string(querySource),
		language,
	)
	if err != nil {
		return Result{}, fmt.Errorf("compile usage query: %w", err)
	}

	cursor := query.Exec(root, language, source)
	if cursor == nil {
		return Result{}, fmt.Errorf("query returned nil cursor")
	}

	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}

		switch {
		case captureNode(match, "import.statement") != nil:
			err = appendESMImports(
				&result,
				captureNode(match, "import.statement"),
				captureNode(match, "import.source"),
				language,
				source,
			)

		case captureNode(match, "call.iife_site") != nil:
			err = appendIIFECall(
				&result,
				captureNode(match, "call.iife_site"),
			)

		case captureNode(match, "call.site") != nil:
			err = appendDirectCall(
				&result,
				captureNode(match, "call.site"),
				captureNode(match, "call.name"),
				captureNode(match, "call.receiver"),
				captureNode(match, "call.property"),
				source,
			)

		case captureNode(match, "function.decl") != nil:
			err = appendFunction(
				&result,
				captureNode(match, "function.decl"),
				captureNode(match, "function.name"),
				language,
				source,
			)
		}

		if err != nil {
			return Result{}, err
		}
	}

	sortResult(&result)
	return result, nil
}

func runExtract(fixturePath string) error {
	result, err := extractFixture(fixturePath)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("encode extraction result: %w", err)
	}

	return nil
}

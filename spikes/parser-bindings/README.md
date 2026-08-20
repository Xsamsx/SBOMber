# Parser Binding Spike

## Purpose

Component 2 needs to parse JavaScript, TypeScript and TSX source code to
identify third-party imports, function calls and source locations.

This spike compares parser options before the production parser is selected.

## Test environment

- Ubuntu 26.04 LTS
- Linux amd64
- Go 1.26.0
- GCC 15.2.0
- VMware virtual machine
- CGO available

## Required capabilities

A suitable parser must:

1. Parse JavaScript.
2. Parse TypeScript.
3. Parse TSX.
4. Identify import statements.
5. Identify call expressions.
6. Identify JSX elements.
7. Report invalid source without crashing.
8. Provide stable AST nodes and source locations.
9. Be practical to package with SBOMber.

## Candidate A: Native Tree-sitter with CGO

Dependencies:

- github.com/tree-sitter/go-tree-sitter v0.25.0
- github.com/tree-sitter/tree-sitter-javascript v0.25.0
- github.com/tree-sitter/tree-sitter-typescript v0.23.2

### Functional results

| Test | Result |
|---|---|
| JavaScript parsing | Pass |
| TypeScript parsing | Pass |
| TSX parsing | Pass |
| Import detection | Pass |
| Call-expression detection | Pass |
| JSX-element detection | Pass |
| Invalid-source detection | Pass |

### Packaging results

| Configuration | Result |
|---|---|
| CGO_ENABLED=1 | Pass |
| CGO_ENABLED=0 | Build fails |
| Linux amd64 native binary | Pass |
| Runtime dependency | Linux libc |

The no-CGO failure is expected. The official JavaScript and TypeScript Go
grammar bindings exclude their source files when CGO is disabled.

### Candidate A conclusion

Native Tree-sitter satisfies the Component 2 parsing requirements and is
currently the leading candidate. However, it requires CGO and affects how
Component 1 builds and distributes SBOMber binaries.

Final selection is pending comparison with the WASM and pure-Go options.

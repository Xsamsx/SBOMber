# T9 — Runtime Grammar Loading

## Result

Runtime grammar loading does not remove Candidate A's CGO requirement.

In version 0.25.0, `NewLanguage` accepts an `unsafe.Pointer` and converts it
to `*C.TSLanguage`. It only wraps a grammar pointer that has already been
obtained; it does not load a grammar library.

No built-in `dlopen`, `LoadLanguage`, `OpenLibrary`, or pure-Go loading
implementation was found.

The binding embeds the Tree-sitter C runtime through `lib.c`, imports `C`,
and calls functions such as `C.ts_parser_new` and
`C.ts_parser_parse_with_options`.

A binding-only program compiled with `CGO_ENABLED=0` failed because
`tree_sitter.NewParser` was unavailable.

## Decision relevance

Moving grammars into runtime-loaded shared libraries would not make the
official binding pure Go. It would still require the CGO-based parser runtime
and would add separate platform-specific grammar libraries to deployment.

Candidate D therefore does not improve packaging over Candidate A and remains
eliminated.

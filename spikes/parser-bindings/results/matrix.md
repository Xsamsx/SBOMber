# Parser Candidate Comparison Matrix

## Candidate status

| Candidate | Description | Final status |
|---|---|---|
| A | Official `go-tree-sitter` binding with CGO | Selected |
| B | `odvcencio/gotreesitter` pure-Go runtime | Fallback |
| C | `dcosson/treesitter-go` | Scope-excluded; earlier location anomaly was not reproduced under this protocol |
| D | Official binding with runtime-loaded grammar libraries | Eliminated; does not remove CGO and increases deployment complexity |
| E | `malivvan/tree-sitter` WASM wrapper | Eliminated at T1; required JS/TS/TSX grammars unavailable |
| F | `t14raptor/go-fast` | Eliminated at T1; no ESM source-type mode |

## Gate A — hard requirements

Candidates A and B passed every hard requirement.

| Gate | Requirement | Candidate A | Candidate B |
|---|---|---:|---:|
| A1 | JavaScript with ESM import/export | Pass | Pass |
| A2 | TypeScript | Pass | Pass |
| A3 | TSX | Pass | Pass |
| A4 | Named, default and aliased imports | Pass | Pass |
| A5 | Namespace imports | Pass | Pass |
| A6 | Destructured CommonJS | Pass | Pass |
| A7 | Package subpaths | Pass | Pass |
| A8 | Type-only imports distinguishable | Pass | Pass |
| A9 | Static and computed dynamic imports | Pass | Pass |
| A10 | Member-expression calls | Pass | Pass |
| A11 | Exact byte and source locations | Pass | Pass |
| A12 | Named functions and call sites | Pass | Pass |
| A13 | Invalid input reports an error without panic | Pass | Pass |
| A14 | Source is parsed without execution | Pass | Pass |

Evidence: `results/B/T1.txt`, `results/B/T2-RESULT.md`,
`results/parity/T2b-RESULT.md`, `results/parity/T3-RESULT.md`, and
`results/parity/T4/T4-RESULT.md`.

## Correctness and robustness

| Measure | Candidate A | Candidate B |
|---|---|---|
| Valid formal-corpus tree parity | 12/12 equivalent | 12/12 equivalent |
| Invalid-source recovery tree | `program` containing `ERROR` | `ERROR` root with different recovery subtree |
| T3 traversal and locations | Exact | Exact |
| Candidate B extraction ground truth | Not separately implemented | 13/13 exact |
| Valid Lodash 4.17.21 bundle | Parsed without error | Incorrectly reported extensive `ERROR` nodes |
| Invalid, empty, deep and binary safety | No panic or timeout | No panic or timeout |

## Gate B — measured results

| Measure | Candidate A | Candidate B |
|---|---:|---:|
| B1 full-corpus throughput | Not run | Not run |
| B2 full-corpus peak RSS | Not run | Not run |
| B3 stripped JS/TS/TSX binary | Approximately 5.3 MB | 9,277,602 bytes |
| B4 packaging | Linux/amd64 CGO build demonstrated | Linux, macOS and Windows; amd64 and arm64 |
| B5 process start-up cost | Not separately measured | Not separately measured |
| B6 long-run memory stability | Not run | Not run |
| Lodash matched workload | 0.21 s, 15,052 KB | 0.91 s, 74,648 KB |
| Deep nesting matched workload | 0.10 s, 13,980 KB | 0.21 s, 51,036 KB |
| Random binary matched workload | 3.01 s, 253,168 KB | 0.11 s, 17,688 KB |

The hostile-input measurements are not a substitute for T6. Different binary
error-recovery tree shapes also make the binary timing comparison unsuitable
as a general performance ranking.

## Gate C — project fit

| Criterion | Candidate A | Candidate B | Favours |
|---|---|---|---|
| C1 maturity | Official organisation; 174 importers | Young project; 26 importers; mostly one maintainer | A |
| C2 API stability | Pre-1.0 and low activity | Pre-1.0 with rapid releases | A |
| C3 caller obligations | Explicit `Close` discipline for C allocations | No CGO allocation discipline, but T5 was not run | B |
| C4 packaging | CGO, native runners and glibc floor | Static CGO-free cross-builds | B |
| C5 licence | MIT | MIT | Tie |
| C6 query support | Available | Available; shared query compiled for JS/TS/TSX | Tie |

## Tests not performed

- T5 long-run resource-safety test;
- T6 full-corpus throughput test;
- T8 GitHub Actions CI matrix;
- Candidate C protocol-level location reproduction.

T6 was unnecessary under the fixed decision rule because Gate C resolved the
decision before throughput. T8 is deferred to S5-03.

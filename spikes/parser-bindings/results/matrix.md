# Parser Candidate Comparison Matrix

## Candidate status

| Candidate | Description | Final status |
|---|---|---|
| A | Official `go-tree-sitter` binding with CGO | Selected |
| B | `odvcencio/gotreesitter` pure-Go runtime | Fallback |
| C | `dcosson/treesitter-go` | Scope-excluded; supplemental smoke passed, but full protocol not run |
| D | Official binding with runtime-loaded grammar libraries | Eliminated; does not remove CGO |
| E | `malivvan/tree-sitter` WASM wrapper | Eliminated at T1; JS/TS/TSX unavailable |
| F | `t14raptor/go-fast` | Eliminated at T1; ESM fixtures failed |

## Gate A — hard requirements

| Gates | Candidate A | Candidate B |
|---|---|---|
| A1–A3 languages | Pass — measured | Pass — measured |
| A4–A10 extraction constructs | Pass — inferred from valid-fixture structural parity; no A semantic adapter | Pass — measured |
| A11 exact locations | Pass — measured | Pass — measured |
| A12 named functions/calls | Pass — inferred from structural parity | Pass — measured |
| A13 invalid-input safety | Pass — measured | Pass — measured |
| A14 no source execution | Pass — parser design and harness inspection | Pass — parser design and harness inspection |

Candidate B matched all 13 expected JSON fixtures exactly.

Candidate A's extraction results are inferred from equivalent named-node
structures on the 12 valid fixtures. T2b removed Candidate A's field labels,
so field-assignment parity was not independently measured. Candidate A's
production adapter must verify every field used by `queries/usage.scm`.

## Correctness and robustness

| Measure | Candidate A | Candidate B |
|---|---|---|
| Valid formal-corpus tree parity | 12/12 equivalent | 12/12 equivalent |
| Invalid recovery tree | `program` containing `ERROR` | Different `ERROR` root |
| T3 traversal and locations | Exact | Exact |
| Semantic extraction ground truth | Not separately implemented | 13/13 exact |
| Valid Lodash 4.17.21 bundle | Accepted without errors | Failed to accept; `no_stacks_alive` |
| Invalid, empty, deep and binary safety | No panic or timeout | No panic or timeout |

The Lodash failure was not classified by `ParseStrict` as an early safety
stop. Raising the node budget did not change it, and documented stack override
attempts did not produce a successful parse. It is recorded as an observed
parser/recovery incompatibility with unresolved internal cause.

## Gate B — measured results

| Measure | Candidate A | Candidate B |
|---|---:|---:|
| B1 full-corpus throughput | Not run | Not run |
| B2 full-corpus peak RSS | Not run | Not run |
| B3 selective prototype binary | About 5.3 MiB | About 8.85 MiB |
| B4 packaging | Linux/amd64 CGO build demonstrated | Linux, macOS and Windows; amd64 and arm64 |
| B5 process start-up cost | Not separately measured | Not separately measured |
| B6 long-run memory stability | Pass with correct `Close` discipline | Not run |
| Lodash matched workload | 0.21 s, 15,052 KB | 0.91 s, 74,648 KB |
| Deep nesting matched workload | 0.10 s, 13,980 KB | 0.21 s, 51,036 KB |
| Random binary matched workload | 3.01 s, 253,168 KB | 0.11 s, 17,688 KB |

The binary sizes are not like-for-like. Candidate A implements s-expression
output only. Candidate B includes multiple commands and embedded grammar data.

The hostile-input measurements are not a substitute for T6. Different binary
error-recovery shapes also make binary timing unsuitable as a general
performance ranking.

## T5 — Candidate A resource discipline

| Mode | Initial RSS | Final RSS | Change |
|---|---:|---:|---:|
| Correct `Close` | 5,332 KB | 5,916 KB | +584 KB |
| 5,000 retained unclosed trees | 5,304 KB | 70,648 KB | +65,344 KB |

The correct-close run reached a plateau. The deliberate-leak control grew
consistently, proving that deterministic closure is a production requirement.

## Gate C — project fit

| Criterion | Candidate A | Candidate B | Favours |
|---|---|---|---|
| C1 runtime maturity | Official C runtime with broad production exposure | Young from-scratch Go reimplementation | A |
| C2 API/runtime risk | Thin pinned binding over mature runtime | Pre-1.0, rapid releases, single-maintainer concentration | A |
| C3 caller obligations | Explicit `Close` discipline, measured by T5 | No CGO closure obligation | B |
| C4 packaging | CGO, native runners and glibc floor | Static CGO-free cross-builds | B |
| C5 licence | MIT | MIT | Tie |
| C6 query support | Available; production fields still to verify | Available and measured by extraction adapter | B |

## Tests not performed

- T6 full-corpus throughput test;
- T8 GitHub Actions CI matrix;
- complete fixed-protocol run for Candidate C.

T6 was unnecessary because Gate C resolved the decision before throughput.
T8 is deferred to S5-03.

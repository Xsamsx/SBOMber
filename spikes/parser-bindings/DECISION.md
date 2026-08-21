# Parser Binding Decision

## Question

Which Go Tree-sitter binding or runtime should Component 2 use to parse
JavaScript, TypeScript and TSX and produce reliable usage and reachability
evidence for SBOMber?

## Candidates

| ID | Candidate | Outcome |
|---|---|---|
| A | Official `github.com/tree-sitter/go-tree-sitter` binding with grammar bindings | Selected |
| B | `github.com/odvcencio/gotreesitter` pure-Go runtime | Fallback |
| C | `github.com/dcosson/treesitter-go` | Scope-excluded |
| D | Official binding with grammars loaded from shared libraries using `purego` | Eliminated |
| E | `github.com/malivvan/tree-sitter` WASM wrapper | Eliminated at T1 |
| F | `github.com/t14raptor/go-fast` | Eliminated at T1 |

Candidate C entered the protocol with an earlier unresolved location
anomaly involving direct-child traversal. A later supplemental test passed
using tree-cursor traversal, which was not the route associated with the
original anomaly. Direct-child behaviour was not re-tested, so the anomaly is
neither reproduced nor refuted.

Candidate C remains scope-excluded because it was not run through the complete
fixed corpus and shared comparison protocol. Candidate B was the protocol's
designated strongest no-CGO candidate.

Candidate D was eliminated because runtime grammar loading does not remove the
official binding's CGO dependency. It also requires separate
platform-specific grammar libraries.

Candidate E lacked the required JavaScript, TypeScript and TSX grammar path.
Candidate F lacked an ESM source-type mode.

## Gates fixed before measurement

### Gate A — hard requirements

A surviving parser must:

1. parse JavaScript with ESM imports and exports;
2. parse TypeScript;
3. parse TSX;
4. distinguish named, default and aliased imports;
5. resolve namespace imports as bindings;
6. recognise destructured CommonJS imports;
7. preserve package subpaths;
8. distinguish TypeScript type-only imports;
9. recognise static and computed dynamic imports;
10. identify member-expression calls and property names;
11. provide exact byte and row/column locations;
12. identify named declarations and their call sites;
13. report invalid input without panic; and
14. parse without executing source code.

Candidate B directly passed all 13 extraction fixtures.

For Candidate A, language availability, exact locations, invalid-input
handling and non-execution were measured directly. Extraction-related gates
A4–A10 and A12 are recorded as passes inferred from T2b structural parity;
a separate Candidate A semantic extraction adapter was not built.

T2b removed Candidate A field labels during normalisation, so it proves
named-node type and nesting parity but not independent field-assignment
parity. The production Candidate A adapter must verify every field used by
`queries/usage.scm`.

### Gate B — measured, non-eliminating

Gate B records throughput, peak memory, binary size, packaging, start-up cost,
and long-run memory stability.

T5 was run after Candidate A was selected. With deterministic `Close`
calls, RSS plateaued during 5,000 parses. A deliberate no-`Close` control grew
by 65,344 KB.

T6 full-corpus throughput was not run. Under the fixed rule, throughput is
used only if correctness, risk and packaging all tie, so T6 was not required
to resolve this decision.

### Gate C — project fit

Gate C evaluates maturity, API stability, caller resource obligations,
packaging consequences, licence compatibility and query API availability.

Both surviving candidates use MIT-licensed dependencies and provide the query
and traversal features needed by the extraction design.

## Results

| Area | Candidate A | Candidate B |
|---|---|---|
| Required languages | Pass — measured | Pass — measured |
| A4–A10 and A12 | Pass — inferred from valid-fixture structural parity; no A semantic adapter | Pass — measured through 13/13 exact extraction |
| A11 exact locations | Pass — measured | Pass — measured |
| A13 invalid-input safety | Pass — measured | Pass — measured |
| Valid formal-corpus tree parity | Equivalent on all 12 valid fixtures | Equivalent on all 12 valid fixtures |
| Field-assignment parity | Not independently compared after field-label removal | Extraction captures measured |
| Invalid recovery | `program` containing `ERROR` | Different `ERROR` root |
| Lodash bundle | Accepted without errors | Failed to accept; `no_stacks_alive` |
| T5 resource stability | Pass with `Close`; deliberate leak confirmed | Not run |
| Hostile-input termination | Pass | Pass |
| Selective binary | About 5.3 MiB | About 8.85 MiB |
| CGO-free build | No | Yes |
| Cross-compilation | Requires native C toolchains | Standard Go cross-compilation |
| Downstream importers | 174 | 26 |
| Recent binding/runtime activity | Binding: 0 commits in measured 90 days | 2,428 commits in measured 90 days |
| Runtime maturity | Official Tree-sitter C runtime with broad production exposure | Young from-scratch Go runtime reimplementation |
| Maintenance concentration | Official organisation; thin binding has low activity | Approximately 98% from primary maintainer |
| Licence | MIT | MIT |

The binary comparison is approximate rather than like-for-like. Candidate A's
harness implements only s-expression output, while Candidate B's selective
binary includes multiple diagnostic/extraction commands and embedded grammar
data.

Detailed evidence is recorded in `results/matrix.md` and the T1–T10 result
files.

## Decision rule applied

### Step 1 — hard-gate elimination

Candidates A and B passed every Gate A requirement and remained in the
comparison.

Candidate D did not improve the CGO packaging model and was eliminated.
Candidates E and F had already failed required language or source-mode
availability at T1.

Candidate C was not advanced through the complete fixed protocol. Its later
supplemental location test passed using tree-cursor traversal. Direct-child
traversal was not re-tested, so the earlier anomaly remains unresolved.
Candidate B remained the designated strongest no-CGO survivor.

### Step 2 — number of survivors

More than one candidate remained, so the decision continued to correctness.

### Step 3 — correctness

Candidates A and B produced equivalent named-node structures on all 12 valid
formal fixtures. Both passed the exact traversal and location tests.

Candidate B's extraction adapter matched all 13 committed expected JSON files
exactly. Candidate A did not have a separate semantic extraction adapter.
Candidate A's extraction-related Gate A results are therefore inferred from
the valid-fixture structural parity rather than directly measured.

T2b removed Candidate A field labels because Candidate B's s-expression output
did not contain them. T2b therefore did not independently compare field
assignments such as `function:`, `source:` and `object:`. This must be verified
while implementing Candidate A's production adapter.

The formal fixture evidence remains a correctness tie.

T4 found that Candidate B failed to accept the valid pinned Lodash 4.17.21
bundle. Follow-up diagnostics reported `ParseStoppedEarly() == false` and
`no_stacks_alive`, rather than a strict early-stop safety reason.

At termination, the quantified safety budgets were far below their limits:
iterations were at 2.0%, nodes at 3.0%, arena memory at 3.9%, and parse depth
at 0.03%. Increasing the node budget did not change the result. The measured
iteration, node, arena and depth limits were therefore not implicated.

Attempts to set `GOT_GLR_MAX_STACKS` to 16, 32 and 64 did not produce a
successful parse. Each override produced the same worse outcome: 26,593
tokens and a root ending at byte 45,497, compared with 41,455 tokens and byte
73,003 by default. The diagnostic still reported `maxStacks=8`, so the
effective stack-limit behaviour remains unresolved.

Candidate B exhausted its viable parse stacks and failed to accept input that
Candidate A accepted. The finding is recorded as an observed parser/recovery
incompatibility. It is not described as a proven grammar defect because the
exact internal cause remains unresolved. It is considered under Gate C and
does not change the formal correctness tie.

### Step 4 — Gate C risk

Gate C resolves the decision in favour of Candidate A.

The strongest argument for Candidate A is the runtime beneath the Go binding:
the official Tree-sitter C implementation has years of production exposure
through editors, code-intelligence tools and language tooling. The Go binding
is comparatively thin. Low activity in that binding is less concerning when
SBOMber pins its parser and grammar versions.

Candidate B is actively maintained, has a much larger Go test suite, frequent
releases, a detailed changelog and responsive issue handling. Those are real
strengths.

However, Candidate B is a young, pre-1.0, from-scratch reimplementation of the
Tree-sitter runtime. Its C-parity is established primarily by its own test
suite, approximately 98% of recent commits are associated with its primary
maintainer, and the Lodash result demonstrates a real compatibility boundary
that was absent from Candidate A.

For a component whose output must provide honest and reproducible evidence,
mature parser-runtime behaviour is more important than activity in a thin
binding repository. Candidate A therefore has lower unfixable upstream risk
during the remaining project period.

Candidate A still carries CGO, native resource cleanup and packaging risks.
T5 showed that these are controllable when `Close` discipline is enforced.

### Step 5 — packaging

The decision was already resolved by Gate C, but packaging was still
recorded.

Candidate B clearly wins packaging. It supports static CGO-free builds using
the standard Go cross-compilation toolchain. Its confirmed JavaScript,
TypeScript and TSX selective build is about 8.85 MiB and passed all 13
extraction fixtures.

Candidate A requires CGO. The tested Linux binary is dynamically linked, about
5.3 MiB, and requires glibc 2.34 or later. Linux and macOS artifacts must be
built on suitable native runners.

The size comparison is approximate rather than like-for-like: Candidate A's
harness implements s-expression output only, while Candidate B's binary
contains multiple commands and embedded grammar data.

R8 makes one operating system a must-have and cross-platform binaries a
should-have. The fixed rule therefore does not allow Candidate B's packaging
advantage to override Candidate A's lower runtime risk.

### Step 6 — throughput

Throughput was not used. The fixed rule permits it only if correctness, risk
and packaging all tie, and they did not.

## Selection

Select Candidate A with these exact pinned modules:

- `github.com/tree-sitter/go-tree-sitter` v0.25.0;
- `github.com/tree-sitter/tree-sitter-javascript` v0.25.0;
- `github.com/tree-sitter/tree-sitter-typescript` v0.23.2.

`Parser.SetLanguage` returned nil during the T5 test, demonstrating that the
pinned parser and JavaScript grammar ABI versions are compatible. Existing
TypeScript and TSX harness tests also passed with the pinned TypeScript grammar
module.

Component 2 must now port the proven extraction contract and shared query
behaviour to the Candidate A production adapter and directly verify every
field assignment used by `queries/usage.scm`.

## Fallback and trigger

Candidate B, `github.com/odvcencio/gotreesitter` v0.51.0, is the fallback.

Switch to Candidate B if an unfixable Candidate A grammar or location defect
blocks the production usage-graph milestone, or if Component 1 cannot produce
the required supported Linux artifact because of Candidate A's CGO and native
runtime requirements.

The shared fixtures, expected JSON schema and query contract remain fixed.
Only the parser adapter and packaging configuration should change.

Before switching, repeat the valid-Lodash regression against the Candidate B
version being considered. Do not switch while the observed valid-source
parser/recovery incompatibility remains relevant to the production scan
scope.

## Consequences for Component 1

Component 1 must plan for:

- `CGO_ENABLED=1`;
- `go-tree-sitter` v0.25.0;
- `tree-sitter-javascript` v0.25.0;
- `tree-sitter-typescript` v0.23.2;
- native build runners for supported operating systems;
- a dynamically linked Linux artifact;
- glibc 2.34 as the tested Linux compatibility floor;
- an approximately 5.3 MiB stripped prototype binary;
- no assumption that ordinary `CGO_ENABLED=0` cross-compilation will work.

The prototype size is approximate because the Candidate A and Candidate B
harnesses do not contain identical commands or embedded data.

Linux is the must-have platform. Additional platform artifacts remain
should-have work and should be built and tested on matching runners.

T8 was not run during this spike and is deferred to S5-03.

## Known limitations

1. Candidate A requires CGO and native platform toolchains.
2. The tested Linux artifact requires glibc 2.34 or later.
3. Objects allocating C memory must be closed deterministically, including
   parsers, trees, tree cursors, queries, query cursors and lookahead
   iterators.
4. T5 proved this obligation: correct closure plateaued, while retaining 5,000
   unclosed trees increased RSS by 65,344 KB.
5. Candidate A and Candidate B produce different recovery trees for invalid
   JavaScript.
6. Candidate B failed to accept the pinned valid Lodash 4.17.21 bundle with
   stop reason `no_stacks_alive`. Its measured iteration, node, arena and depth
   budgets were below 4% utilisation, but the exact stack-exhaustion cause
   remains unresolved.
7. Minified and generated files are normally excluded, but filename-based
   exclusions may be incomplete.
8. The maximum individual source-file size is 1,000,000 bytes.
9. The per-file processing timeout is 5 seconds.
10. Files exceeding either limit must be reported as skipped or unresolved,
    never silently omitted.
11. T2b removed Candidate A field labels and therefore did not independently
    compare field assignments.
12. T6 full-corpus throughput was not measured because it could not change the
    decision under the fixed rule.
13. T8 GitHub Actions validation was not run and is deferred to S5-03.
14. Candidate C passed a supplemental tree-cursor location test, but its
    earlier direct-child traversal anomaly was not re-tested and it was not
    run through the complete fixed protocol.
15. The selected Candidate A semantic extraction adapter still needs to be
    completed and must verify the shared query's field captures directly.

## Benchmark caveat

Measurements were collected inside a VMware Workstation guest with 4 vCPUs,
4 GB allocated RAM and a 50 GB virtual disk.

Absolute timings must not be compared with published measurements from other
machines. Only comparisons measured on this VM under matched workloads are
meaningful.

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

Candidate C entered the protocol with an unresolved location anomaly. Its
protocol-level reproduction was optional and was not completed. It is
therefore recorded as scope-excluded, and this decision does not claim that a
new Candidate C defect was proven.

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

Candidates A and B passed all 14 hard requirements.

### Gate B — measured, non-eliminating

Gate B records throughput, peak memory, binary size, packaging, start-up cost,
and long-run memory stability.

T5 long-run stability and T6 full-corpus throughput were not run. Under the
fixed rule, throughput is used only if correctness, risk and packaging all
tie, so T6 was not required to resolve this decision.

### Gate C — project fit

Gate C evaluates maturity, API stability, caller resource obligations,
packaging consequences, licence compatibility and query API availability.

Both surviving candidates use MIT-licensed dependencies and provide the query
and traversal features needed by the extraction design.

## Results

| Area | Candidate A | Candidate B |
|---|---|---|
| Required languages | Pass | Pass |
| Gate A | 14/14 pass | 14/14 pass |
| Valid formal-corpus tree parity | Equivalent on all 12 valid fixtures | Equivalent on all 12 valid fixtures |
| Invalid recovery | `program` containing `ERROR` | Different `ERROR` root |
| Exact locations | Pass | Pass |
| Extraction ground truth | No separate semantic adapter | 13/13 exact |
| Valid Lodash bundle | Parsed without error | Incorrectly produced extensive error nodes |
| Hostile-input termination | Pass | Pass |
| Selective binary | Approximately 5.3 MB | 9,277,602 bytes |
| CGO-free build | No | Yes |
| Cross-compilation | Requires native C toolchains | Standard Go cross-compilation |
| Downstream importers | 174 | 26 |
| Recent activity | 0 commits in measured 90 days | 2,428 commits in measured 90 days |
| Maintenance concentration | Official organisation, but low activity | Approximately 98% from primary maintainer |
| Licence | MIT | MIT |

Detailed evidence is recorded in `results/matrix.md` and the T1–T10 result
files.

## Decision rule applied

### Step 1 — hard-gate elimination

Candidates A and B passed every Gate A requirement and remained in the
comparison.

Candidate D did not improve the CGO packaging model and was eliminated.
Candidates E and F had already failed required language or source-mode
availability at T1.

Candidate C was not advanced because its earlier location anomaly remained
unresolved and the optional protocol-level reproduction was not performed.
No unsupported defect claim is made against Candidate C.

### Step 2 — number of survivors

More than one candidate remained, so the decision continued to correctness.

### Step 3 — correctness

Candidates A and B produced equivalent named-node structures on all 12 valid
formal fixtures. Both passed the exact traversal and location tests.

Candidate B's extraction adapter also matched all 13 committed expected JSON
files exactly.

The formal micro-fixture evidence therefore did not distinguish A and B.

T4 found an additional parser-level difference: Candidate B incorrectly
reported extensive error nodes in the valid pinned Lodash 4.17.21 bundle,
while Candidate A parsed it without error. Minified files are normally
excluded, so this did not retrospectively change Gate A, but it is a material
upstream correctness and robustness risk.

### Step 4 — Gate C risk

Gate C resolves the decision in favour of Candidate A.

Candidate A is maintained under the official Tree-sitter organisation and has
174 known downstream importers. Candidate B is a young pre-1.0
reimplementation with 26 known importers and strong dependence on one primary
maintainer.

Candidate B has a larger test suite, active issue handling and frequent
releases. However, its rapid pre-1.0 change rate, single-maintainer
concentration, heavy upstream test-suite resource demand, and demonstrated
valid-Lodash parsing defect create greater risk during the remaining project
period.

Candidate A also has risks: low recent repository activity, incomplete
release metadata, CGO integration requirements and explicit resource cleanup
obligations. These risks are more controllable within SBOMber's project scope.

### Step 5 — packaging

The decision was already resolved by Gate C, but packaging was still recorded.

Candidate B clearly wins packaging. It supports static CGO-free builds using
the standard Go cross-compilation toolchain. Its confirmed JavaScript,
TypeScript and TSX selective build is 9,277,602 bytes and passed all 13
extraction fixtures.

Candidate A requires CGO. The tested Linux binary is dynamically linked,
approximately 5.3 MB, and requires glibc 2.34 or later. Linux and macOS
artifacts must be built on suitable native runners.

R8 makes one operating system a must-have and cross-platform binaries a
should-have. The fixed rule therefore does not allow Candidate B's packaging
advantage to override Candidate A's lower upstream risk.

### Step 6 — throughput

Throughput was not used. The fixed rule permits it only if correctness, risk
and packaging all tie, and they did not.

## Selection

Select Candidate A: `github.com/tree-sitter/go-tree-sitter` v0.25.0 with the
pinned JavaScript and TypeScript grammar bindings.

Component 2 must now port the proven extraction contract and shared query
behaviour to the Candidate A production adapter.

## Fallback and trigger

Candidate B, `github.com/odvcencio/gotreesitter` v0.51.0, is the fallback.

Switch to Candidate B if an unfixable Candidate A grammar or location defect
blocks the production usage-graph milestone, or if Component 1 cannot produce
the required supported Linux artifact because of Candidate A's CGO and native
runtime requirements.

The shared fixtures, expected JSON schema and query contract remain fixed.
Only the parser adapter and packaging configuration should change.

Before switching, repeat the valid-Lodash regression against the Candidate B
version being considered. Do not switch while that valid-source parsing
defect remains relevant to the production scan scope.

## Consequences for Component 1

Component 1 must plan for:

- `CGO_ENABLED=1`;
- native build runners for supported operating systems;
- a dynamically linked Linux artifact;
- glibc 2.34 as the tested Linux compatibility floor;
- a stripped binary of approximately 5.3 MB;
- no assumption that ordinary `CGO_ENABLED=0` cross-compilation will work.

Linux is the must-have platform. Additional platform artifacts remain
should-have work and should be built and tested on matching runners.

T8 was not run during this spike and is deferred to S5-03.

## Known limitations

1. Candidate A requires CGO and native platform toolchains.
2. The tested Linux artifact requires glibc 2.34 or later.
3. Objects allocating C memory must be closed explicitly, including parsers,
   trees, tree cursors, queries, query cursors and lookahead iterators.
4. T5 long-run memory stability was not measured. It should be completed for
   the selected Candidate A adapter.
5. Candidate A and Candidate B produce different recovery trees for invalid
   JavaScript.
6. Candidate B incorrectly parsed the valid pinned Lodash 4.17.21 minified
   bundle during T4.
7. Minified and generated files are normally excluded, but filename-based
   exclusions may be incomplete.
8. The maximum individual source-file size is 1,000,000 bytes.
9. The per-file processing timeout is 5 seconds.
10. Files exceeding either limit must be reported as skipped or unresolved,
    never silently omitted.
11. T6 full-corpus throughput was not measured because it could not change the
    decision under the fixed rule.
12. T8 GitHub Actions validation was not run and is deferred to S5-03.
13. Candidate C's earlier location anomaly was not reproduced under this
    protocol, so it is not recorded as a confirmed defect.
14. The selected Candidate A semantic extraction adapter still needs to be
    completed using the proven shared contract.

## Benchmark caveat

Measurements were collected inside a VMware Workstation guest with 4 vCPUs,
4 GB allocated RAM and a 50 GB virtual disk.

Absolute timings must not be compared with published measurements from other
machines. Only comparisons measured on this VM under matched workloads are
meaningful.

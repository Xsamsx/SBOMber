# Component 2 — Parser Binding Selection: Test Protocol

**SBOMber · ICT30018 Project B · Sprint 4, task S4-05**
**Owner: Yevhen Hrekov · Reviewed by: Component 3 · Consumed by: Component 1 (packaging)**
**Status: protocol fixed before measurement — do not edit acceptance criteria after results exist**

---

## 0. What this document is

A repeatable procedure for choosing the parser Component 2 will use, and for producing the evidence that defends that choice.

It exists because the first pass of the spike compared candidates with different ad-hoc tests, on 8-line fixtures, and reached a conclusion partly on performance figures that do not matter at SBOMber's workload. That conclusion may still be right. This protocol is how you find out, and how you make the answer survive a panel question.

**Read section 1 before running anything.** The rules there are what make the result defensible; the tests are just mechanics.

---

## 1. Five rules that make this defensible

**1. Acceptance criteria are fixed before measurement.** Section 3 lists every gate. Once you run a candidate, those gates do not change. This is the same rule S4-18 applies to localisation ground truth, and it applies to your own spike for the same reason.

**2. Every candidate runs the identical fixture set through the identical harness contract.** One shared corpus, one shared output schema, one shared expected-results file. Different tests per candidate is how a spike becomes a justification for a decision already made.

**3. A hard gate failure eliminates. A measured difference informs.** Do not let a candidate that fails a required construct survive because it is faster. Do not let speed decide between two candidates that both pass.

**4. Claims about a library need a reproduction.** If you say a library returns wrong byte offsets, you must be able to hand someone twenty lines that demonstrate it. Definition of Done item 8 applies to your findings about other people's code too.

**5. Record what you did not test, by name.** A candidate excluded with a stated reason is a defensible scope decision. A candidate nobody noticed is a gap.

---

## 2. Candidates

| ID | Candidate | Module | Status entering this protocol |
|---|---|---|---|
| **A** | Official native Tree-sitter (CGO) | `github.com/tree-sitter/go-tree-sitter` + grammar bindings | Partially tested, passed smoke tests |
| **B** | Pure-Go runtime | `github.com/odvcencio/gotreesitter` | **Not tested — the gap** |
| **C** | Pure-Go Tree-sitter | `github.com/dcosson/treesitter-go` v0.1.0 | Partially tested, location anomaly unresolved |
| **D** | Official binding + runtime grammar loading | `go-tree-sitter` + purego `.so` loading | Not tested — record-only (§7, T9) |
| **E** | WASM wrapper | `github.com/malivvan/tree-sitter` | Eliminated at T1 — no JS/TS/TSX grammar |
| **F** | go-fAST | `github.com/t14raptor/go-fast` | Eliminated at T1 — no ESM source-type mode |

E and F are already eliminated on hard gates. Keep their evidence; do not re-run them.

**B is the one that matters.** The only cost of CGO is packaging, so the strongest no-CGO candidate has to be in the comparison. B is a pure-Go reimplementation of the tree-sitter runtime that ships javascript, typescript and tsx grammars, is at v0.20.x with a C-parity harness, and has build tags that embed only the grammars you name. If you reject the no-CGO route, reject it against B, not against C.

**Before writing any adapter for B, confirm its API on your VM** — do not copy names out of a README:

```bash
go doc github.com/odvcencio/gotreesitter/grammars
go doc github.com/odvcencio/gotreesitter.Parser
go doc github.com/odvcencio/gotreesitter.Node
go doc github.com/odvcencio/gotreesitter.NewQuery
```

---

## 3. The gates, fixed now

### Gate A — hard requirements. Any failure eliminates the candidate.

Derived from R1's construct list. A parser that cannot do these cannot produce `usage-graph.json`.

| # | Requirement | Why it is required |
|---|---|---|
| A1 | Parses JavaScript with ESM `import`/`export` | S4-06 |
| A2 | Parses TypeScript | Committed ecosystem |
| A3 | Parses TSX | Real npm apps contain it |
| A4 | Distinguishes named / default / aliased imports | R1 construct list |
| A5 | Namespace import `import * as _` resolves as a binding | R1 states explicitly this must not be marked unresolved |
| A6 | Destructured CommonJS `const { x } = require(...)` | R1 construct list |
| A7 | Package subpaths, including `@scope/pkg/sub` | R1 — easy to miss, maps to same SBOM component |
| A8 | **TypeScript `import type` is distinguishable in the AST from a value import** | R1 requires `type_only` classification; if the AST cannot tell them apart you cannot implement it, and counting type imports as usage produces false positives |
| A9 | Dynamic `import(expr)` is recognisable as a call, including the computed case | R1 — recorded as `unresolved`, never dropped |
| A10 | Member-expression call sites (`_.template(...)`) identifiable with the property name | This *is* the usage evidence |
| A11 | **Byte offsets and row/column exact against hand-labelled ground truth** | File and line are contract fields in `usage-graph.json` |
| A12 | Named function declarations and their call sites identifiable | R1.3 reachability: entry point → named function → dependency call |
| A13 | Invalid source produces an error node and does not panic | R9: the tool parses untrusted repositories |
| A14 | Does not execute or evaluate the source | R10 Component 2 obligation |

### Gate B — measured. Reported, does not eliminate.

| # | Measure | Recorded as |
|---|---|---|
| B1 | Full-corpus parse throughput on a real repository | ms total, MB/s |
| B2 | Peak RSS over the same corpus | KB |
| B3 | Stripped binary size, with only JS/TS/TSX grammars | MB |
| B4 | Build matrix outcome | pass/fail per target |
| B5 | Process start-up cost | ms |
| B6 | Memory stability over a long run | RSS drift |

### Gate C — project fit. Judgement, stated with reasons.

| # | Criterion |
|---|---|
| C1 | Maturity: version, release cadence, commit activity, test suite, downstream use |
| C2 | API stability risk over the remaining nine weeks |
| C3 | Resource-safety obligations the API imposes on the caller |
| C4 | Packaging effect on Component 1, weighed against R8 (one OS is must-have; cross-platform binaries are should-have) |
| C5 | Licence compatibility with Apache-2.0 |
| C6 | Query API availability — extraction written as `.scm` queries is far more maintainable than hand-written tree walks |

---

## 4. The fixture corpus

Your current fixtures are too small to distinguish the candidates. Replace them.

```
spikes/parser-bindings/
  corpus/
    micro/                  one construct per file, hand-labelled
    expected/               ground truth, written before any tool runs
    golden/                 s-expression snapshots from the reference parser
    perf/                   generated + real-world source for benchmarks
  queries/
    usage.scm               shared extraction query
  harness/                  per-candidate adapters, common output schema
  results/                  committed evidence
```

### 4.1 Micro fixtures

Create each file exactly as written. Keep them short — you are hand-labelling every offset.

**`corpus/micro/01-esm-named.js`**
```javascript
import { template } from "lodash";

export function render(data) {
  return template("<%= name %>")(data);
}
```

**`corpus/micro/02-esm-default-alias.js`**
```javascript
import axios from "axios";
import { get as httpGet } from "axios";

export async function fetchOne(url) {
  await axios.post(url, {});
  return httpGet(url);
}
```

**`corpus/micro/03-esm-namespace.js`**
```javascript
import * as _ from "lodash";

const method = "template";

export function build(src, data) {
  const compiled = _.template(src);
  const dynamic = _[method](src);
  return [compiled(data), dynamic(data)];
}
```
> A5 and A9 in one file. `_.template` must resolve; `_[method]` must be reported unresolved, not dropped.

**`corpus/micro/04-cjs-require.js`**
```javascript
const _ = require("lodash");
const path = require("node:path");

module.exports = function join(parts) {
  return _.uniq(parts).join(path.sep);
};
```

**`corpus/micro/05-cjs-destructured.js`**
```javascript
const { template, merge: mergeDeep } = require("lodash");

module.exports.apply = function apply(src, a, b) {
  return template(src)(mergeDeep(a, b));
};
```

**`corpus/micro/06-subpath.js`**
```javascript
import template from "lodash/template";
import parser from "@babel/parser/lib/index.js";

export function compile(src) {
  parser.parse(src);
  return template(src);
}
```

**`corpus/micro/07-dynamic-import.js`**
```javascript
const NAME = "lodash";

export async function loadStatic() {
  const mod = await import("lodash");
  return mod.template;
}

export async function loadComputed() {
  const mod = await import(NAME);
  return mod.template;
}
```

**`corpus/micro/08-type-only.ts`**
```typescript
import type { DebouncedFunc } from "lodash";
import { debounce } from "lodash";

export function wrap(fn: () => void): DebouncedFunc<() => void> {
  return debounce(fn, 100);
}
```
> A8. The `lodash` type import must be distinguishable from the value import on the next line.

**`corpus/micro/09-tsx-component.tsx`**
```typescript
import * as React from "react";
import { format } from "date-fns";

export const Stamp = ({ at }: { at: Date }) => <span>{format(at, "PP")}</span>;
```

**`corpus/micro/10-reachability.js`**
```javascript
import express from "express";
import { template } from "lodash";

const app = express();

function renderWelcome(name) {
  return template("hi <%= n %>")({ n: name });
}

function sendWelcome(res, name) {
  res.send(renderWelcome(name));
}

app.get("/welcome", (req, res) => sendWelcome(res, req.query.name));

export default app;
```
> A12, and this is the R1.3 fixture: route handler → `sendWelcome` → `renderWelcome` → `template`. Reuse it for S4-17.

**`corpus/micro/11-invalid.js`**
```javascript
import { template } from "lodash"

export function broken( {
  return template(
```

**`corpus/micro/12-edge.js`** — create with a script, not an editor, so the bytes are exact:
```bash
printf '\xEF\xBB\xBFimport { template } from "lodash";\r\nconst caf\xc3\xa9 = template("x");\r\nexport default caf\xc3\xa9;\r\n' \
  > corpus/micro/12-edge.js
```
> BOM, CRLF line endings, non-ASCII identifier. All three appear in real repositories and all three break naive offset handling.

**`corpus/micro/13-empty.js`** — zero bytes: `: > corpus/micro/13-empty.js`

### 4.2 Ground truth

For every micro fixture, write the expected result **before running any parser**. One file per fixture in `corpus/expected/`.

**`corpus/expected/01-esm-named.json`**
```json
{
  "fixture": "01-esm-named.js",
  "language": "javascript",
  "hasError": false,
  "imports": [
    {
      "specifier": "lodash",
      "kind": "esm_named",
      "local": "template",
      "imported": "template",
      "typeOnly": false,
      "line": 1,
      "column": 9
    }
  ],
  "calls": [
    { "callee": "template", "receiver": null, "line": 4, "column": 10 },
    { "callee": null, "receiver": null, "line": 4, "column": 10, "note": "immediately-invoked result" }
  ],
  "functions": [
    { "name": "render", "line": 3, "exported": true }
  ],
  "unresolved": []
}
```

Rules for filling these in:

- **Lines are 1-based, columns are 0-based UTF-8 byte columns.** Write that convention at the top of the expected file and make every harness normalise to it. Half the disagreements between parsers are this and nothing else.
- Derive offsets by hand or with `grep -bn`, not from a parser you are about to test.
- If you are unsure what the correct answer is for a construct, **drop the case rather than guess it**. Same rule as S4-18.
- Where two defensible answers exist (does an immediately-invoked call count as one call site or two?), pick one, write the rule down in the file header, and apply it identically to every candidate.

### 4.3 Golden s-expressions

The strongest fairness test available, and it costs almost nothing.

Both A and B claim to implement the same tree-sitter grammar tables. If that is true, their s-expression output for a given source file should be **identical**. Dump it from every candidate and diff.

```bash
# per candidate, per fixture
./harness-<id> sexp corpus/micro/01-esm-named.js > results/<id>/sexp/01-esm-named.txt
diff -u results/A/sexp/01-esm-named.txt results/B/sexp/01-esm-named.txt
```

A byte-identical diff across all 13 fixtures is far stronger evidence of parity than any set of assertions you could write by hand. A diff that is non-empty tells you exactly where a candidate deviates, and that difference goes in the report whichever way you decide.

### 4.4 Performance corpus

Micro fixtures cannot measure throughput. Build two:

```bash
mkdir -p corpus/perf

# real-world: a mid-sized TypeScript project, pinned by commit
git clone --depth 1 https://github.com/expressjs/express corpus/perf/express
git -C corpus/perf/express rev-parse HEAD > corpus/perf/express.commit

# synthetic: predictable size for MB/s reporting
python3 - <<'EOF'
import pathlib
out = pathlib.Path("corpus/perf/generated.ts")
parts = ['import { template } from "lodash";\n']
for i in range(2000):
    parts.append(f'export function fn{i}(x: string): string {{ return template(x + "{i}")(x); }}\n')
out.write_text("".join(parts))
EOF
wc -c corpus/perf/generated.ts
```

Pin the commit. A benchmark against a moving checkout is not reproducible.

---

## 5. The shared extraction query

Write the extraction once as a tree-sitter query and have every candidate run the same file. This tests Gate C6, removes "the adapters were written differently" as an objection, and is the code you will actually keep.

**`queries/usage.scm`**
```scheme
; ---- ESM imports -------------------------------------------------------
(import_statement
  source: (string) @import.source) @import.statement

; ---- CommonJS require --------------------------------------------------
(call_expression
  function: (identifier) @_req
  arguments: (arguments (string) @require.source)
  (#eq? @_req "require")) @require.statement

; ---- Dynamic import ----------------------------------------------------
(call_expression
  function: (import)
  arguments: (arguments) @dynamic.arguments) @dynamic.statement

; ---- Call sites --------------------------------------------------------
(call_expression
  function: (identifier) @call.name) @call.site

(call_expression
  function: (member_expression
    object: (identifier) @call.receiver
    property: (property_identifier) @call.property)) @call.site

; ---- Named functions, for reachability ---------------------------------
(function_declaration
  name: (identifier) @function.name) @function.decl
```

Verify each pattern compiles against each grammar before relying on it:

```bash
tree-sitter query queries/usage.scm corpus/micro/01-esm-named.js   # if the CLI is installed
```

If a candidate's query engine rejects a pattern the reference accepts, that is a Gate C6 finding — record it, do not silently rewrite the query for that candidate.

> **`import type` note.** The query above does not capture the type-only distinction, because how the grammar represents it is exactly what A8 is testing. Determine it empirically: dump the s-expression for `08-type-only.ts` and compare the two `import_statement` nodes. Write down what distinguishes them, then add a pattern. Do not assume a node name.

---

## 6. Harness contract

One small program per candidate. Same CLI, same output. Adapters differ only in how they obtain a language and walk a node.

```
harness-<id> gates      <fixture>   -> JSON: which Gate A items the parser supports
harness-<id> extract    <fixture>   -> JSON: the schema below
harness-<id> sexp       <fixture>   -> plain text s-expression
harness-<id> bench      <dir>       -> JSON: timing and allocation
```

**Output schema for `extract` — identical to `corpus/expected/*.json`.** That is the whole point: comparison is `diff <(harness extract f) corpus/expected/f.json` and nothing more.

```go
type Result struct {
    Fixture    string       `json:"fixture"`
    Language   string       `json:"language"`
    HasError   bool         `json:"hasError"`
    Imports    []Import     `json:"imports"`
    Calls      []Call       `json:"calls"`
    Functions  []Function   `json:"functions"`
    Unresolved []Unresolved `json:"unresolved"`
}

type Import struct {
    Specifier string `json:"specifier"`
    Kind      string `json:"kind"`      // esm_named | esm_default | esm_namespace | cjs | cjs_destructured | dynamic
    Local     string `json:"local"`
    Imported  string `json:"imported"`
    TypeOnly  bool   `json:"typeOnly"`
    Line      int    `json:"line"`      // 1-based
    Column    int    `json:"column"`    // 0-based UTF-8 byte column
}
```

Sort every slice deterministically before emitting — by line, then column, then name. Unsorted output produces diff noise that looks like disagreement.

**Both A and B must emit the same JSON for a passing fixture.** If they do not, one of them is wrong and finding out which is the work.

---

## 7. Test procedures

Run in order. Stop on a Gate A failure and record the elimination.

### T1 — Language availability
**Question:** can the candidate obtain javascript, typescript and tsx languages at all?

```bash
cd spikes/parser-bindings/harness/<id>
go doc <module> | grep -i -E 'javascript|typescript|tsx'
./harness-<id> gates corpus/micro/01-esm-named.js
```

**Pass:** all three languages instantiate and produce a non-nil tree.
**Evidence:** `results/<id>/T1.txt`.

This is where E was eliminated. Cost: 10 minutes per candidate.

### T2 — Construct extraction
**Question:** does it find every construct R1 requires?

```bash
for f in corpus/micro/*.{js,ts,tsx}; do
  base=$(basename "$f")
  ./harness-<id> extract "$f" > "results/<id>/extract/$base.json"
  diff -u "corpus/expected/${base%.*}.json" "results/<id>/extract/$base.json" \
    > "results/<id>/diff/$base.diff" || echo "MISMATCH: $base"
done
```

**Pass:** empty diff on every fixture except `11-invalid.js` (see T4).
**Fail is per-gate, not per-file:** map each mismatch to the A-number it breaks. A candidate that misses A5 is eliminated; a candidate that disagrees on the immediately-invoked-call convention is a specification bug in your expected file — fix the file and re-run every candidate.

### T2b — S-expression parity
**Question:** do the tree-sitter implementations actually agree?

```bash
for f in corpus/micro/*.{js,ts,tsx}; do
  base=$(basename "$f")
  ./harness-A sexp "$f" > "results/A/sexp/$base.txt"
  ./harness-B sexp "$f" > "results/B/sexp/$base.txt"
  diff -u "results/A/sexp/$base.txt" "results/B/sexp/$base.txt" \
    > "results/parity/$base.diff"
done
wc -l results/parity/*.diff
```

**Pass:** all diffs empty.
**If not empty:** the differing nodes go in the report by name. This is a genuine research observation about a parity claim, and it is worth more in the write-up than another benchmark.

### T3 — Location accuracy
**Question:** are byte offsets, lines and columns exactly right?

Location is the one thing you cannot recover from downstream — a wrong line in `usage-graph.json` is a wrong line in the client's report.

```bash
./harness-<id> extract corpus/micro/12-edge.js | jq '.imports[0], .calls[0]'
```

Check specifically:

- **Every fixture:** every reported line/column matches the expected file exactly.
- **`12-edge.js`:** the BOM must not shift column 0 on line 1; CRLF must not add a column; the non-ASCII identifier must not shift subsequent byte columns. State whether the parser reports UTF-8 byte columns or UTF-16 code units and normalise in the adapter, not in the comparison.
- **Traversal independence (this is the C anomaly):** extract the same node's range three ways — direct `Child(i)` indexing, named-child indexing, and a tree cursor — and assert all three agree.

```go
func TestTraversalAgreement(t *testing.T) {
    // locate the first call_expression by each route, compare StartByte/EndByte
    viaChild  := findCallByChildIndex(root)
    viaNamed  := findCallByNamedChild(root)
    viaCursor := findCallByCursor(tree)

    for _, pair := range [][2]Range{{viaChild, viaCursor}, {viaNamed, viaCursor}} {
        if pair[0] != pair[1] {
            t.Fatalf("traversal disagreement: %+v vs %+v", pair[0], pair[1])
        }
    }
    // and against ground truth
    if viaCursor.StartByte != 60 || viaCursor.EndByte != 80 {
        t.Fatalf("range %v does not match hand-labelled 60-80", viaCursor)
    }
}
```

**Pass:** all three routes agree with each other and with the hand-labelled offsets.
**If a candidate fails:** you now have the twenty-line reproduction rule 4 requires. Save it as `results/<id>/T3-repro/` with the fixture, the test, and the output. That is what makes "library X reports offset byte ranges under direct child traversal" a finding rather than an assertion. Open an upstream issue and cite it.

### T4 — Robustness against hostile input
**Question:** does it fail safely on input a real repository will contain?

```bash
./harness-<id> extract corpus/micro/11-invalid.js    # error node, exit 0, no panic
./harness-<id> extract corpus/micro/13-empty.js      # empty result, no panic

# minified bundle
curl -sL https://cdn.jsdelivr.net/npm/lodash@4.17.21/lodash.min.js \
  > corpus/perf/lodash.min.js
/usr/bin/time -f 'elapsed=%e maxrss=%M' ./harness-<id> extract corpus/perf/lodash.min.js

# deep nesting
python3 -c "print('f(' * 5000 + 'x' + ')' * 5000)" > corpus/perf/deep.js
/usr/bin/time -f 'elapsed=%e maxrss=%M' ./harness-<id> extract corpus/perf/deep.js

# not actually JavaScript
head -c 1000000 /dev/urandom > corpus/perf/binary.js
./harness-<id> extract corpus/perf/binary.js
```

**Pass:** every case terminates without panic, without unbounded memory, and reports an error state rather than a silent empty success.
**Record for R9:** the elapsed time and peak RSS on `deep.js` and the minified bundle. Those numbers set the defaults for "maximum individual file size" and "per-file parse timeout", which R9 says must be numbers rather than adjectives. This test does double duty — it is the only place in Sprint 4 those defaults get evidence.

> `node_modules`, minified bundles and generated output are excluded from production scans by R1. You still test them here, because exclusion is a filename rule and filename rules are wrong sometimes.

### T5 — Resource safety over a long run
**Question:** does memory stay flat across thousands of files?

This matters disproportionately for candidate A: the official binding requires explicit `Close` on anything holding C memory because `runtime.SetFinalizer` is unreliable across cgo. A missed `Close` is a native leak the Go race detector and heap profiler cannot see.

```bash
# 5000 parses, RSS sampled
./harness-<id> bench corpus/perf/express --repeat 5000 --rss-sample 500 \
  > results/<id>/T5.json
```

For candidate A, run it twice — once with correct `Close` discipline and once with the `Close` calls deliberately removed. If the second run's RSS climbs and the first stays flat, you have measured the cost of the discipline and proved your production code needs it.

**Pass:** RSS flat within noise across the run.
**Feeds:** a design rule in the architecture document and a leak test in the production package.

### T6 — Throughput
**Question:** how fast, on work that resembles a real scan?

```bash
CGO_ENABLED=<0|1> go test -run '^$' -bench 'BenchmarkCorpus' -benchmem -count=10 \
  ./harness/<id> | tee results/<id>/T6.txt
```

Report `ns/op`, `MB/s`, `B/op`, `allocs/op` as medians over 10 runs, on `corpus/perf/generated.ts` and on the pinned Express checkout.

**Interpretation rule, written now so it constrains you later:** a difference under 5× on a full-corpus run does not decide this. Scan wall-clock is dominated by network enrichment with 30-second HTTP timeouts, not by parsing. If the gap exceeds 5×, or if either candidate cannot parse the Express checkout inside 10 seconds, it becomes relevant and you say why. Otherwise it is a footnote.

**Note for candidate A:** `-benchmem` counts Go allocations only. C-side allocation is invisible to it. Report RSS from T5 alongside, and say plainly that the allocation figures are not comparable across the cgo boundary.

### T7 — Packaging matrix
**Question:** what does Component 1 have to do to ship this?

```bash
for target in "linux amd64" "linux arm64" "darwin amd64" "darwin arm64" "windows amd64"; do
  set -- $target
  for cgo in 0 1; do
    CGO_ENABLED=$cgo GOOS=$1 GOARCH=$2 go build -trimpath -ldflags='-s -w' \
      -o "/tmp/h-<id>-$1-$2-cgo$cgo" ./harness/<id> \
      2> "results/<id>/build-$1-$2-cgo$cgo.log" \
      && echo "PASS $1/$2 CGO=$cgo" || echo "FAIL $1/$2 CGO=$cgo"
  done
done

ls -lh /tmp/h-<id>-*
ldd /tmp/h-<id>-linux-amd64-cgo1 2>/dev/null || echo "static"
```

For candidate B, also measure the selective-embed path — its build tags let you embed only the grammars you name, which is what determines the real binary size:

```bash
go build -tags 'grammar_subset grammar_subset_javascript grammar_subset_typescript grammar_subset_tsx' \
  -trimpath -ldflags='-s -w' -o /tmp/h-B-subset ./harness/B
ls -lh /tmp/h-B-subset
```
Confirm the exact tag names against the repository before relying on them.

**Also record the glibc floor for any dynamically linked result:**
```bash
objdump -T /tmp/h-A-linux-amd64-cgo1 | grep GLIBC_ | \
  sed 's/.*GLIBC_\([0-9.]*\).*/\1/' | sort -uV | tail -1
```
That number is the oldest Linux the binary will run on, and it belongs in Known Limitations if the demo installation is ever a downloaded binary rather than a build from source.

### T8 — CI feasibility
**Question:** does it build and test on a GitHub Actions runner, on the OSes R8 cares about?

Add a throwaway workflow on your feature branch:

```yaml
name: parser-spike
on: workflow_dispatch
jobs:
  spike:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
        candidate: [A, B]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.26' }
      - run: go test -v ./spikes/parser-bindings/harness/${{ matrix.candidate }}
```

**Record:** pass/fail per cell, and job duration. Delete the workflow before the branch merges; keep the run URL as evidence.

### T9 — Runtime grammar loading (candidate D), record-only
**Question:** does the documented purego grammar-loading path remove the CGO requirement?

R1 instructs you to test the available binding *and loading* options, so this needs an answer even though it is 20 minutes of work.

```bash
go doc github.com/tree-sitter/go-tree-sitter | grep -i -E 'purego|dlopen|LoadLanguage'
CGO_ENABLED=0 go build ./harness/D 2>&1 | tee results/D/T9.txt
```

**Expected finding:** it does not. The grammar moves out of the binary into a `.so` loaded at runtime, but the tree-sitter *runtime* is still C reached through cgo. For packaging this is worse, not better — you would ship a binary plus per-platform shared libraries. Confirm that on your VM rather than asserting it, then record it in one paragraph and move on.

### T10 — Maturity review (Gate C, no code)

Fill this table per surviving candidate from the repository itself, not from a summary:

| Signal | How to check |
|---|---|
| Latest version and date | `git ls-remote --tags`, releases page |
| Release cadence | dates of last 5 releases |
| Commit activity | commits in the last 90 days |
| Open issue count and oldest unanswered | issues tab |
| Test suite scale | `go test ./... -count=1` in a clone; count test files |
| Downstream use | pkg.go.dev importers |
| Licence | `LICENSE`, check Apache-2.0 compatibility |
| Breaking-change history | CHANGELOG for API breaks in the last 6 months |

Both A and B carry maturity risk of different kinds — A is established but the ecosystem around grammar module paths is untidy (you already lost time to `tree-sitter-typescript` not being a separate module at `bindings/go`). B is a single-maintainer pre-1.0 reimplementation. State both honestly; do not let "official" do the argument's work for you.

---

## 8. Decision rule — fixed before results

Apply in order. Do not skip a step because you already know the answer.

1. **Eliminate** every candidate failing any Gate A item. Record which item, with the evidence file.
2. If one candidate remains, it is selected. Write the record and stop.
3. If more than one remains, rank the survivors on **correctness first**: T2 diff count, T2b parity, T3 traversal agreement. A candidate with zero location defects beats one with any.
4. If correctness ties, rank on **risk (Gate C)**: maturity, API stability, caller obligations. Nine weeks remain; an unfixable upstream bug in week 9 ends the deliverable.
5. If risk ties, rank on **packaging (T7) weighed against R8**: one OS is the must-have, cross-platform binaries are should-have. A CGO requirement costs you only should-have work. Do not inflate it.
6. Only if all of the above tie, use **throughput (T6)**, and only if the gap exceeds 5×.

**Name the fallback.** Whichever candidate places second is recorded as the fallback with the specific trigger that would switch to it — for example: *if an unfixable location or grammar defect in the selected parser blocks S5-04, switch; the harness contract means the adapter is the only code that changes.* Write that trigger down; it is a live risk-register entry, not decoration.

---

## 9. Evidence to commit

```
spikes/parser-bindings/
  TEST_PROTOCOL.md              this document, committed before results
  DECISION.md                   the record (below)
  corpus/                       fixtures, expected, golden, perf manifest
  queries/usage.scm
  harness/{A,B,C,D}/
  results/
    environment.txt             §10
    {A,B,C,D}/T1..T10 outputs
    parity/                     T2b diffs
    A/T3-repro/  B/T3-repro/    only if a defect was found
    matrix.md                   the filled comparison table
```

**`DECISION.md`** contains, in this order: the question, the candidates including the eliminated ones, the gates as fixed in §3, the results table, the decision rule applied step by step, the selection, the fallback and its trigger, the consequences for Component 1, and the Known Limitations entries this produces.

Do not commit the perf checkout — commit its pinned commit SHA.

---

## 10. Environment capture

Run once, before anything else, and commit the output. Every number in this spike is meaningless without it.

```bash
{
  date -Is
  uname -a
  lsb_release -d 2>/dev/null
  go version
  go env GOOS GOARCH CGO_ENABLED GOFLAGS
  gcc --version | head -1
  lscpu | grep -E 'Model name|CPU\(s\):|MHz'
  free -h | head -2
  echo "VM: VMware, host OS and allocated vCPU/RAM: <fill in by hand>"
} > results/environment.txt
```

Note the benchmark caveat plainly in `DECISION.md`: these figures come from a 4-vCPU VMware guest, so absolute timings are not comparable to published figures and only the ratios between candidates measured on this machine are meaningful.

---

## 11. Time budget

S4-05 is 6 hours and part of it is spent. Realistic remaining allocation:

| Step | Hours | Must / optional |
|---|---:|---|
| Fixture corpus and ground truth (§4) | 1.25 | Must — everything else depends on it |
| Shared query + harness contract (§5–6) | 0.75 | Must |
| Candidate B adapter and T1–T3 | 1.0 | Must — this is the gap |
| Re-run A through the same harness | 0.5 | Must — fairness |
| T2b parity diff | 0.25 | Must — highest evidence per minute |
| T4 robustness + R9 limit numbers | 0.5 | Must — also closes an R9 gap |
| T7 packaging matrix | 0.5 | Must — Component 1 is waiting |
| T9 purego record | 0.25 | Must — R1 asks for it |
| `DECISION.md` | 0.5 | Must |
| T5 leak test | 0.5 | Optional — do it if A is selected |
| T6 throughput | 0.5 | Optional — low decision weight |
| T8 CI matrix | 0.5 | Optional — defer to S5-03 |
| T10 maturity table | 0.25 | Must |
| C re-test and T3 repro | 0.5 | Optional — only to substantiate the defect claim |

**Must total: ~5.5 hours.** If time runs short, drop T6 first, then T8, then the C re-test. Do not drop T2b, T3 or the B adapter — those are the tests that change the answer.

---

## 12. What comes out of this, for other people

**For Component 1 (Sadman), the week you decide:**
- Whether CGO is required, and therefore whether `CGO_ENABLED=0` builds remain possible.
- If CGO: the glibc floor from T7, and that Linux and macOS binaries must be built on matching runners rather than cross-compiled.
- If pure Go: the build tags needed to keep the binary small, and the resulting size.
- The stripped binary size either way, so R8 packaging can be planned rather than discovered.

**For the architecture document (FEAT-15, owed to the client since Semester 1):**
- The selected parser, the pinned versions, and the decision record.
- The `import type` representation you determined empirically in §5 — that is the basis of `type_only` classification.
- The `Close`/resource-discipline rule from T5, as a design rule with a test behind it.
- The per-file size and timeout defaults from T4, feeding R9.

**For Known Limitations:**
- Constructs no candidate handled.
- The glibc floor, if applicable.
- Any s-expression parity divergence found in T2b.

**For the supervisor, Tuesday:** one sentence. *We compared four runnable parser candidates against a hand-labelled fixture set with fixed acceptance criteria, eliminated three on hard requirements, and selected the fourth on location correctness and maturity rather than performance; the packaging cost falls on should-have work.*

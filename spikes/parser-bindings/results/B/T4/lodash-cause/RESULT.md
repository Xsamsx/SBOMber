# Candidate B Lodash Cause Isolation

## Fixture

- Lodash version: 4.17.21
- bytes: 73,015
- SHA-256:
  `a9705dfc47c0763380d851ab1801be6f76019f6b67e40e9b873f8b4a0603f7a9`

## Default result

`ParseStrict` returned a tree without `ErrParseStoppedEarly`.

The tree reported:

- `ParseStoppedEarly()`: false;
- stop reason: `no_stacks_alive`;
- root type: `program`;
- root contained errors;
- root ended at byte 73,003 rather than byte 73,015.

The runtime diagnostic reported a truncated parse, but not a strict
early-stop safety reason.

## Limit tests

Setting `GOT_PARSE_NODE_LIMIT_SCALE=3` increased the recorded node budget from
3,796,780 to 11,390,340. It did not change the result.

Attempts with `GOT_GLR_MAX_STACKS` set to 16, 32 and 64 also did not produce a
successful parse. The combined node-scale and stack override did not produce
a successful parse.

The stack-override runs continued to report `maxStacks=8`, so this evidence
does not prove that a larger effective GLR stack budget was applied. It does
show that the documented override attempts did not resolve the failure.

## Interpretation

The result is not attributable to the documented node-count limit.

The public strict-parsing diagnostics did not classify it as an early parser
safety stop. Candidate B exhausted its viable parse stacks and failed to
accept a valid bundle that Candidate A accepted.

This is recorded as an observed parser/recovery incompatibility. It is not
described as a proven grammar defect, and the exact internal cause remains
unresolved.

The formal fixture comparison remains a correctness tie. This real-world
finding is considered under Gate C as reimplementation and operational risk.

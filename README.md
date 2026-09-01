# gooo-measurement-boundary-projector

`gooo-measurement-boundary-projector` compiles a `.gooo` measurement contract into a semantic measurement IR and a generated collector/wrapper. The collector turns deterministic fixture observations into receipt-bound consumer artifacts; the evaluator then projects each declared metric to `CLOSED`, `UNKNOWN`, or `REFUTED`.

The protocol is deliberately fail-closed:

- A metric with more than one authority, or with differing scope, unit, or identity digests, is `UNKNOWN`. Values are preserved as observations; they are never selected, averaged, or collapsed.
- An explicit measured zero is `0`. Missing or unexecuted work is `null` plus the six-field UNKNOWN frontier: `stage`, `step`, `reason`, `unknown_class`, `next_operation`, and `blocked_by`.
- Contradictory value/unit/scope evidence and tampered receipt chains are `REFUTED`; precedence is `REFUTED > UNKNOWN > CLOSED`.
- `before`, `after`, and `delta` are emitted only for an exact scope/authority/identity pair. Unscoped aggregate claims are forbidden.

The canonical v0.48 fixture reproduces `11ms/9748KiB` from `runtime.json` and `report.json` against `0ms/0KiB` from `releases/verification.json`. It must resolve to `UNKNOWN(CONFLICT)`. The repaired single-authority fixture measures each metric once and makes every consumer artifact reference the same receipt digest; it must resolve to `CLOSED`.

## v2 stage-boundary contract

The v2 contract is additive. The v1 contract, ten-case corpus, and mutable/immutable release history remain intact. `examples/measurement-boundary-v2.gooo` owns the v2 stage identity, causal start/end events, covered stable operation IDs, nested child-process coverage, monotonic clock and resolution, process-tree RSS scope, and input/output receipt digests. Go generates the collector and evaluates its receipts.

The v2 denominator is exactly twelve cases: `4 CLOSED`, `4 UNKNOWN`, and `4 REFUTED`, with precedence `REFUTED > UNKNOWN > CLOSED`. A positive `work_units` value paired with `wall_ms=0` is UNKNOWN at the declared resolution; zero work with zero duration is an explicit CLOSED measurement. UNKNOWN always carries `stage`, `step`, `reason`, `unknown_class`, `next_operation`, and `blocked_by`.

The generated integration fixture records before/after integer pairs in one GitHub Actions job only when scenario, input, contract, fixture, toolchain, runner, and job identity all match. No unscoped aggregate is emitted. The immutable optional v0.49 observation preserves the shared-ledger `168000ms` main-lock observation against product receipt `0ms/0ms` as `UNKNOWN(MEASUREMENT_SCOPE_MISMATCH)` and is not a product acceptance gate.

## CI-only workflow

GitHub Actions runs Go 1.27 compilation, build, tests, canonical conformance, and integration. All outputs are written to caller-owned temporary directories. Runtime authority is zero for repository writes, apply, commit, merge, tag, and release. The release workflow creates a draft from an annotated tag, audits its assets, then publishes and verifies the immutable release using `github.token`.

Run the pipeline in CI or with a caller-owned output directory:

```sh
go run ./cmd/measurement-boundary-projector run \
  --source examples/measurement-boundary.gooo \
  --fixture fixtures/cases/closed-single-authority-receipt.json \
  --out /tmp/gooo-measurement-boundary-projector-output
```

The local repository is an input only. The project intentionally does not prescribe local validation commands; the evidence record retains `local_validation_commands` and `OPERATIONAL_REFUTED` fields if a prohibited local operation is ever detected.

Operational provenance is separate from product semantics. The initial implementation push preceded a PR, so `pr_first_conformance=REFUTED` and one exact `OPERATIONAL_REFUTED` record are permanently preserved: stage `AUTHORING`, step `OPEN_IMPLEMENTATION_PR_BEFORE_MAIN_INTEGRATION`, reason `INITIAL_IMPLEMENTATION_PUSH_PRECEDED_PR`, initial main SHA `8c63fb3abb6f6026bbf274bc6b938c54b4fca3de`, and initial CI run `33540259958`. The follow-up is a remediation/conformance PR; history is not rewritten.

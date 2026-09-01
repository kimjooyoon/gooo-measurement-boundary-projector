# gooo-measurement-boundary-projector

`gooo-measurement-boundary-projector` compiles a `.gooo` measurement contract into a semantic measurement IR and a generated collector/wrapper. The collector turns deterministic fixture observations into receipt-bound consumer artifacts; the evaluator then projects each declared metric to `CLOSED`, `UNKNOWN`, or `REFUTED`.

The protocol is deliberately fail-closed:

- A metric with more than one authority, or with differing scope, unit, or identity digests, is `UNKNOWN`. Values are preserved as observations; they are never selected, averaged, or collapsed.
- An explicit measured zero is `0`. Missing or unexecuted work is `null` plus the six-field UNKNOWN frontier: `stage`, `step`, `reason`, `unknown_class`, `next_operation`, and `blocked_by`.
- Contradictory value/unit/scope evidence and tampered receipt chains are `REFUTED`; precedence is `REFUTED > UNKNOWN > CLOSED`.
- `before`, `after`, and `delta` are emitted only for an exact scope/authority/identity pair. Unscoped aggregate claims are forbidden.

The canonical v0.48 fixture reproduces `11ms/9748KiB` from `runtime.json` and `report.json` against `0ms/0KiB` from `releases/verification.json`. It must resolve to `UNKNOWN(CONFLICT)`. The repaired single-authority fixture measures each metric once and makes every consumer artifact reference the same receipt digest; it must resolve to `CLOSED`.

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

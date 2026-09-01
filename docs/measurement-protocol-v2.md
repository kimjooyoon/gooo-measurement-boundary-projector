# Measurement boundary protocol v2

The v2 `.gooo` source is the semantic authority. Each declared indicator binds a stable `stage_id` and `step` to causal `start_event` and `end_event` records, an exact set of stable covered operation IDs, expected child-process IDs and coverage mode, clock and resolution evidence, RSS process-tree scope, input/output receipt digests, unit, authority, and identity digests. Go is the generator, evaluator, and runtime for those declarations.

## Resolution rules

The evaluator checks receipt-chain integrity and refuting evidence first. A forged stage ID, receipt chain break, output inside a read-only input, or authority escalation is `REFUTED`. It then checks competing collectors and missing or mismatched stage evidence. Those conditions are `UNKNOWN` with all six frontier fields. Only exact stage coverage can be `CLOSED`.

`work_units` and `wall_ms` are integer observations. `work_units > 0` with `wall_ms == 0` cannot be CLOSED because the work is below the declared wall-clock resolution. `work_units == 0` with `wall_ms == 0` is a valid measured zero. Missing values are `null`; they are never replaced with zero.

An improvement is a per-indicator result only for a before/after pair with the same `scenario_id`, input digest, contract digest, fixture digest, toolchain, runner, CI job, pair ID, and exact scope. Otherwise that per-indicator improvement remains UNKNOWN. The report exposes measured stage IDs, causal events, covered operations, child-process coverage, clock/resolution, RSS scope, and receipt digests to the consumer.

## Canonical denominator and optional input

The canonical v2 corpus has twelve cases: four CLOSED, four UNKNOWN, and four REFUTED. Its UNKNOWN cases cover stage-not-entered, missing end event, positive work below resolution, and competing collector authority. Its REFUTED cases cover forged stage ID, receipt chain break, output inside read-only input, and authority escalation.

The immutable optional v0.49 input records the shared-ledger main-lock observation as `168000ms` while the product integration baseline and candidate receipts are both `0ms`. The conflict is retained as `UNKNOWN/MEASUREMENT_SCOPE_MISMATCH`; it is not a required product acceptance gate.

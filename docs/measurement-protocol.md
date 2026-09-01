# Measurement boundary protocol v1

## Source authority

The `.gooo` file is the business-intent authority. Every measurement declares a stable ID, stage, step, inclusive start/end boundaries, included and excluded operations, unit, source authority, observation method, scope, identity digests, direction, nullable policy, and conflict precedence. The semantic IR is a normalized transport, not a second source of intent.

## Resolution

The evaluator first checks receipt integrity and explicit contradictions. A failed integrity or contradiction is `REFUTED`. It then checks whether all observations for a metric share one exact authority, scope, unit, identity, stage, step, and boundary. Any conflict is `UNKNOWN` with no chosen value. Missing observations and `measured=false` observations produce a null value and an UNKNOWN frontier. Only a single valid receipt chain with all consumer references matching can be `CLOSED`.

The six UNKNOWN fields are always emitted together:

```json
{
  "stage": "report",
  "step": "report/verify",
  "reason": "MULTIPLE_SOURCE_AUTHORITIES",
  "unknown_class": "DUPLICATE_AUTHORITY",
  "next_operation": "OBTAIN_SINGLE_AUTHORITY_MEASUREMENT",
  "blocked_by": ["runtime.json", "releases/verification.json"]
}
```

An explicit zero is a measured value, not a missing sentinel. Before/after/delta is an exact-pair projection and is not a score, percentage, or overall improvement metric.

## Evidence boundary

The generated collector writes only to the caller-owned output directory. It declares zero runtime writes and zero apply/commit/merge/tag/release authority. The GitHub Actions workflow is the only validation authority. Local validation commands remain an explicit empty vector in CI evidence; an accidental local validation operation must be preserved as `OPERATIONAL_REFUTED` with the exact command rather than removed from the record.

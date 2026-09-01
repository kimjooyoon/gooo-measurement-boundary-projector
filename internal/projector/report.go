package projector

import (
	"fmt"
	"strings"
)

func RenderHumanReport(evaluation Evaluation) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Measurement boundary projector report\n\n")
	fmt.Fprintf(&builder, "decision: `%s`\n", evaluation.Decision)
	fmt.Fprintf(&builder, "fail_closed: `%t`\n", evaluation.FailClosed)
	fmt.Fprintf(&builder, "semantic_ir_digest: `%s`\n", evaluation.IRDigest)
	fmt.Fprintf(&builder, "collection_digest: `%s`\n", evaluation.CollectionDigest)
	fmt.Fprintf(&builder, "aggregate_policy: `%s`\n\n", evaluation.AggregatePolicy)
	fmt.Fprintf(&builder, "summary: closed=%d unknown=%d refuted=%d\n\n", evaluation.ClosedCount, evaluation.UnknownCount, evaluation.RefutedCount)
	builder.WriteString("## Claim\n\n")
	fmt.Fprintf(&builder, "state: `%s`\n", evaluation.Claim.State)
	fmt.Fprintf(&builder, "stage: `%s`\n", evaluation.Claim.Stage)
	fmt.Fprintf(&builder, "step: `%s`\n", evaluation.Claim.Step)
	fmt.Fprintf(&builder, "reason: `%s`\n", evaluation.Claim.Reason)
	fmt.Fprintf(&builder, "unknown_class: `%s`\n", evaluation.Claim.UnknownClass)
	fmt.Fprintf(&builder, "next_operation: `%s`\n", evaluation.Claim.NextOperation)
	fmt.Fprintf(&builder, "blocked_by: `%s`\n\n", strings.Join(evaluation.Claim.BlockedBy, ", "))
	builder.WriteString("## Measurements\n\n")
	for _, metric := range evaluation.Metrics {
		fmt.Fprintf(&builder, "### `%s`\n\n", metric.MeasurementID)
		fmt.Fprintf(&builder, "state: `%s`\n", metric.State)
		fmt.Fprintf(&builder, "stage: `%s`; step: `%s`\n", metric.Stage, metric.Step)
		fmt.Fprintf(&builder, "reason: `%s`\n", metric.Reason)
		fmt.Fprintf(&builder, "value: `%s`; declared_unit: `%s`; declared_scope: `%s`\n", displayValue(metric.Value), metric.Unit, metric.Scope)
		fmt.Fprintf(&builder, "authority: `%s`\n", metric.Authority)
		fmt.Fprintf(&builder, "receipt_digests: `%s`\n", strings.Join(metric.ReceiptDigests, ", "))
		fmt.Fprintf(&builder, "consumer_artifacts: `%s`\n", strings.Join(metric.ConsumerArtifacts, ", "))
		if metric.Before != nil || metric.After != nil || metric.Delta != nil {
			fmt.Fprintf(&builder, "exact_pair: before=%s after=%s delta=%s\n", displayValue(metric.Before), displayValue(metric.After), displayValue(metric.Delta))
		}
		for _, observed := range metric.ObservedValues {
			fmt.Fprintf(&builder, "observed: authority=%s artifact=%s value=%s unit=%s scope=%s receipt=%s\n", observed.Authority, observed.SourceArtifact, displayValue(observed.Value), observed.Unit, observed.Scope, observed.ReceiptDigest)
		}
		if metric.Unknown != nil {
			builder.WriteString("unknown_frontier:\n")
			fmt.Fprintf(&builder, "- stage=%s\n- step=%s\n- reason=%s\n- unknown_class=%s\n- next_operation=%s\n- blocked_by=%s\n", metric.Unknown.Stage, metric.Unknown.Step, metric.Unknown.Reason, metric.Unknown.UnknownClass, metric.Unknown.NextOperation, strings.Join(metric.Unknown.BlockedBy, ", "))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("## Boundary policy\n\n")
	builder.WriteString("A metric is CLOSED only when one declared scope, unit, authority, exact identity, and valid receipt chain are present. Missing or unexecuted values remain `null` and become UNKNOWN. REFUTED takes precedence over UNKNOWN, which takes precedence over CLOSED. No value is selected or averaged across conflicting authorities.\n\n")
	builder.WriteString("Before/after/delta is emitted only for an exact pair. Unscoped aggregate claims are forbidden.\n")
	return builder.String()
}

func displayValue(value *float64) string {
	if value == nil {
		return "null"
	}
	return fmt.Sprintf("%g", *value)
}

package projector

import (
	"fmt"
	"strings"
)

func RenderV2HumanReport(evaluation V2Evaluation) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Measurement boundary projector v2 report\n\n")
	fmt.Fprintf(&builder, "decision: `%s`\n", evaluation.Decision)
	fmt.Fprintf(&builder, "fail_closed: `%t`\n", evaluation.FailClosed)
	fmt.Fprintf(&builder, "semantic_ir_digest: `%s`\n", evaluation.IRDigest)
	fmt.Fprintf(&builder, "collection_digest: `%s`\n", evaluation.CollectionDigest)
	fmt.Fprintf(&builder, "denominator: closed=%d unknown=%d refuted=%d\n", evaluation.ClosedCount, evaluation.UnknownCount, evaluation.RefutedCount)
	fmt.Fprintf(&builder, "aggregate_policy: `%s`\n\n", evaluation.AggregatePolicy)
	builder.WriteString("## Claim\n\n")
	fmt.Fprintf(&builder, "state: `%s`\n", evaluation.Claim.State)
	fmt.Fprintf(&builder, "stage_id: `%s`\n", evaluation.Claim.StageID)
	fmt.Fprintf(&builder, "stage: `%s`; step: `%s`\n", evaluation.Claim.Stage, evaluation.Claim.Step)
	fmt.Fprintf(&builder, "reason: `%s`\n", evaluation.Claim.Reason)
	fmt.Fprintf(&builder, "unknown_class: `%s`\n", evaluation.Claim.UnknownClass)
	fmt.Fprintf(&builder, "next_operation: `%s`\n", evaluation.Claim.NextOperation)
	fmt.Fprintf(&builder, "blocked_by: `%s`\n\n", strings.Join(evaluation.Claim.BlockedBy, ", "))
	builder.WriteString("## Measured stages and indicators\n\n")
	for _, metric := range evaluation.Metrics {
		fmt.Fprintf(&builder, "### `%s`\n\n", metric.MeasurementID)
		fmt.Fprintf(&builder, "state: `%s`; reason: `%s`\n", metric.State, metric.Reason)
		fmt.Fprintf(&builder, "stage_id: `%s`; stage: `%s`; step: `%s`\n", metric.StageID, metric.Stage, metric.Step)
		fmt.Fprintf(&builder, "causal_events: start=`%s`; end=`%s`\n", metric.CausalEvents.Start, metric.CausalEvents.End)
		fmt.Fprintf(&builder, "covered_operations: `%s`\n", strings.Join(metric.CoveredOperations, ", "))
		fmt.Fprintf(&builder, "covered_child_processes: `%s`; child_process_coverage: `%s`\n", strings.Join(metric.CoveredChildProcesses, ", "), metric.ChildProcessCoverage)
		fmt.Fprintf(&builder, "clock: `%s`; resolution_ms: `%d`; rss_process_tree_scope: `%s`\n", metric.Clock, metric.ResolutionMS, metric.RSSProcessTreeScope)
		fmt.Fprintf(&builder, "input_receipt_digest: `%s`\noutput_receipt_digest: `%s`\n", metric.InputReceiptDigest, metric.OutputReceiptDigest)
		fmt.Fprintf(&builder, "value: `%s`; work_units: `%s`; peak_rss_kib: `%s`; unit: `%s`; scope: `%s`\n", displayV2Int(metric.Value), displayV2Int(metric.WorkUnits), displayV2Int(metric.PeakRSSKiB), metric.Unit, metric.Scope)
		fmt.Fprintf(&builder, "authority: `%s`; receipt_digests: `%s`\n", metric.Authority, strings.Join(metric.ReceiptDigests, ", "))
		fmt.Fprintf(&builder, "consumer_artifacts: `%s`\n", strings.Join(metric.ConsumerArtifacts, ", "))
		fmt.Fprintf(&builder, "improvement: state=`%s`; reason=`%s`\n", metric.Improvement.State, metric.Improvement.Reason)
		if metric.Improvement.Before != nil || metric.Improvement.After != nil || metric.Improvement.Delta != nil {
			fmt.Fprintf(&builder, "improvement_pair: pair_id=`%s`; before=%s; after=%s; delta=%s\n", metric.Improvement.PairID, displayV2Int(metric.Improvement.Before), displayV2Int(metric.Improvement.After), displayV2Int(metric.Improvement.Delta))
		}
		for _, observed := range metric.ObservedValues {
			fmt.Fprintf(&builder, "observed: stage_id=%s authority=%s artifact=%s value=%s work_units=%s peak_rss_kib=%s unit=%s scope=%s receipt=%s\n", observed.StageID, observed.Authority, observed.SourceArtifact, displayV2Int(observed.Value), displayV2Int(observed.WorkUnits), displayV2Int(observed.PeakRSSKiB), observed.Unit, observed.Scope, observed.ReceiptDigest)
		}
		if metric.Unknown != nil {
			builder.WriteString("unknown_frontier:\n")
			fmt.Fprintf(&builder, "- stage=%s\n- step=%s\n- reason=%s\n- unknown_class=%s\n- next_operation=%s\n- blocked_by=%s\n", metric.Unknown.Stage, metric.Unknown.Step, metric.Unknown.Reason, metric.Unknown.UnknownClass, metric.Unknown.NextOperation, strings.Join(metric.Unknown.BlockedBy, ", "))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("## Boundary policy\n\n")
	builder.WriteString("CLOSED requires one exact measured stage, causal start/end events, covered operations, declared child-process coverage, clock and resolution evidence, RSS process-tree scope, input/output receipt digests, and a valid receipt chain. A positive work_units observation with wall_ms=0 is UNKNOWN because the duration is below the declared resolution. Missing values remain null plus the six-field UNKNOWN frontier. REFUTED takes precedence over UNKNOWN, which takes precedence over CLOSED.\n\n")
	builder.WriteString("Improvement is per-indicator only when the before/after pair has the same scenario, input, contract, fixture, toolchain, runner, and CI job. No unscoped scalar is emitted.\n")
	return builder.String()
}

func displayV2Int(value *int64) string {
	if value == nil {
		return "null"
	}
	return fmt.Sprintf("%d", *value)
}

package projector

import (
	"fmt"
	"path/filepath"
	"sort"
)

func EvaluateV2(ir V2SemanticIR, collection V2Collection, outputPath string) (V2Evaluation, error) {
	if ir.Schema != V2IRSchema {
		return V2Evaluation{}, fmt.Errorf("unexpected v2 semantic IR schema %q", ir.Schema)
	}
	computedIR, err := digestV2IR(ir)
	if err != nil {
		return V2Evaluation{}, err
	}
	if computedIR != ir.Digest {
		return V2Evaluation{}, fmt.Errorf("v2 semantic IR digest mismatch: declared %s computed %s", ir.Digest, computedIR)
	}
	if collection.Schema != V2CollectionSchema {
		return V2Evaluation{}, fmt.Errorf("unexpected v2 collection schema %q", collection.Schema)
	}
	computedCollection, err := digestV2Collection(collection)
	if err != nil {
		return V2Evaluation{}, err
	}
	if computedCollection != collection.Digest {
		return V2Evaluation{}, fmt.Errorf("v2 collection digest mismatch: declared %s computed %s", collection.Digest, computedCollection)
	}
	if collection.IRDigest != ir.Digest {
		return V2Evaluation{}, fmt.Errorf("v2 collection is bound to IR %s, expected %s", collection.IRDigest, ir.Digest)
	}
	byMetric := map[string][]V2CollectedObservation{}
	for _, observation := range collection.Observations {
		byMetric[observation.MetricID] = append(byMetric[observation.MetricID], observation)
	}
	results := make([]V2MetricResult, 0, len(ir.Measurements))
	for _, spec := range ir.Measurements {
		results = append(results, evaluateV2Metric(spec, byMetric[spec.MeasurementID], collection))
	}
	closedCount, unknownCount, refutedCount := 0, 0, 0
	for _, result := range results {
		switch result.State {
		case V2Closed:
			closedCount++
		case V2Unknown:
			unknownCount++
		case V2Refuted:
			refutedCount++
		}
	}
	decision := V2Closed
	if refutedCount > 0 {
		decision = V2Refuted
	} else if unknownCount > 0 {
		decision = V2Unknown
	}
	evaluation := V2Evaluation{
		Schema: V2EvaluationSchema, IRDigest: ir.Digest, CollectionDigest: collection.Digest,
		Decision: decision, FailClosed: decision != V2Closed, Claim: buildV2Claim(decision, results), Metrics: results,
		ClosedCount: closedCount, UnknownCount: unknownCount, RefutedCount: refutedCount,
		AggregatePolicy: "FORBID_UNSCOPED_SCALAR",
	}
	if outputPath != "" {
		if err := requireOutputOutside(outputPath, filepath.Dir(ir.SourcePath)); err != nil {
			return V2Evaluation{}, err
		}
		if err := WriteJSON(outputPath, evaluation); err != nil {
			return V2Evaluation{}, err
		}
	}
	return evaluation, nil
}

func evaluateV2Metric(spec V2MeasurementSpec, observations []V2CollectedObservation, collection V2Collection) V2MetricResult {
	result := V2MetricResult{
		MeasurementID: spec.MeasurementID, State: V2Closed, StageID: spec.StageID, Stage: spec.Stage, Step: spec.Step,
		CausalEvents: spec.CausalEvents, CoveredOperations: append([]string(nil), spec.CoveredOperations...),
		ChildProcessCoverage: spec.ChildProcessCoverage, Clock: spec.Clock,
		ResolutionMS: spec.ResolutionMS, RSSProcessTreeScope: spec.RSSProcessTreeScope,
		InputReceiptDigest: spec.InputReceiptDigest, OutputReceiptDigest: spec.OutputReceiptDigest,
		Reason: "EXACT_SINGLE_STAGE_COVERAGE", Unit: spec.Unit, Scope: spec.Scope, Authority: spec.SourceAuthority,
		Authorities: make([]string, 0), Scopes: make([]string, 0), IdentityDigests: spec.IdentityDigests,
		ReceiptDigests: make([]string, 0), ConsumerArtifacts: make([]string, 0), ObservedValues: make([]V2ObservedValue, 0),
		Improvement: v2ImprovementUnknown(spec, "NO_EXACT_BEFORE_AFTER_PAIR", "PAIR_NOT_EXACT", "PROVIDE_EXACT_BEFORE_AFTER_IDENTITY_PAIR", []string{spec.MeasurementID}),
	}
	if len(observations) == 0 {
		return setV2Unknown(result, spec, "STAGE_NOT_ENTERED", "STAGE_NOT_ENTERED", "ENTER_DECLARED_STAGE_AND_EMIT_START_EVENT", []string{spec.StageID})
	}
	for _, observation := range observations {
		result.ObservedValues = append(result.ObservedValues, V2ObservedValue{
			Authority: observation.SourceAuthority, SourceArtifact: observation.SourceArtifact, Value: observation.Value,
			WorkUnits: observation.WorkUnits, PeakRSSKiB: observation.PeakRSSKiB, Unit: observation.Unit,
			Scope: observation.Scope, StageID: observation.StageID, ReceiptDigest: observation.ReceiptDigest,
		})
		result.ReceiptDigests = append(result.ReceiptDigests, observation.ReceiptDigest)
		result.ConsumerArtifacts = append(result.ConsumerArtifacts, v2ConsumerNames(observation)...)
		result.Authorities = append(result.Authorities, observation.SourceAuthority)
		result.Scopes = append(result.Scopes, observation.Scope)
	}
	result.Authorities = uniqueStrings(result.Authorities)
	result.Scopes = uniqueStrings(result.Scopes)
	result.ReceiptDigests = uniqueStrings(result.ReceiptDigests)
	result.ConsumerArtifacts = uniqueStrings(result.ConsumerArtifacts)
	result.CoveredChildProcesses = append([]string(nil), observations[0].CoveredChildProcesses...)

	for _, observation := range observations {
		if reason := v2ReceiptProblem(observation, collection); reason != "" {
			return setV2Refuted(result, spec, reason, "RESTORE_RECEIPT_CHAIN", []string{observation.SourceArtifact})
		}
		if observation.StageID != spec.StageID {
			return setV2Refuted(result, spec, "FORGED_STAGE_ID", "RESTORE_DECLARED_STAGE_ID", []string{observation.StageID})
		}
		if observation.OutputInsideReadOnlyInput {
			return setV2Refuted(result, spec, "OUTPUT_INSIDE_READ_ONLY_INPUT", "WRITE_ONLY_TO_CALLER_OWNED_OUTPUT", []string{observation.SourceArtifact})
		}
		if observation.AuthorityEscalation {
			return setV2Refuted(result, spec, "AUTHORITY_ESCALATION", "RESTORE_ZERO_RUNTIME_AUTHORITY", []string{observation.SourceArtifact})
		}
	}
	if v2AuthorityEscalated(collection.Collector.RuntimeAuthority) || collection.Collector.OperatorAuthority != "authoring-only" {
		return setV2Refuted(result, spec, "AUTHORITY_ESCALATION", "RESTORE_ZERO_RUNTIME_AUTHORITY", []string{"collector"})
	}
	if len(collection.Collector.Authorities) > 1 || len(result.Authorities) > 1 {
		return setV2Unknown(result, spec, "COMPETING_COLLECTOR_AUTHORITY", "COMPETING_COLLECTOR_AUTHORITY", "SELECT_ONE_COLLECTOR_AUTHORITY", append(result.Authorities, collection.Collector.Authorities...))
	}
	if len(result.Authorities) == 1 && result.Authorities[0] != spec.SourceAuthority {
		return setV2Unknown(result, spec, "DECLARED_AUTHORITY_NOT_OBSERVED", "COMPETING_COLLECTOR_AUTHORITY", "EMIT_THE_DECLARED_COLLECTOR_AUTHORITY", result.Authorities)
	}
	if !collection.Collector.Generated || !collection.Collector.MeasuredOnce {
		return setV2Unknown(result, spec, "GENERATED_COLLECTOR_NOT_SINGLE_STAGE", "COLLECTOR_NOT_CLOSED", "RUN_GENERATED_STAGE_COLLECTOR_ONCE", []string{spec.StageID})
	}
	for _, observation := range observations {
		if !observation.StageEntered {
			return setV2Unknown(result, spec, "STAGE_NOT_ENTERED", "STAGE_NOT_ENTERED", "ENTER_DECLARED_STAGE_AND_EMIT_START_EVENT", []string{spec.StageID})
		}
		if !observation.EndObserved || observation.EndEvent == "" {
			return setV2Unknown(result, spec, "END_EVENT_MISSING", "END_EVENT_MISSING", "EMIT_DECLARED_END_EVENT", []string{spec.CausalEvents.End})
		}
		if observation.Stage != spec.Stage || observation.Step != spec.Step || observation.StartEvent != spec.CausalEvents.Start || observation.EndEvent != spec.CausalEvents.End {
			return setV2Unknown(result, spec, "MEASUREMENT_SCOPE_MISMATCH", "MEASUREMENT_SCOPE_MISMATCH", "RECORD_EXACT_CAUSAL_STAGE_BOUNDARY", []string{observation.SourceArtifact})
		}
		if !v2ExactStrings(observation.CoveredOperations, spec.CoveredOperations) || hasV2ExcludedOperation(spec, observation) {
			return setV2Unknown(result, spec, "MEASUREMENT_SCOPE_MISMATCH", "MEASUREMENT_SCOPE_MISMATCH", "RECORD_EXACT_COVERED_OPERATIONS", []string{observation.SourceArtifact})
		}
		if observation.ChildProcessCoverage != spec.ChildProcessCoverage || !v2ExactStrings(observation.CoveredChildProcesses, spec.ExpectedChildProcesses) {
			return setV2Unknown(result, spec, "MEASUREMENT_SCOPE_MISMATCH", "MEASUREMENT_SCOPE_MISMATCH", "RECORD_EXACT_CHILD_PROCESS_COVERAGE", []string{observation.SourceArtifact})
		}
		if observation.Clock != spec.Clock || observation.ResolutionMS != spec.ResolutionMS || observation.RSSProcessTreeScope != spec.RSSProcessTreeScope || observation.Scope != spec.Scope {
			return setV2Unknown(result, spec, "MEASUREMENT_SCOPE_MISMATCH", "MEASUREMENT_SCOPE_MISMATCH", "RECORD_CLOCK_RESOLUTION_AND_PROCESS_TREE_SCOPE", []string{observation.SourceArtifact})
		}
		if !isValidDigest(observation.InputReceiptDigest) || !isValidDigest(observation.OutputReceiptDigest) || observation.InputReceiptDigest != spec.InputReceiptDigest || observation.OutputReceiptDigest != spec.OutputReceiptDigest {
			return setV2Unknown(result, spec, "MEASUREMENT_SCOPE_MISMATCH", "MEASUREMENT_SCOPE_MISMATCH", "RECORD_INPUT_OUTPUT_RECEIPT_DIGESTS", []string{observation.SourceArtifact})
		}
		if !mapsEqual(observation.IdentityDigests, spec.IdentityDigests) {
			return setV2Unknown(result, spec, "MEASUREMENT_SCOPE_MISMATCH", "MEASUREMENT_SCOPE_MISMATCH", "RECORD_EXACT_STAGE_IDENTITY_DIGESTS", []string{observation.SourceArtifact})
		}
		if observation.ObservationMethod == "external-utility" && !observation.ExternalUtilityEvidence {
			return setV2Unknown(result, spec, "EXTERNAL_UTILITY_EVIDENCE_UNAVAILABLE", "EXTERNAL_EVIDENCE_MISSING", "OBTAIN_INDEPENDENT_USER_WORKLOAD_EVIDENCE", []string{observation.SourceArtifact})
		}
		if observation.ObservationMethod != spec.ObservationMethod || observation.Direction != spec.Direction || observation.Unit != spec.Unit {
			return setV2Unknown(result, spec, "MEASUREMENT_SCOPE_MISMATCH", "MEASUREMENT_SCOPE_MISMATCH", "RECORD_DECLARED_METHOD_DIRECTION_AND_UNIT", []string{observation.SourceArtifact})
		}
		if observation.Clock == "" || observation.ResolutionMS <= 0 || observation.RSSProcessTreeScope == "" || observation.Scope == "" {
			return setV2Unknown(result, spec, "MEASUREMENT_SCOPE_MISMATCH", "MEASUREMENT_SCOPE_MISMATCH", "RECORD_RESOLUTION_AND_PROCESS_TREE_SCOPE", []string{observation.SourceArtifact})
		}
		if observation.WorkUnits == nil {
			return setV2Unknown(result, spec, "WORK_UNITS_MISSING", "MEASUREMENT_SCOPE_MISMATCH", "RECORD_INTEGER_WORK_UNITS", []string{observation.SourceArtifact})
		}
		if !observation.Measured || observation.Value == nil {
			return setV2Unknown(result, spec, "MEASUREMENT_NOT_EXECUTED", "MISSING_MEASUREMENT", "EXECUTE_DECLARED_STAGE_MEASUREMENT", []string{observation.SourceArtifact})
		}
		if spec.Unit == "ms" && *observation.WorkUnits > 0 && *observation.Value == 0 {
			return setV2Unknown(result, spec, "POSITIVE_WORK_BELOW_RESOLUTION", "MEASUREMENT_SCOPE_MISMATCH", "RECORD_MONOTONIC_DURATION_AT_DECLARED_RESOLUTION", []string{observation.SourceArtifact})
		}
	}
	if len(result.ConsumerArtifacts) == 0 {
		return setV2Unknown(result, spec, "NO_CONSUMER_RECEIPT_REFERENCE", "RECEIPT_REFERENCE_MISSING", "REFERENCE_THE_MEASURED_STAGE_RECEIPT", []string{spec.MeasurementID})
	}
	allowedReceipts := map[string]bool{}
	for _, observation := range observations {
		allowedReceipts[observation.ReceiptDigest] = true
	}
	for _, consumer := range collection.Consumers {
		if consumer.MetricID != spec.MeasurementID {
			continue
		}
		if !allowedReceipts[consumer.ReceiptDigest] || consumer.StageID != spec.StageID || !v2ExactStrings(consumer.CoveredOperations, spec.CoveredOperations) {
			return setV2Refuted(result, spec, "RECEIPT_CHAIN_BREAK", "RESTORE_CONSUMER_STAGE_RECEIPT_REFERENCE", []string{consumer.Name})
		}
	}
	if len(observations) > 2 {
		return setV2Unknown(result, spec, "MULTIPLE_OBSERVATIONS_NOT_MATCHED_PAIR", "PAIR_NOT_EXACT", "PUBLISH_ONE_EXACT_BEFORE_AFTER_PAIR", result.ReceiptDigests)
	}
	if len(observations) == 1 {
		result.Value = observations[0].Value
		result.WorkUnits = observations[0].WorkUnits
		result.PeakRSSKiB = observations[0].PeakRSSKiB
		if observations[0].Phase != "" {
			result.Improvement = v2ImprovementUnknown(spec, "BEFORE_AFTER_PAIR_NOT_EXACT", "PAIR_NOT_EXACT", "PROVIDE_EXACT_BEFORE_AFTER_IDENTITY_PAIR", []string{observations[0].SourceArtifact})
		}
		return result
	}
	if !v2PairShape(observations) || observations[0].Value == nil || observations[1].Value == nil {
		return setV2Unknown(result, spec, "BEFORE_AFTER_PAIR_NOT_EXACT", "PAIR_NOT_EXACT", "PROVIDE_EXACT_BEFORE_AFTER_IDENTITY_PAIR", []string{spec.MeasurementID})
	}
	var before, after *int64
	if observations[0].Phase == "before" {
		before, after = observations[0].Value, observations[1].Value
	} else {
		before, after = observations[1].Value, observations[0].Value
	}
	delta := *after - *before
	result.Value = nil
	result.Improvement = V2Improvement{State: V2Closed, Reason: "EXACT_BEFORE_AFTER_IDENTITY_PAIR", PairID: observations[0].PairID, Before: before, After: after, Delta: &delta}
	return result
}

func v2ReceiptProblem(observation V2CollectedObservation, collection V2Collection) string {
	if !isValidDigest(observation.ReceiptDigest) {
		return "RECEIPT_CHAIN_BREAK"
	}
	for _, receipt := range collection.Receipts {
		if receipt.ReceiptDigest != observation.ReceiptDigest {
			continue
		}
		if receipt.Schema != V2ReceiptSchema || receipt.MetricID != observation.MetricID || receipt.StageID != observation.StageID || receipt.Stage != observation.Stage || receipt.Step != observation.Step || receipt.CausalEvents != (V2CausalEvents{Start: observation.StartEvent, End: observation.EndEvent}) || receipt.EndObserved != observation.EndObserved || receipt.StageEntered != observation.StageEntered || !v2ExactStrings(receipt.CoveredOperations, observation.CoveredOperations) || !v2ExactStrings(receipt.CoveredChildProcesses, observation.CoveredChildProcesses) || receipt.ChildProcessCoverage != observation.ChildProcessCoverage || receipt.Clock != observation.Clock || receipt.ResolutionMS != observation.ResolutionMS || receipt.RSSProcessTreeScope != observation.RSSProcessTreeScope || receipt.InputReceiptDigest != observation.InputReceiptDigest || receipt.OutputReceiptDigest != observation.OutputReceiptDigest || receipt.Unit != observation.Unit || receipt.Scope != observation.Scope || receipt.SourceAuthority != observation.SourceAuthority || !mapsEqual(receipt.IdentityDigests, observation.IdentityDigests) || receipt.SourceArtifact != observation.SourceArtifact || receipt.Measured != observation.Measured || !equalInt64(receipt.Value, observation.Value) || !equalInt64(receipt.WorkUnits, observation.WorkUnits) || !equalInt64(receipt.PeakRSSKiB, observation.PeakRSSKiB) || receipt.ObservationMethod != observation.ObservationMethod || receipt.Direction != observation.Direction || receipt.ExternalUtilityEvidence != observation.ExternalUtilityEvidence || receipt.OutputInsideReadOnlyInput != observation.OutputInsideReadOnlyInput || receipt.AuthorityEscalation != observation.AuthorityEscalation || receipt.Phase != observation.Phase || receipt.PairID != observation.PairID || receipt.ScenarioID != observation.ScenarioID || receipt.InputDigest != observation.InputDigest || receipt.ContractDigest != observation.ContractDigest || receipt.FixtureDigest != observation.FixtureDigest || receipt.Toolchain != observation.Toolchain || receipt.Runner != observation.Runner || receipt.Job != observation.Job {
			return "RECEIPT_CHAIN_BREAK"
		}
		if !isValidDigest(receipt.ObservationDigest) || receipt.ObservationDigest != receipt.ReceiptDigest {
			return "RECEIPT_CHAIN_BREAK"
		}
		return ""
	}
	return "RECEIPT_CHAIN_BREAK"
}

func equalInt64(left, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func v2ConsumerNames(observation V2CollectedObservation) []string {
	if len(observation.ConsumerArtifacts) > 0 {
		return append([]string(nil), observation.ConsumerArtifacts...)
	}
	return []string{observation.SourceArtifact}
}

func hasV2ExcludedOperation(spec V2MeasurementSpec, observation V2CollectedObservation) bool {
	for _, observed := range observation.CoveredOperations {
		for _, excluded := range spec.ExcludedOperations {
			if observed == excluded {
				return true
			}
		}
	}
	return false
}

func setV2Unknown(result V2MetricResult, spec V2MeasurementSpec, reason, class, next string, blocked []string) V2MetricResult {
	result.State = V2Unknown
	result.Reason = reason
	result.Value = nil
	result.Authority = ""
	result.Unknown = &V2UnknownFrontier{Stage: spec.Stage, Step: spec.Step, Reason: reason, UnknownClass: class, NextOperation: next, BlockedBy: uniqueStrings(blocked)}
	return result
}

func setV2Refuted(result V2MetricResult, spec V2MeasurementSpec, reason, next string, blocked []string) V2MetricResult {
	result.State = V2Refuted
	result.Reason = reason
	result.Value = nil
	result.Authority = ""
	result.Unknown = nil
	result.StageID = spec.StageID
	result.Stage = spec.Stage
	result.Step = spec.Step
	result.ConsumerArtifacts = uniqueStrings(append(result.ConsumerArtifacts, blocked...))
	return result
}

func buildV2Claim(decision V2Decision, results []V2MetricResult) V2Claim {
	ordered := append([]V2MetricResult(nil), results...)
	sort.SliceStable(ordered, func(i, j int) bool { return v2DecisionRank(ordered[i].State) < v2DecisionRank(ordered[j].State) })
	if len(ordered) == 0 {
		return V2Claim{State: V2Unknown, Reason: "NO_DECLARED_MEASUREMENTS", UnknownClass: "EMPTY_IR", NextOperation: "DECLARE_MEASUREMENT", BlockedBy: []string{}}
	}
	selected := ordered[0]
	claim := V2Claim{State: decision, StageID: selected.StageID, Stage: selected.Stage, Step: selected.Step, Reason: selected.Reason, BlockedBy: []string{}}
	if selected.Unknown != nil {
		claim.UnknownClass = selected.Unknown.UnknownClass
		claim.NextOperation = selected.Unknown.NextOperation
		claim.BlockedBy = selected.Unknown.BlockedBy
	}
	if decision == V2Closed {
		claim.NextOperation = "NONE"
	}
	return claim
}

func v2DecisionRank(decision V2Decision) int {
	switch decision {
	case V2Refuted:
		return 0
	case V2Unknown:
		return 1
	default:
		return 2
	}
}

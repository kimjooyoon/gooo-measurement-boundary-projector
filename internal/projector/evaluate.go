package projector

import (
	"fmt"
	"path/filepath"
	"sort"
)

func Evaluate(ir SemanticIR, collection Collection, outputPath string) (Evaluation, error) {
	if ir.Schema != IRScheme {
		return Evaluation{}, fmt.Errorf("unexpected semantic IR schema %q", ir.Schema)
	}
	computedIR, err := digestIR(ir)
	if err != nil {
		return Evaluation{}, err
	}
	if computedIR != ir.Digest {
		return Evaluation{}, fmt.Errorf("semantic IR digest mismatch: declared %s computed %s", ir.Digest, computedIR)
	}
	if collection.Schema != CollectionSchema {
		return Evaluation{}, fmt.Errorf("unexpected collection schema %q", collection.Schema)
	}
	computedCollection, err := digestCollection(collection)
	if err != nil {
		return Evaluation{}, err
	}
	if computedCollection != collection.Digest {
		return Evaluation{}, fmt.Errorf("collection digest mismatch: declared %s computed %s", collection.Digest, computedCollection)
	}
	if collection.IRDigest != ir.Digest {
		return Evaluation{}, fmt.Errorf("collection is bound to IR %s, expected %s", collection.IRDigest, ir.Digest)
	}

	byMetric := map[string][]CollectedObservation{}
	for _, observation := range collection.Observations {
		byMetric[observation.MetricID] = append(byMetric[observation.MetricID], observation)
	}
	results := make([]MetricResult, 0, len(ir.Measurements))
	for _, spec := range ir.Measurements {
		result := evaluateMetric(spec, byMetric[spec.MeasurementID], collection)
		results = append(results, result)
	}
	closedCount, unknownCount, refutedCount := 0, 0, 0
	for _, result := range results {
		switch result.State {
		case Closed:
			closedCount++
		case Unknown:
			unknownCount++
		case Refuted:
			refutedCount++
		}
	}
	decision := Closed
	if refutedCount > 0 {
		decision = Refuted
	} else if unknownCount > 0 {
		decision = Unknown
	}
	claim := buildClaim(decision, results)
	evaluation := Evaluation{
		Schema: EvaluationSchema, IRDigest: ir.Digest, CollectionDigest: collection.Digest,
		Decision: decision, FailClosed: decision != Closed, Claim: claim, Metrics: results,
		ClosedCount: closedCount, UnknownCount: unknownCount, RefutedCount: refutedCount,
		AggregatePolicy: "FORBID_UNSCOPED_SCALAR",
	}
	if outputPath != "" {
		if err := requireOutputOutside(outputPath, filepath.Dir(ir.SourcePath)); err != nil {
			return Evaluation{}, err
		}
		if err := WriteJSON(outputPath, evaluation); err != nil {
			return Evaluation{}, err
		}
	}
	return evaluation, nil
}

func evaluateMetric(spec MeasurementSpec, observations []CollectedObservation, collection Collection) MetricResult {
	result := MetricResult{
		MeasurementID: spec.MeasurementID, State: Closed, Stage: spec.Stage, Step: spec.Step,
		Reason: "EXACT_SINGLE_AUTHORITY_RECEIPT", Unit: spec.Unit, Scope: spec.Scope,
		Authority: spec.SourceAuthority, Authorities: make([]string, 0), Scopes: make([]string, 0),
		Units: make([]string, 0), IdentityDigests: spec.IdentityDigests,
		ReceiptDigests: make([]string, 0), ConsumerArtifacts: make([]string, 0),
		ObservedValues: make([]ObservedValue, 0),
	}
	if len(observations) == 0 {
		return setUnknown(result, spec, "MISSING_MEASUREMENT", "MISSING_MEASUREMENT", "COLLECT_DECLARED_MEASUREMENT", []string{spec.MeasurementID})
	}
	for _, observation := range observations {
		result.ObservedValues = append(result.ObservedValues, ObservedValue{
			Authority: observation.SourceAuthority, SourceArtifact: observation.SourceArtifact,
			Value: observation.Value, Unit: observation.Unit, Scope: observation.Scope,
			ReceiptDigest: observation.ReceiptDigest,
		})
		result.ReceiptDigests = append(result.ReceiptDigests, observation.ReceiptDigest)
		result.ConsumerArtifacts = append(result.ConsumerArtifacts, consumerNames(observation)...)
		result.Authorities = append(result.Authorities, observation.SourceAuthority)
		result.Scopes = append(result.Scopes, observation.Scope)
		result.Units = append(result.Units, observation.Unit)
		if reason := receiptProblem(observation, collection); reason != "" {
			return setRefuted(result, spec, reason, "RESTORE_RECEIPT_CHAIN", []string{observation.SourceArtifact})
		}
		if observation.Contradiction {
			return setRefuted(result, spec, "CONTRADICTORY_VALUE_ASSERTION", "REMOVE_CONTRADICTORY_ASSERTION", []string{observation.SourceArtifact})
		}
		if hasExcludedOperation(spec, observation) {
			return setRefuted(result, spec, "EXCLUDED_OPERATION_OBSERVED", "RESTORE_DECLARED_SPAN_BOUNDARY", []string{observation.SourceArtifact})
		}
	}
	result.Authorities = uniqueStrings(result.Authorities)
	result.Scopes = uniqueStrings(result.Scopes)
	result.Units = uniqueStrings(result.Units)
	result.ReceiptDigests = uniqueStrings(result.ReceiptDigests)
	result.ConsumerArtifacts = uniqueStrings(result.ConsumerArtifacts)
	switch {
	case len(result.Authorities) > 1:
		return setUnknown(result, spec, "MULTIPLE_SOURCE_AUTHORITIES", "DUPLICATE_AUTHORITY", "OBTAIN_SINGLE_AUTHORITY_MEASUREMENT", result.Authorities)
	case len(result.Scopes) > 1:
		return setUnknown(result, spec, "SCOPE_BOUNDARY_MISMATCH", "SCOPE_MISMATCH", "RECORD_ONE_EXACT_SCOPE", result.Scopes)
	case len(result.Units) > 1:
		return setUnknown(result, spec, "UNIT_BOUNDARY_MISMATCH", "UNIT_MISMATCH", "RECORD_ONE_EXACT_UNIT", result.Units)
	case len(result.ReceiptDigests) > 1 && len(observations) > 1 && !pairShape(observations):
		return setUnknown(result, spec, "MULTIPLE_RECEIPTS_FOR_ONE_METRIC", "DUPLICATE_OBSERVATION", "PUBLISH_ONE_RECEIPT_PER_METRIC", result.ReceiptDigests)
	}
	for _, observation := range observations {
		if observation.Stage != spec.Stage || observation.Step != spec.Step || observation.StartBoundary != spec.Span.StartBoundary || observation.EndBoundary != spec.Span.EndBoundary {
			return setUnknown(result, spec, "SPAN_BOUNDARY_MISMATCH", "SPAN_MISMATCH", "RECORD_DECLARED_START_END_BOUNDARIES", []string{observation.SourceArtifact})
		}
		if observation.Unit != spec.Unit {
			return setUnknown(result, spec, "DECLARED_UNIT_NOT_OBSERVED", "UNIT_MISMATCH", "RECORD_DECLARED_UNIT", []string{spec.Unit, observation.Unit})
		}
		if observation.Scope != spec.Scope {
			return setUnknown(result, spec, "DECLARED_SCOPE_NOT_OBSERVED", "SCOPE_MISMATCH", "RECORD_DECLARED_SCOPE", []string{spec.Scope, observation.Scope})
		}
		if !mapsEqual(observation.IdentityDigests, spec.IdentityDigests) {
			return setUnknown(result, spec, "IDENTITY_DIGEST_NOT_EXACT", "IDENTITY_MISMATCH", "RECORD_EXACT_IDENTITY_DIGESTS", []string{observation.SourceArtifact})
		}
		if externalMethodWithoutEvidence(observation) {
			return setUnknown(result, spec, "EXTERNAL_UTILITY_EVIDENCE_UNAVAILABLE", "EXTERNAL_EVIDENCE_MISSING", "OBTAIN_EXTERNAL_UTILITY_RECEIPT", []string{observation.SourceArtifact})
		}
		if observation.ObservationMethod != spec.ObservationMethod || observation.Direction != spec.Direction {
			return setUnknown(result, spec, "OBSERVATION_CONTRACT_MISMATCH", "METHOD_MISMATCH", "RECORD_DECLARED_OBSERVATION_METHOD", []string{observation.SourceArtifact})
		}
		if missingIncludedOperation(spec, observation) {
			return setUnknown(result, spec, "INCLUDED_OPERATION_NOT_OBSERVED", "MISSING_BOUNDARY_OPERATION", "OBSERVE_DECLARED_INCLUDED_OPERATIONS", []string{observation.SourceArtifact})
		}
		if !observation.Measured || observation.Value == nil {
			return setUnknown(result, spec, "MEASUREMENT_NOT_EXECUTED", "MISSING_MEASUREMENT", "EXECUTE_GENERATED_COLLECTOR", []string{observation.SourceArtifact})
		}
	}
	if !collection.Collector.Generated || !collection.Collector.MeasuredOnce {
		return setUnknown(result, spec, "GENERATED_COLLECTOR_NOT_SINGLE_MEASUREMENT", "COLLECTOR_NOT_CLOSED", "RUN_GENERATED_COLLECTOR_ONCE", []string{spec.MeasurementID})
	}
	if result.ConsumerArtifacts == nil || len(result.ConsumerArtifacts) == 0 {
		return setUnknown(result, spec, "NO_CONSUMER_RECEIPT_REFERENCE", "RECEIPT_REFERENCE_MISSING", "REFERENCE_THE_COLLECTOR_RECEIPT", []string{spec.MeasurementID})
	}
	allowedReceipts := map[string]bool{}
	for _, observation := range observations {
		allowedReceipts[observation.ReceiptDigest] = true
	}
	for _, consumer := range collection.Consumers {
		if consumer.MetricID == spec.MeasurementID && !allowedReceipts[consumer.ReceiptDigest] {
			return setRefuted(result, spec, "CONSUMER_RECEIPT_DIGEST_MISMATCH", "RESTORE_CONSUMER_RECEIPT_REFERENCE", []string{consumer.Name})
		}
	}
	if len(observations) == 1 {
		observation := observations[0]
		result.Value = observation.Value
		if observation.Phase != "" {
			return setUnknown(result, spec, "BEFORE_AFTER_PAIR_NOT_EXACT", "PAIR_NOT_EXACT", "PROVIDE_EXACT_BEFORE_AFTER_IDENTITY_PAIR", []string{observation.SourceArtifact})
		}
		return result
	}
	if !pairShape(observations) || observations[0].PairID == "" || observations[0].PairID != observations[1].PairID || observations[0].Phase == observations[1].Phase || !equalFloat(observations[0].Value, observations[1].Value) && observations[0].Value == nil && observations[1].Value == nil {
		return setUnknown(result, spec, "BEFORE_AFTER_PAIR_NOT_EXACT", "PAIR_NOT_EXACT", "PROVIDE_EXACT_BEFORE_AFTER_IDENTITY_PAIR", []string{spec.MeasurementID})
	}
	var before, after *float64
	if observations[0].Phase == "before" {
		before, after = observations[0].Value, observations[1].Value
	} else {
		before, after = observations[1].Value, observations[0].Value
	}
	if before == nil || after == nil {
		return setUnknown(result, spec, "BEFORE_AFTER_VALUE_MISSING", "MISSING_MEASUREMENT", "MEASURE_BOTH_PAIR_MEMBERS", []string{spec.MeasurementID})
	}
	delta := *after - *before
	result.Value = nil
	result.Before = before
	result.After = after
	result.Delta = &delta
	result.Reason = "EXACT_BEFORE_AFTER_IDENTITY_PAIR"
	return result
}

func pairShape(observations []CollectedObservation) bool {
	if len(observations) != 2 {
		return false
	}
	return (observations[0].Phase == "before" || observations[0].Phase == "after") && (observations[1].Phase == "before" || observations[1].Phase == "after")
}

func receiptProblem(observation CollectedObservation, collection Collection) string {
	if !isValidDigest(observation.ReceiptDigest) {
		return "RECEIPT_DIGEST_INVALID"
	}
	for _, receipt := range collection.Receipts {
		if receipt.ReceiptDigest != observation.ReceiptDigest {
			continue
		}
		if receipt.Schema != ReceiptSchema || receipt.MetricID != observation.MetricID || receipt.Stage != observation.Stage || receipt.Step != observation.Step || receipt.Unit != observation.Unit || receipt.Scope != observation.Scope || receipt.SourceAuthority != observation.SourceAuthority || receipt.SourceArtifact != observation.SourceArtifact || receipt.Measured != observation.Measured || !mapsEqual(receipt.IdentityDigests, observation.IdentityDigests) || !equalFloat(receipt.Value, observation.Value) {
			return "RECEIPT_PAYLOAD_MISMATCH"
		}
		if receipt.ObservationDigest != receipt.ReceiptDigest || !isValidDigest(receipt.ObservationDigest) {
			return "RECEIPT_DIGEST_CHAIN_INVALID"
		}
		return ""
	}
	return "RECEIPT_DIGEST_NOT_FOUND"
}

func equalFloat(left, right *float64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func consumerNames(observation CollectedObservation) []string {
	if len(observation.ConsumerArtifacts) > 0 {
		return append([]string(nil), observation.ConsumerArtifacts...)
	}
	return []string{observation.SourceArtifact}
}

func hasExcludedOperation(spec MeasurementSpec, observation CollectedObservation) bool {
	for _, observed := range observation.IncludedOperations {
		for _, excluded := range spec.ExcludedOperations {
			if observed == excluded {
				return true
			}
		}
	}
	return false
}

func missingIncludedOperation(spec MeasurementSpec, observation CollectedObservation) bool {
	observed := map[string]bool{}
	for _, operation := range observation.IncludedOperations {
		observed[operation] = true
	}
	for _, operation := range spec.IncludedOperations {
		if !observed[operation] {
			return true
		}
	}
	return false
}

func externalMethodWithoutEvidence(observation CollectedObservation) bool {
	return observation.ObservationMethod == "external-utility" && !observation.ExternalUtilityEvidence
}

func setUnknown(result MetricResult, spec MeasurementSpec, reason, class, next string, blocked []string) MetricResult {
	result.State = Unknown
	result.Reason = reason
	result.Value = nil
	result.Unknown = &UnknownFrontier{Stage: spec.Stage, Step: spec.Step, Reason: reason, UnknownClass: class, NextOperation: next, BlockedBy: uniqueStrings(blocked)}
	result.Authority = ""
	return result
}

func setRefuted(result MetricResult, spec MeasurementSpec, reason, next string, blocked []string) MetricResult {
	result.State = Refuted
	result.Reason = reason
	result.Value = nil
	result.Unknown = nil
	result.Authority = ""
	result.Stage = spec.Stage
	result.Step = spec.Step
	if len(blocked) > 0 {
		result.ConsumerArtifacts = uniqueStrings(append(result.ConsumerArtifacts, blocked...))
	}
	return result
}

func buildClaim(decision Decision, results []MetricResult) Claim {
	ordered := append([]MetricResult(nil), results...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return decisionRank(ordered[i].State) < decisionRank(ordered[j].State)
	})
	if len(ordered) == 0 {
		return Claim{State: Unknown, Reason: "NO_DECLARED_MEASUREMENTS", UnknownClass: "EMPTY_IR", NextOperation: "DECLARE_MEASUREMENT", BlockedBy: []string{}}
	}
	selected := ordered[0]
	claim := Claim{State: decision, Stage: selected.Stage, Step: selected.Step, Reason: selected.Reason, BlockedBy: []string{}}
	if selected.Unknown != nil {
		claim.UnknownClass = selected.Unknown.UnknownClass
		claim.NextOperation = selected.Unknown.NextOperation
		claim.BlockedBy = selected.Unknown.BlockedBy
	}
	if decision == Closed {
		claim.NextOperation = "NONE"
	}
	return claim
}

func decisionRank(decision Decision) int {
	switch decision {
	case Refuted:
		return 0
	case Unknown:
		return 1
	default:
		return 2
	}
}

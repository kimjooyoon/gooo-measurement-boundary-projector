package projector

import "testing"

func TestDecisionPrecedenceFailsClosed(t *testing.T) {
	claim := buildClaim(Unknown, []MetricResult{
		{MeasurementID: "closed", State: Closed, Stage: "s", Step: "closed", Reason: "ok"},
		{MeasurementID: "unknown", State: Unknown, Stage: "s", Step: "unknown", Reason: "missing", Unknown: &UnknownFrontier{Stage: "s", Step: "unknown", Reason: "missing", UnknownClass: "MISSING_MEASUREMENT", NextOperation: "collect", BlockedBy: []string{"x"}}},
		{MeasurementID: "refuted", State: Refuted, Stage: "s", Step: "refuted", Reason: "tampered"},
	})
	if claim.State != Unknown {
		t.Fatalf("expected unknown claim, got %s", claim.State)
	}
	claim = buildClaim(Refuted, []MetricResult{{State: Closed}, {State: Refuted, Stage: "s", Step: "r", Reason: "tampered"}})
	if claim.State != Refuted || claim.Reason != "tampered" {
		t.Fatalf("expected refuted claim, got %#v", claim)
	}
}

func TestExplicitZeroIsNotMissing(t *testing.T) {
	zero := float64(0)
	spec := MeasurementSpec{
		MeasurementID: "m", Stage: "report", Step: "verify", Span: Span{StartBoundary: "a", EndBoundary: "b"},
		IncludedOperations: []string{"report"}, Unit: "count", SourceAuthority: "runtime.json",
		ObservationMethod: "deterministic-fixture", Scope: "exact", IdentityDigests: map[string]string{"fixture": "sha256:x"},
		Direction: "lower_is_better", NullablePolicy: "null+UNKNOWN", ConflictPrecedence: []Decision{Refuted, Unknown, Closed},
	}
	observation := CollectedObservation{
		MetricID: "m", Stage: "report", Step: "verify", StartBoundary: "a", EndBoundary: "b", IncludedOperations: []string{"report"}, Unit: "count", SourceAuthority: "runtime.json", ObservationMethod: "deterministic-fixture", Scope: "exact", IdentityDigests: map[string]string{"fixture": "sha256:x"}, Direction: "lower_is_better", Measured: true, Value: &zero, SourceArtifact: "runtime.json", ConsumerArtifacts: []string{"runtime.json"}, ReceiptDigest: "sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	collection := Collection{
		Collector: CollectorEvidence{Generated: true, MeasuredOnce: true},
		Receipts: []Receipt{{Schema: ReceiptSchema, MetricID: "m", Stage: "report", Step: "verify", Unit: "count", Scope: "exact", SourceAuthority: "runtime.json", IdentityDigests: map[string]string{"fixture": "sha256:x"}, SourceArtifact: "runtime.json", Measured: true, Value: &zero, ObservationDigest: observation.ReceiptDigest, ReceiptDigest: observation.ReceiptDigest}},
	}
	result := evaluateMetric(spec, []CollectedObservation{observation}, collection)
	if result.State != Closed || result.Value == nil || *result.Value != 0 {
		t.Fatalf("explicit zero was not closed: %#v", result)
	}
}

func TestMissingMeasurementIsNullUnknown(t *testing.T) {
	spec := MeasurementSpec{MeasurementID: "m", Stage: "report", Step: "verify"}
	result := evaluateMetric(spec, nil, Collection{})
	if result.State != Unknown || result.Value != nil || result.Unknown == nil {
		t.Fatalf("missing measurement was not null+unknown: %#v", result)
	}
	if result.Unknown.Stage != "report" || result.Unknown.Step != "verify" || result.Unknown.Reason == "" || result.Unknown.UnknownClass == "" || result.Unknown.NextOperation == "" || result.Unknown.BlockedBy == nil {
		t.Fatalf("unknown frontier did not have six fields: %#v", result.Unknown)
	}
}

package projector

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type measurementPayload struct {
	MetricID                string            `json:"metric_id"`
	Stage                   string            `json:"stage"`
	Step                    string            `json:"step"`
	StartBoundary           string            `json:"start_boundary"`
	EndBoundary             string            `json:"end_boundary"`
	IncludedOperations      []string          `json:"included_operations"`
	Unit                    string            `json:"unit"`
	SourceAuthority         string            `json:"source_authority"`
	ObservationMethod       string            `json:"observation_method"`
	Scope                   string            `json:"scope"`
	IdentityDigests         map[string]string `json:"identity_digests"`
	Direction               string            `json:"direction"`
	Measured                bool              `json:"measured"`
	Value                   *float64          `json:"value"`
	SourceArtifact          string            `json:"source_artifact"`
	ExternalUtilityEvidence bool              `json:"external_utility_evidence"`
	Contradiction           bool              `json:"contradiction"`
	Phase                   string            `json:"phase,omitempty"`
	PairID                  string            `json:"pair_id,omitempty"`
}

func CollectFixture(ir SemanticIR, fixturePath, outputDir string) (Collection, Fixture, error) {
	if err := requireOutputOutside(outputDir, filepath.Dir(fixturePath)); err != nil {
		return Collection{}, Fixture{}, err
	}
	fixtureDigest, data, err := DigestFile(fixturePath)
	if err != nil {
		return Collection{}, Fixture{}, err
	}
	var fixture Fixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return Collection{}, Fixture{}, err
	}
	if fixture.Schema != "gooo/measurement-boundary/fixture/v1" {
		return Collection{}, Fixture{}, fmt.Errorf("fixture %q has schema %q", fixture.CaseID, fixture.Schema)
	}
	declared := map[string]MeasurementSpec{}
	for _, spec := range ir.Measurements {
		declared[spec.MeasurementID] = spec
	}
	counts := map[string]int{}
	for _, sample := range fixture.Samples {
		if _, ok := declared[sample.MetricID]; !ok {
			return Collection{}, Fixture{}, fmt.Errorf("fixture %q refers to undeclared metric %q", fixture.CaseID, sample.MetricID)
		}
		counts[sample.MetricID]++
	}
	measuredOnce := true
	for _, spec := range ir.Measurements {
		if counts[spec.MeasurementID] == 0 {
			measuredOnce = false
		}
	}
	collectorDigest, err := DigestJSON(map[string]string{"kind": "generated-collector", "ir_digest": ir.Digest})
	if err != nil {
		return Collection{}, Fixture{}, err
	}
	collection := Collection{
		Schema:        CollectionSchema,
		IRDigest:      ir.Digest,
		FixtureDigest: fixtureDigest,
		Collector: CollectorEvidence{
			Kind: "generated-collector", Generated: true, MeasuredOnce: measuredOnce,
			IdentityDigest: collectorDigest, OutputScope: "caller-owned-temp-output-only",
			RepositoryWrites: 0, ApplyAuthority: 0, CommitAuthority: 0,
			MergeAuthority: 0, TagAuthority: 0, ReleaseAuthority: 0,
		},
		Observations: make([]CollectedObservation, 0, len(fixture.Samples)),
		Receipts:     make([]Receipt, 0, len(fixture.Samples)),
		Consumers:    make([]ConsumerArtifact, 0),
	}
	for _, sample := range fixture.Samples {
		payload := measurementPayload{
			MetricID: sample.MetricID, Stage: sample.Stage, Step: sample.Step,
			StartBoundary: sample.StartBoundary, EndBoundary: sample.EndBoundary,
			IncludedOperations: sample.IncludedOperations, Unit: sample.Unit,
			SourceAuthority: sample.SourceAuthority, ObservationMethod: sample.ObservationMethod,
			Scope: sample.Scope, IdentityDigests: sample.IdentityDigests, Direction: sample.Direction,
			Measured: sample.Measured, Value: sample.Value, SourceArtifact: sample.SourceArtifact,
			ExternalUtilityEvidence: sample.ExternalUtilityEvidence, Contradiction: sample.Contradiction,
			Phase: sample.Phase, PairID: sample.PairID,
		}
		observationDigest, err := DigestJSON(payload)
		if err != nil {
			return Collection{}, Fixture{}, err
		}
		reportedDigest := observationDigest
		if sample.TamperReceipt {
			reportedDigest = "sha256:" + strings.Repeat("0", 64)
		}
		collection.Receipts = append(collection.Receipts, Receipt{
			Schema: ReceiptSchema, MetricID: sample.MetricID, Stage: sample.Stage, Step: sample.Step,
			Unit: sample.Unit, Scope: sample.Scope, SourceAuthority: sample.SourceAuthority,
			IdentityDigests: sample.IdentityDigests, SourceArtifact: sample.SourceArtifact,
			Measured: sample.Measured, Value: sample.Value, ObservationDigest: observationDigest,
			ReceiptDigest: observationDigest,
		})
		collection.Observations = append(collection.Observations, CollectedObservation{
			MetricID: sample.MetricID, Stage: sample.Stage, Step: sample.Step,
			StartBoundary: sample.StartBoundary, EndBoundary: sample.EndBoundary,
			IncludedOperations: sample.IncludedOperations, Unit: sample.Unit,
			SourceAuthority: sample.SourceAuthority, ObservationMethod: sample.ObservationMethod,
			Scope: sample.Scope, IdentityDigests: sample.IdentityDigests, Direction: sample.Direction,
			Measured: sample.Measured, Value: sample.Value, SourceArtifact: sample.SourceArtifact,
			ConsumerArtifacts: sample.ConsumerArtifacts, ExternalUtilityEvidence: sample.ExternalUtilityEvidence,
			Contradiction: sample.Contradiction, Phase: sample.Phase, PairID: sample.PairID,
			ReceiptDigest: reportedDigest,
		})
		consumers := sample.ConsumerArtifacts
		if len(consumers) == 0 {
			consumers = []string{sample.SourceArtifact}
		}
		for _, name := range consumers {
			consumerDigest := reportedDigest
			if sample.TamperConsumer {
				consumerDigest = "sha256:" + strings.Repeat("f", 64)
			}
			collection.Consumers = append(collection.Consumers, ConsumerArtifact{Name: name, MetricID: sample.MetricID, ReceiptDigest: consumerDigest})
		}
	}
	sort.Slice(collection.Observations, func(i, j int) bool {
		if collection.Observations[i].MetricID != collection.Observations[j].MetricID {
			return collection.Observations[i].MetricID < collection.Observations[j].MetricID
		}
		return collection.Observations[i].SourceArtifact < collection.Observations[j].SourceArtifact
	})
	collection.Digest, err = digestCollection(collection)
	if err != nil {
		return Collection{}, Fixture{}, err
	}
	if err := WriteJSON(filepath.Join(outputDir, "collection.json"), collection); err != nil {
		return Collection{}, Fixture{}, err
	}
	for index, receipt := range collection.Receipts {
		if err := WriteJSON(filepath.Join(outputDir, "receipts", fmt.Sprintf("receipt-%03d.json", index+1)), receipt); err != nil {
			return Collection{}, Fixture{}, err
		}
	}
	return collection, fixture, nil
}

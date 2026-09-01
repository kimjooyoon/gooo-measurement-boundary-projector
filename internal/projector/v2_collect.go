package projector

import (
	"fmt"
	"path/filepath"
)

type v2ObservationPayload struct {
	MetricID                  string            `json:"metric_id"`
	StageID                   string            `json:"stage_id"`
	Stage                     string            `json:"stage"`
	Step                      string            `json:"step"`
	StartEvent                string            `json:"start_event"`
	EndEvent                  string            `json:"end_event"`
	EndObserved               bool              `json:"end_observed"`
	StageEntered              bool              `json:"stage_entered"`
	CoveredOperations         []string          `json:"covered_operations"`
	CoveredChildProcesses     []string          `json:"covered_child_processes"`
	ChildProcessCoverage      string            `json:"child_process_coverage"`
	Clock                     string            `json:"clock"`
	ResolutionMS              int64             `json:"resolution_ms"`
	RSSProcessTreeScope       string            `json:"rss_process_tree_scope"`
	InputReceiptDigest        string            `json:"input_receipt_digest"`
	OutputReceiptDigest       string            `json:"output_receipt_digest"`
	Unit                      string            `json:"unit"`
	SourceAuthority           string            `json:"source_authority"`
	ObservationMethod         string            `json:"observation_method"`
	Scope                     string            `json:"scope"`
	IdentityDigests           map[string]string `json:"identity_digests"`
	Direction                 string            `json:"direction"`
	Measured                  bool              `json:"measured"`
	Value                     *int64            `json:"value"`
	WorkUnits                 *int64            `json:"work_units"`
	PeakRSSKiB                *int64            `json:"peak_rss_kib"`
	SourceArtifact            string            `json:"source_artifact"`
	ExternalUtilityEvidence   bool              `json:"external_utility_evidence"`
	OutputInsideReadOnlyInput bool              `json:"output_inside_read_only_input"`
	AuthorityEscalation       bool              `json:"authority_escalation"`
	Phase                     string            `json:"phase,omitempty"`
	PairID                    string            `json:"pair_id,omitempty"`
	ScenarioID                string            `json:"scenario_id,omitempty"`
	InputDigest               string            `json:"input_digest,omitempty"`
	ContractDigest            string            `json:"contract_digest,omitempty"`
	FixtureDigest             string            `json:"fixture_digest,omitempty"`
	Toolchain                 string            `json:"toolchain,omitempty"`
	Runner                    string            `json:"runner,omitempty"`
	Job                       string            `json:"job,omitempty"`
}

func v2Payload(sample V2Sample) v2ObservationPayload {
	return v2ObservationPayload{
		MetricID: sample.MetricID, StageID: sample.StageID, Stage: sample.Stage, Step: sample.Step,
		StartEvent: sample.StartEvent, EndEvent: sample.EndEvent, EndObserved: sample.EndObserved,
		StageEntered: sample.StageEntered, CoveredOperations: sample.CoveredOperations,
		CoveredChildProcesses: sample.CoveredChildProcesses, ChildProcessCoverage: sample.ChildProcessCoverage,
		Clock: sample.Clock, ResolutionMS: sample.ResolutionMS, RSSProcessTreeScope: sample.RSSProcessTreeScope,
		InputReceiptDigest: sample.InputReceiptDigest, OutputReceiptDigest: sample.OutputReceiptDigest,
		Unit: sample.Unit, SourceAuthority: sample.SourceAuthority, ObservationMethod: sample.ObservationMethod,
		Scope: sample.Scope, IdentityDigests: sample.IdentityDigests, Direction: sample.Direction,
		Measured: sample.Measured, Value: sample.Value, WorkUnits: sample.WorkUnits, PeakRSSKiB: sample.PeakRSSKiB,
		SourceArtifact: sample.SourceArtifact, ExternalUtilityEvidence: sample.ExternalUtilityEvidence,
		OutputInsideReadOnlyInput: sample.OutputInsideReadOnlyInput, AuthorityEscalation: sample.AuthorityEscalation,
		Phase: sample.Phase, PairID: sample.PairID, ScenarioID: sample.ScenarioID, InputDigest: sample.InputDigest,
		ContractDigest: sample.ContractDigest, FixtureDigest: sample.FixtureDigest, Toolchain: sample.Toolchain,
		Runner: sample.Runner, Job: sample.Job,
	}
}

func CollectV2Fixture(ir V2SemanticIR, fixturePath, outputDir string) (V2Collection, V2Fixture, error) {
	if err := requireOutputOutside(outputDir, filepath.Dir(fixturePath)); err != nil {
		return V2Collection{}, V2Fixture{}, err
	}
	fixtureDigest, _, err := DigestFile(fixturePath)
	if err != nil {
		return V2Collection{}, V2Fixture{}, err
	}
	var fixture V2Fixture
	if err := LoadJSON(fixturePath, &fixture); err != nil {
		return V2Collection{}, V2Fixture{}, err
	}
	if fixture.Schema != V2FixtureSchema {
		return V2Collection{}, V2Fixture{}, fmt.Errorf("fixture %q has schema %q", fixture.CaseID, fixture.Schema)
	}
	declared := map[string]V2MeasurementSpec{}
	for _, spec := range ir.Measurements {
		declared[spec.MeasurementID] = spec
	}
	counts := map[string]int{}
	for _, sample := range fixture.Samples {
		if _, ok := declared[sample.MetricID]; !ok {
			return V2Collection{}, V2Fixture{}, fmt.Errorf("fixture %q refers to undeclared v2 metric %q", fixture.CaseID, sample.MetricID)
		}
		counts[sample.MetricID]++
	}
	measuredOnce := true
	for _, spec := range ir.Measurements {
		if counts[spec.MeasurementID] == 0 {
			measuredOnce = false
		}
	}
	authorities := uniqueStrings(fixture.CollectorAuthorities)
	if len(authorities) == 0 {
		authorities = []string{"generated-stage-collector"}
	}
	collectorAuthority := authorities[0]
	if len(authorities) > 1 {
		collectorAuthority = "competing-collectors"
	}
	collectorDigest, err := DigestV2JSON(map[string]any{
		"kind": "generated-stage-collector", "ir_digest": ir.Digest, "authorities": authorities,
	})
	if err != nil {
		return V2Collection{}, V2Fixture{}, err
	}
	collection := V2Collection{
		Schema: V2CollectionSchema, IRDigest: ir.Digest, FixtureDigest: fixtureDigest,
		Collector: V2CollectorEvidence{
			Kind: "generated-stage-collector", Generated: true, MeasuredOnce: measuredOnce,
			Authority: collectorAuthority, Authorities: authorities, IdentityDigest: collectorDigest,
			InputScope: "repository-read-only", OutputScope: "caller-owned-temp-output-only",
			OperatorAuthority: "authoring-only", RuntimeAuthority: fixture.RuntimeAuthority,
		},
		Observations: make([]V2CollectedObservation, 0, len(fixture.Samples)),
		Receipts:     make([]V2Receipt, 0, len(fixture.Samples)), Consumers: make([]V2ConsumerArtifact, 0),
	}
	for _, sample := range fixture.Samples {
		payload := v2Payload(sample)
		observationDigest, err := DigestV2JSON(payload)
		if err != nil {
			return V2Collection{}, V2Fixture{}, err
		}
		reportedDigest := observationDigest
		if sample.TamperReceipt {
			reportedDigest = "sha256:" + stringsRepeat("0", 64)
		}
		collection.Receipts = append(collection.Receipts, V2Receipt{
			Schema: V2ReceiptSchema, MetricID: sample.MetricID, StageID: sample.StageID, Stage: sample.Stage,
			Step: sample.Step, CausalEvents: V2CausalEvents{Start: sample.StartEvent, End: sample.EndEvent},
			EndObserved: sample.EndObserved, StageEntered: sample.StageEntered, CoveredOperations: sample.CoveredOperations,
			CoveredChildProcesses: sample.CoveredChildProcesses, ChildProcessCoverage: sample.ChildProcessCoverage,
			Clock: sample.Clock, ResolutionMS: sample.ResolutionMS, RSSProcessTreeScope: sample.RSSProcessTreeScope,
			InputReceiptDigest: sample.InputReceiptDigest, OutputReceiptDigest: sample.OutputReceiptDigest,
			Unit: sample.Unit, Scope: sample.Scope, SourceAuthority: sample.SourceAuthority,
			IdentityDigests: sample.IdentityDigests, SourceArtifact: sample.SourceArtifact, Measured: sample.Measured,
			Value: sample.Value, WorkUnits: sample.WorkUnits, PeakRSSKiB: sample.PeakRSSKiB,
			ObservationMethod: sample.ObservationMethod, Direction: sample.Direction,
			ExternalUtilityEvidence:   sample.ExternalUtilityEvidence,
			OutputInsideReadOnlyInput: sample.OutputInsideReadOnlyInput, AuthorityEscalation: sample.AuthorityEscalation,
			ScenarioID: sample.ScenarioID, InputDigest: sample.InputDigest, ContractDigest: sample.ContractDigest,
			FixtureDigest: sample.FixtureDigest, Toolchain: sample.Toolchain, Runner: sample.Runner, Job: sample.Job,
			Phase: sample.Phase, PairID: sample.PairID,
			ObservationDigest: observationDigest, ReceiptDigest: observationDigest,
		})
		collection.Observations = append(collection.Observations, V2CollectedObservation{
			MetricID: sample.MetricID, StageID: sample.StageID, Stage: sample.Stage, Step: sample.Step,
			StartEvent: sample.StartEvent, EndEvent: sample.EndEvent, EndObserved: sample.EndObserved,
			StageEntered: sample.StageEntered, CoveredOperations: sample.CoveredOperations,
			CoveredChildProcesses: sample.CoveredChildProcesses, ChildProcessCoverage: sample.ChildProcessCoverage,
			Clock: sample.Clock, ResolutionMS: sample.ResolutionMS, RSSProcessTreeScope: sample.RSSProcessTreeScope,
			InputReceiptDigest: sample.InputReceiptDigest, OutputReceiptDigest: sample.OutputReceiptDigest,
			Unit: sample.Unit, SourceAuthority: sample.SourceAuthority, ObservationMethod: sample.ObservationMethod,
			Scope: sample.Scope, IdentityDigests: sample.IdentityDigests, Direction: sample.Direction,
			Measured: sample.Measured, Value: sample.Value, WorkUnits: sample.WorkUnits, PeakRSSKiB: sample.PeakRSSKiB,
			SourceArtifact: sample.SourceArtifact, ConsumerArtifacts: sample.ConsumerArtifacts,
			ExternalUtilityEvidence:   sample.ExternalUtilityEvidence,
			OutputInsideReadOnlyInput: sample.OutputInsideReadOnlyInput, AuthorityEscalation: sample.AuthorityEscalation,
			Phase: sample.Phase, PairID: sample.PairID, ScenarioID: sample.ScenarioID, InputDigest: sample.InputDigest,
			ContractDigest: sample.ContractDigest, FixtureDigest: sample.FixtureDigest, Toolchain: sample.Toolchain,
			Runner: sample.Runner, Job: sample.Job, ReceiptDigest: reportedDigest,
		})
		consumers := sample.ConsumerArtifacts
		if len(consumers) == 0 {
			consumers = []string{sample.SourceArtifact}
		}
		for _, name := range consumers {
			consumerDigest := reportedDigest
			if sample.TamperConsumer {
				consumerDigest = "sha256:" + stringsRepeat("f", 64)
			}
			collection.Consumers = append(collection.Consumers, V2ConsumerArtifact{
				Name: name, MetricID: sample.MetricID, StageID: sample.StageID,
				CoveredOperations: sample.CoveredOperations, ReceiptDigest: consumerDigest,
			})
		}
	}
	v2SortedObservations(collection.Observations)
	collection.Digest, err = digestV2Collection(collection)
	if err != nil {
		return V2Collection{}, V2Fixture{}, err
	}
	if err := WriteJSON(filepath.Join(outputDir, "collection.json"), collection); err != nil {
		return V2Collection{}, V2Fixture{}, err
	}
	for index, receipt := range collection.Receipts {
		if err := WriteJSON(filepath.Join(outputDir, "receipts", fmt.Sprintf("receipt-%03d.json", index+1)), receipt); err != nil {
			return V2Collection{}, V2Fixture{}, err
		}
	}
	return collection, fixture, nil
}

func stringsRepeat(value string, count int) string {
	result := ""
	for index := 0; index < count; index++ {
		result += value
	}
	return result
}

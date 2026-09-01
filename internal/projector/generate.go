package projector

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
)

func Compile(sourcePath, outputDir string) (SemanticIR, error) {
	if err := requireOutputOutside(outputDir, filepath.Dir(sourcePath)); err != nil {
		return SemanticIR{}, err
	}
	ir, err := ParseSource(sourcePath)
	if err != nil {
		return SemanticIR{}, err
	}
	if err := WriteJSON(filepath.Join(outputDir, "semantic-ir.json"), ir); err != nil {
		return SemanticIR{}, err
	}
	collector, err := renderGeneratedCollector(ir)
	if err != nil {
		return SemanticIR{}, err
	}
	if err := WriteText(filepath.Join(outputDir, "generated", "collector.go"), collector); err != nil {
		return SemanticIR{}, err
	}
	wrapper := "#!/usr/bin/env bash\nset -Eeuo pipefail\nfixture=${1:?fixture path is required}\nout=${2:?output directory is required}\nexec go run \"$(dirname \"$0\")/collector.go\" --fixture \"$fixture\" --out \"$out\"\n"
	if err := WriteText(filepath.Join(outputDir, "generated", "collect.sh"), wrapper); err != nil {
		return SemanticIR{}, err
	}
	return ir, nil
}

func renderGeneratedCollector(ir SemanticIR) (string, error) {
	data, err := json.Marshal(ir)
	if err != nil {
		return "", err
	}
	template := `package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const embeddedIR = __EMBEDDED_IR__

type measurement struct {
	MeasurementID string ` + "`json:\"measurement_id\"`" + `
	Stage string ` + "`json:\"stage\"`" + `
	Step string ` + "`json:\"step\"`" + `
	Unit string ` + "`json:\"unit\"`" + `
	Scope string ` + "`json:\"scope\"`" + `
	SourceAuthority string ` + "`json:\"source_authority\"`" + `
}

type irFile struct {
	Digest string ` + "`json:\"digest\"`" + `
	Measurements []measurement ` + "`json:\"measurements\"`" + `
}

type fixture struct {
	Schema string ` + "`json:\"schema\"`" + `
	CaseID string ` + "`json:\"case_id\"`" + `
	Name string ` + "`json:\"name\"`" + `
	Samples []sample ` + "`json:\"samples\"`" + `
}

type sample struct {
	MetricID string ` + "`json:\"metric_id\"`" + `
	Stage string ` + "`json:\"stage\"`" + `
	Step string ` + "`json:\"step\"`"` + `
	StartBoundary string ` + "`json:\"start_boundary\"`" + `
	EndBoundary string ` + "`json:\"end_boundary\"`" + `
	IncludedOperations []string ` + "`json:\"included_operations\"`" + `
	Unit string ` + "`json:\"unit\"`" + `
	SourceAuthority string ` + "`json:\"source_authority\"`" + `
	ObservationMethod string ` + "`json:\"observation_method\"`" + `
	Scope string ` + "`json:\"scope\"`" + `
	IdentityDigests map[string]string ` + "`json:\"identity_digests\"`" + `
	Direction string ` + "`json:\"direction\"`" + `
	Measured bool ` + "`json:\"measured\"`" + `
	Value *float64 ` + "`json:\"value\"`" + `
	SourceArtifact string ` + "`json:\"source_artifact\"`" + `
	ConsumerArtifacts []string ` + "`json:\"consumer_artifacts\"`" + `
	ExternalUtilityEvidence bool ` + "`json:\"external_utility_evidence\"`" + `
	TamperReceipt bool ` + "`json:\"tamper_receipt\"`" + `
	TamperConsumer bool ` + "`json:\"tamper_consumer\"`" + `
	Contradiction bool ` + "`json:\"contradiction\"`" + `
	Phase string ` + "`json:\"phase,omitempty\"`" + `
	PairID string ` + "`json:\"pair_id,omitempty\"`" + `
}

type observation struct {
	MetricID string ` + "`json:\"metric_id\"`" + `
	Stage string ` + "`json:\"stage\"`"` + `
	Step string ` + "`json:\"step\"`" + `
	StartBoundary string ` + "`json:\"start_boundary\"`" + `
	EndBoundary string ` + "`json:\"end_boundary\"`" + `
	IncludedOperations []string ` + "`json:\"included_operations\"`" + `
	Unit string ` + "`json:\"unit\"`" + `
	SourceAuthority string ` + "`json:\"source_authority\"`" + `
	ObservationMethod string ` + "`json:\"observation_method\"`" + `
	Scope string ` + "`json:\"scope\"`" + `
	IdentityDigests map[string]string ` + "`json:\"identity_digests\"`" + `
	Direction string ` + "`json:\"direction\"`" + `
	Measured bool ` + "`json:\"measured\"`" + `
	Value *float64 ` + "`json:\"value\"`" + `
	SourceArtifact string ` + "`json:\"source_artifact\"`" + `
	ConsumerArtifacts []string ` + "`json:\"consumer_artifacts\"`" + `
	ExternalUtilityEvidence bool ` + "`json:\"external_utility_evidence\"`" + `
	Contradiction bool ` + "`json:\"contradiction\"`" + `
	Phase string ` + "`json:\"phase,omitempty\"`" + `
	PairID string ` + "`json:\"pair_id,omitempty\"`" + `
	ReceiptDigest string ` + "`json:\"receipt_digest\"`" + `
}

type receipt struct {
	Schema string ` + "`json:\"schema\"`" + `
	MetricID string ` + "`json:\"metric_id\"`" + `
	Stage string ` + "`json:\"stage\"`"` + `
	Step string ` + "`json:\"step\"`" + `
	Unit string ` + "`json:\"unit\"`" + `
	Scope string ` + "`json:\"scope\"`" + `
	SourceAuthority string ` + "`json:\"source_authority\"`" + `
	IdentityDigests map[string]string ` + "`json:\"identity_digests\"`" + `
	SourceArtifact string ` + "`json:\"source_artifact\"`" + `
	Measured bool ` + "`json:\"measured\"`" + `
	Value *float64 ` + "`json:\"value\"`" + `
	ObservationDigest string ` + "`json:\"observation_digest\"`" + `
	ReceiptDigest string ` + "`json:\"receipt_digest\"`" + `
}

type consumer struct {
	Name string ` + "`json:\"name\"`" + `
	MetricID string ` + "`json:\"metric_id\"`"` + `
	ReceiptDigest string ` + "`json:\"receipt_digest\"`" + `
}

type collection struct {
	Schema string ` + "`json:\"schema\"`" + `
	IRDigest string ` + "`json:\"ir_digest\"`" + `
	FixtureDigest string ` + "`json:\"fixture_digest\"`" + `
	Collector collectorEvidence ` + "`json:\"collector\"`" + `
	Observations []observation ` + "`json:\"observations\"`" + `
	Receipts []receipt ` + "`json:\"receipts\"`" + `
	Consumers []consumer ` + "`json:\"consumers\"`" + `
	Digest string ` + "`json:\"digest\"`" + `
}

type collectorEvidence struct {
	Kind string ` + "`json:\"kind\"`" + `
	Generated bool ` + "`json:\"generated\"`" + `
	MeasuredOnce bool ` + "`json:\"measured_once\"`" + `
	IdentityDigest string ` + "`json:\"identity_digest\"`" + `
	OutputScope string ` + "`json:\"output_scope\"`" + `
	RepositoryWrites int ` + "`json:\"repository_writes\"`"` + `
	ApplyAuthority int ` + "`json:\"apply_authority\"`" + `
	CommitAuthority int ` + "`json:\"commit_authority\"`" + `
	MergeAuthority int ` + "`json:\"merge_authority\"`" + `
	TagAuthority int ` + "`json:\"tag_authority\"`" + `
	ReleaseAuthority int ` + "`json:\"release_authority\"`" + `
}

func digest(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func main() {
	fixturePath := flag.String("fixture", "", "deterministic fixture path")
	outDir := flag.String("out", "", "caller-owned output directory")
	flag.Parse()
	if *fixturePath == "" || *outDir == "" {
		fmt.Fprintln(os.Stderr, "fixture and out are required")
		os.Exit(64)
	}
	if err := run(*fixturePath, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(fixturePath, outDir string) error {
	fixtureBytes, err := os.ReadFile(fixturePath)
	if err != nil {
		return err
	}
	var input fixture
	if err := json.Unmarshal(fixtureBytes, &input); err != nil {
		return err
	}
	var ir irFile
	if err := json.Unmarshal([]byte(embeddedIR), &ir); err != nil {
		return err
	}
	specs := map[string]measurement{}
	for _, item := range ir.Measurements {
		specs[item.MeasurementID] = item
	}
	counts := map[string]int{}
	for _, item := range input.Samples {
		if _, ok := specs[item.MetricID]; !ok {
			return fmt.Errorf("fixture %q refers to undeclared metric %q", input.CaseID, item.MetricID)
		}
		counts[item.MetricID]++
	}
	measuredOnce := true
	for _, item := range ir.Measurements {
		if counts[item.MeasurementID] == 0 {
			measuredOnce = false
		}
	}
	collectorDigest, err := digest(map[string]string{"kind": "generated-collector", "ir_digest": ir.Digest})
	if err != nil {
		return err
	}
	result := collection{
		Schema: "gooo/measurement-boundary/collection/v1",
		IRDigest: ir.Digest,
		FixtureDigest: digestBytes(fixtureBytes),
		Collector: collectorEvidence{
			Kind: "generated-collector", Generated: true, MeasuredOnce: measuredOnce,
			IdentityDigest: collectorDigest, OutputScope: "caller-owned-temp-output-only",
			RepositoryWrites: 0, ApplyAuthority: 0, CommitAuthority: 0, MergeAuthority: 0,
			TagAuthority: 0, ReleaseAuthority: 0,
		},
		Observations: make([]observation, 0, len(input.Samples)),
		Receipts: make([]receipt, 0, len(input.Samples)),
		Consumers: make([]consumer, 0),
	}
	for _, item := range input.Samples {
		payload := struct {
			MetricID string ` + "`json:\"metric_id\"`" + `
			Stage string ` + "`json:\"stage\"`" + `
			Step string ` + "`json:\"step\"`" + `
			StartBoundary string ` + "`json:\"start_boundary\"`" + `
			EndBoundary string ` + "`json:\"end_boundary\"`" + `
			IncludedOperations []string ` + "`json:\"included_operations\"`" + `
			Unit string ` + "`json:\"unit\"`" + `
			SourceAuthority string ` + "`json:\"source_authority\"`" + `
			ObservationMethod string ` + "`json:\"observation_method\"`" + `
			Scope string ` + "`json:\"scope\"`" + `
			IdentityDigests map[string]string ` + "`json:\"identity_digests\"`" + `
			Direction string ` + "`json:\"direction\"`" + `
			Measured bool ` + "`json:\"measured\"`" + `
			Value *float64 ` + "`json:\"value\"`" + `
			SourceArtifact string ` + "`json:\"source_artifact\"`" + `
			ExternalUtilityEvidence bool ` + "`json:\"external_utility_evidence\"`" + `
			Contradiction bool ` + "`json:\"contradiction\"`" + `
			Phase string ` + "`json:\"phase,omitempty\"`" + `
			PairID string ` + "`json:\"pair_id,omitempty\"`" + `
		}{item.MetricID, item.Stage, item.Step, item.StartBoundary, item.EndBoundary, item.IncludedOperations, item.Unit, item.SourceAuthority, item.ObservationMethod, item.Scope, item.IdentityDigests, item.Direction, item.Measured, item.Value, item.SourceArtifact, item.ExternalUtilityEvidence, item.Contradiction, item.Phase, item.PairID}
		observationDigest, err := digest(payload)
		if err != nil {
			return err
		}
		reportedReceiptDigest := observationDigest
		if item.TamperReceipt {
			reportedReceiptDigest = "sha256:" + strings.Repeat("0", 64)
		}
		result.Receipts = append(result.Receipts, receipt{
			Schema: "gooo/measurement-boundary/receipt/v1", MetricID: item.MetricID,
			Stage: item.Stage, Step: item.Step, Unit: item.Unit, Scope: item.Scope,
			SourceAuthority: item.SourceAuthority, IdentityDigests: item.IdentityDigests,
			SourceArtifact: item.SourceArtifact, Measured: item.Measured, Value: item.Value,
			ObservationDigest: observationDigest, ReceiptDigest: observationDigest,
		})
		result.Observations = append(result.Observations, observation{
			MetricID: item.MetricID, Stage: item.Stage, Step: item.Step,
			StartBoundary: item.StartBoundary, EndBoundary: item.EndBoundary,
			IncludedOperations: item.IncludedOperations, Unit: item.Unit,
			SourceAuthority: item.SourceAuthority, ObservationMethod: item.ObservationMethod,
			Scope: item.Scope, IdentityDigests: item.IdentityDigests, Direction: item.Direction,
			Measured: item.Measured, Value: item.Value, SourceArtifact: item.SourceArtifact,
			ConsumerArtifacts: item.ConsumerArtifacts, ExternalUtilityEvidence: item.ExternalUtilityEvidence,
			Contradiction: item.Contradiction, Phase: item.Phase, PairID: item.PairID,
			ReceiptDigest: reportedReceiptDigest,
		})
		consumers := item.ConsumerArtifacts
		if len(consumers) == 0 {
			consumers = []string{item.SourceArtifact}
		}
		for _, name := range consumers {
			consumerDigest := reportedReceiptDigest
			if item.TamperConsumer {
				consumerDigest = "sha256:" + strings.Repeat("f", 64)
			}
			result.Consumers = append(result.Consumers, consumer{Name: name, MetricID: item.MetricID, ReceiptDigest: consumerDigest})
		}
	}
	sort.Slice(result.Observations, func(i, j int) bool {
		if result.Observations[i].MetricID != result.Observations[j].MetricID {
			return result.Observations[i].MetricID < result.Observations[j].MetricID
		}
		return result.Observations[i].SourceArtifact < result.Observations[j].SourceArtifact
	})
	result.Digest, err = digest(result)
	if err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "collection.json"), result); err != nil {
		return err
	}
	for index, item := range result.Receipts {
		if err := writeJSON(filepath.Join(outDir, "receipts", fmt.Sprintf("receipt-%03d.json", index+1)), item); err != nil {
			return err
		}
	}
	return nil
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
`
	template = strings.ReplaceAll(template, "__EMBEDDED_IR__", strconv.Quote(string(data)))
	return template, nil
}

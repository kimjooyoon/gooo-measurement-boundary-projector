package projector

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
)

func CompileV2(sourcePath, outputDir string) (V2SemanticIR, error) {
	if err := requireOutputOutside(outputDir, filepath.Dir(sourcePath)); err != nil { return V2SemanticIR{}, err }
	ir, err := ParseV2Source(sourcePath)
	if err != nil { return V2SemanticIR{}, err }
	if err := WriteJSON(filepath.Join(outputDir, "semantic-ir.json"), ir); err != nil { return V2SemanticIR{}, err }
	collector, err := renderGeneratedV2Collector(ir)
	if err != nil { return V2SemanticIR{}, err }
	if err := WriteText(filepath.Join(outputDir, "generated", "collector.go"), collector); err != nil { return V2SemanticIR{}, err }
	wrapper := "#!/usr/bin/env bash\nset -Eeuo pipefail\nfixture=${1:?fixture path is required}\nout=${2:?output directory is required}\nexec go run \"$(dirname \"$0\")/collector.go\" --fixture \"$fixture\" --out \"$out\"\n"
	if err := WriteText(filepath.Join(outputDir, "generated", "collect.sh"), wrapper); err != nil { return V2SemanticIR{}, err }
	return ir, nil
}

func renderGeneratedV2Collector(ir V2SemanticIR) (string, error) {
	data, err := json.Marshal(ir)
	if err != nil { return "", err }
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

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func digestJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil { return "", err }
	var normalized any
	if err := json.Unmarshal(data, &normalized); err != nil { return "", err }
	canonical, err := json.Marshal(normalized)
	if err != nil { return "", err }
	return digestBytes(canonical), nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil { return err }
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { return err }
	return os.WriteFile(path, data, 0o644)
}

func copyMap(value map[string]any) map[string]any {
	result := make(map[string]any, len(value))
	for key, item := range value { result[key] = item }
	return result
}

func removeEmptyOptional(value map[string]any) {
	for _, key := range []string{"phase", "pair_id", "scenario_id", "input_digest", "contract_digest", "fixture_digest", "toolchain", "runner", "job"} {
		if raw, ok := value[key]; ok {
			if text, ok := raw.(string); ok && text == "" { delete(value, key) }
		}
	}
}

func stringValue(value map[string]any, key string) string {
	text, _ := value[key].(string)
	return text
}

func main() {
	fixture := flag.String("fixture", "", "deterministic v2 fixture path")
	out := flag.String("out", "", "caller-owned output directory")
	flag.Parse()
	if *fixture == "" || *out == "" { fmt.Fprintln(os.Stderr, "fixture and out are required"); os.Exit(64) }
	if err := run(*fixture, *out); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}

func run(fixturePath, outDir string) error {
	fixtureBytes, err := os.ReadFile(fixturePath)
	if err != nil { return err }
	var input map[string]any
	if err := json.Unmarshal(fixtureBytes, &input); err != nil { return err }
	var ir map[string]any
	if err := json.Unmarshal([]byte(embeddedIR), &ir); err != nil { return err }
	declared := map[string]bool{}
	measurements, _ := ir["measurements"].([]any)
	for _, raw := range measurements { item, _ := raw.(map[string]any); declared[stringValue(item, "measurement_id")] = true }
	samples, _ := input["samples"].([]any)
	counts := map[string]int{}
	for _, raw := range samples {
		item, _ := raw.(map[string]any)
		metricID := stringValue(item, "metric_id")
		if !declared[metricID] { return fmt.Errorf("fixture refers to undeclared v2 metric %q", metricID) }
		counts[metricID]++
	}
	measuredOnce := true
	for _, raw := range measurements {
		item, _ := raw.(map[string]any)
		if counts[stringValue(item, "measurement_id")] == 0 { measuredOnce = false }
	}
	authorities := []string{"generated-stage-collector"}
	if raw, ok := input["collector_authorities"].([]any); ok && len(raw) > 0 {
		authorities = make([]string, 0, len(raw))
		for _, item := range raw { if text, ok := item.(string); ok { authorities = append(authorities, text) } }
		if len(authorities) == 0 { authorities = []string{"generated-stage-collector"} }
	}
	collectorAuthority := authorities[0]
	if len(authorities) > 1 { collectorAuthority = "competing-collectors" }
	collectorIdentity, err := digestJSON(map[string]any{"kind":"generated-stage-collector", "ir_digest":stringValue(ir, "digest"), "authorities":authorities})
	if err != nil { return err }
	runtimeAuthority, ok := input["runtime_authority"].(map[string]any)
	if !ok { runtimeAuthority = map[string]any{"repository_writes":0, "apply_authority":0, "commit_authority":0, "merge_authority":0, "tag_authority":0, "release_authority":0, "local_test_executions":0, "cross_project_required_gates":0} }
	result := map[string]any{
		"schema":"gooo/measurement-boundary/collection/v2", "ir_digest":stringValue(ir, "digest"), "fixture_digest":digestBytes(fixtureBytes),
		"collector":map[string]any{"kind":"generated-stage-collector", "generated":true, "measured_once":measuredOnce, "authority":collectorAuthority, "authorities":authorities, "identity_digest":collectorIdentity, "input_scope":"repository-read-only", "output_scope":"caller-owned-temp-output-only", "operator_authority":"authoring-only", "runtime_authority":runtimeAuthority},
		"observations":make([]any, 0, len(samples)), "receipts":make([]any, 0, len(samples)), "consumers":make([]any, 0),
	}
	for _, raw := range samples {
		item, _ := raw.(map[string]any)
		payload := copyMap(item)
		delete(payload, "consumer_artifacts"); delete(payload, "tamper_receipt"); delete(payload, "tamper_consumer")
		removeEmptyOptional(payload)
		observationDigest, err := digestJSON(payload)
		if err != nil { return err }
		reportedDigest := observationDigest
		if tampered, ok := item["tamper_receipt"].(bool); ok && tampered { reportedDigest = "sha256:" + strings.Repeat("0", 64) }
		receipt := copyMap(item)
		delete(receipt, "start_event"); delete(receipt, "end_event"); delete(receipt, "consumer_artifacts"); delete(receipt, "tamper_receipt"); delete(receipt, "tamper_consumer")
		removeEmptyOptional(receipt)
		receipt["schema"] = "gooo/measurement-boundary/receipt/v2"
		receipt["causal_events"] = map[string]any{"start":stringValue(item, "start_event"), "end":stringValue(item, "end_event")}
		receipt["observation_digest"] = observationDigest
		receipt["receipt_digest"] = observationDigest
		observation := copyMap(item)
		delete(observation, "tamper_receipt"); delete(observation, "tamper_consumer")
		removeEmptyOptional(observation)
		observation["receipt_digest"] = reportedDigest
		result["receipts"] = append(result["receipts"].([]any), receipt)
		result["observations"] = append(result["observations"].([]any), observation)
		consumerNames, _ := item["consumer_artifacts"].([]any)
		if len(consumerNames) == 0 { consumerNames = []any{stringValue(item, "source_artifact")} }
		for _, name := range consumerNames {
			consumerDigest := reportedDigest
			if tampered, ok := item["tamper_consumer"].(bool); ok && tampered { consumerDigest = "sha256:" + strings.Repeat("f", 64) }
			result["consumers"] = append(result["consumers"].([]any), map[string]any{"name":name, "metric_id":stringValue(item, "metric_id"), "stage_id":stringValue(item, "stage_id"), "covered_operations":item["covered_operations"], "receipt_digest":consumerDigest})
		}
	}
	observations := result["observations"].([]any)
	sort.SliceStable(observations, func(i, j int) bool {
		left, _ := observations[i].(map[string]any); right, _ := observations[j].(map[string]any)
		if stringValue(left, "metric_id") != stringValue(right, "metric_id") { return stringValue(left, "metric_id") < stringValue(right, "metric_id") }
		if stringValue(left, "phase") != stringValue(right, "phase") { return stringValue(left, "phase") < stringValue(right, "phase") }
		return stringValue(left, "source_artifact") < stringValue(right, "source_artifact")
	})
	result["digest"] = ""
	collectionDigest, err := digestJSON(result)
	if err != nil { return err }
	result["digest"] = collectionDigest
	if err := writeJSON(filepath.Join(outDir, "collection.json"), result); err != nil { return err }
	for index, raw := range result["receipts"].([]any) { if err := writeJSON(filepath.Join(outDir, "receipts", fmt.Sprintf("receipt-%03d.json", index+1)), raw); err != nil { return err } }
	return nil
}
`
	template = strings.ReplaceAll(template, "__EMBEDDED_IR__", strconv.Quote(string(data)))
	return template, nil
}

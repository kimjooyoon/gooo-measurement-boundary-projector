package projector

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func RunV2Conformance(sourcePath, corpusPath, outputDir string) (V2ConformanceSummary, error) {
	if err := requireOutputOutside(outputDir, filepath.Dir(sourcePath), filepath.Dir(corpusPath)); err != nil {
		return V2ConformanceSummary{}, err
	}
	ir, err := ParseV2Source(sourcePath)
	if err != nil { return V2ConformanceSummary{}, err }
	var corpus V2Corpus
	if err := LoadJSON(corpusPath, &corpus); err != nil { return V2ConformanceSummary{}, err }
	if corpus.Schema != V2CorpusSchema || len(corpus.Cases) != 12 {
		return V2ConformanceSummary{}, fmt.Errorf("v2 corpus must use %s and contain exactly 12 cases", V2CorpusSchema)
	}
	entries := append([]V2CorpusEntry(nil), corpus.Cases...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Ordinal < entries[j].Ordinal })
	counts := map[V2Decision]int{V2Closed: 0, V2Unknown: 0, V2Refuted: 0}
	for _, entry := range entries { counts[entry.Expected]++ }
	if counts[V2Closed] != 4 || counts[V2Unknown] != 4 || counts[V2Refuted] != 4 {
		return V2ConformanceSummary{}, fmt.Errorf("v2 corpus denominator must be CLOSED=4 UNKNOWN=4 REFUTED=4")
	}
	summary := V2ConformanceSummary{
		Schema: V2ConformanceSchema, Total: 12, Tests: make([]V2TestVector, 0, 12),
		ControlledPairs: make([]V2PairVector, 0), OptionalObservations: append([]V2OptionalObservation(nil), ir.OptionalObservations...),
		RuntimeAuthority: V2RuntimeAuthority{},
	}
	for _, entry := range entries {
		caseDir := filepath.Join(outputDir, entry.CaseID)
		if err := os.MkdirAll(caseDir, 0o755); err != nil { return V2ConformanceSummary{}, err }
		fixturePath := filepath.Join(filepath.Dir(corpusPath), filepath.FromSlash(entry.Path))
		if _, err := CompileV2(sourcePath, filepath.Join(caseDir, "compile")); err != nil {
			return V2ConformanceSummary{}, fmt.Errorf("v2 case %s compile: %w", entry.CaseID, err)
		}
		collection, _, err := CollectV2Fixture(ir, fixturePath, filepath.Join(caseDir, "collection"))
		if err != nil { return V2ConformanceSummary{}, fmt.Errorf("v2 case %s collect: %w", entry.CaseID, err) }
		evaluation, err := EvaluateV2(ir, collection, filepath.Join(caseDir, "evaluation.json"))
		if err != nil { return V2ConformanceSummary{}, fmt.Errorf("v2 case %s evaluate: %w", entry.CaseID, err) }
		if evaluation.Decision != entry.Expected {
			return V2ConformanceSummary{}, fmt.Errorf("v2 case %s observed %s, expected %s", entry.CaseID, evaluation.Decision, entry.Expected)
		}
		if err := WriteText(filepath.Join(caseDir, "human-report.md"), RenderV2HumanReport(evaluation)); err != nil { return V2ConformanceSummary{}, err }
		evaluationDigest, _, err := DigestFile(filepath.Join(caseDir, "evaluation.json"))
		if err != nil { return V2ConformanceSummary{}, err }
		summary.Tests = append(summary.Tests, V2TestVector{
			Ordinal: entry.Ordinal, CaseID: entry.CaseID, Expected: entry.Expected, Observed: evaluation.Decision,
			FixtureDigest: collection.FixtureDigest, EvaluationDigest: evaluationDigest,
		})
		switch evaluation.Decision { case V2Closed: summary.Closed++; case V2Unknown: summary.Unknown++; case V2Refuted: summary.Refuted++ }
		summary.ControlledPairs = append(summary.ControlledPairs, v2PairVectors(collection)...)
	}
	if summary.Closed != 4 || summary.Unknown != 4 || summary.Refuted != 4 {
		return V2ConformanceSummary{}, fmt.Errorf("v2 observed denominator must be CLOSED=4 UNKNOWN=4 REFUTED=4")
	}
	if err := WriteJSON(filepath.Join(outputDir, "semantic-ir.json"), ir); err != nil { return V2ConformanceSummary{}, err }
	if err := WriteJSON(filepath.Join(outputDir, "conformance-summary.json"), summary); err != nil { return V2ConformanceSummary{}, err }
	return summary, nil
}

func v2PairVectors(collection V2Collection) []V2PairVector {
	byMetric := map[string][]V2CollectedObservation{}
	for _, observation := range collection.Observations { byMetric[observation.MetricID] = append(byMetric[observation.MetricID], observation) }
	result := make([]V2PairVector, 0)
	for metricID, observations := range byMetric {
		if !v2PairShape(observations) || observations[0].Value == nil || observations[1].Value == nil { continue }
		var before, after int64
		if observations[0].Phase == "before" { before, after = *observations[0].Value, *observations[1].Value } else { before, after = *observations[1].Value, *observations[0].Value }
		item := observations[0]
		result = append(result, V2PairVector{MetricID: metricID, PairID: item.PairID, ScenarioID: item.ScenarioID, InputDigest: item.InputDigest, ContractDigest: item.ContractDigest, FixtureDigest: item.FixtureDigest, Toolchain: item.Toolchain, Runner: item.Runner, Job: item.Job, Before: before, After: after})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].MetricID < result[j].MetricID })
	return result
}

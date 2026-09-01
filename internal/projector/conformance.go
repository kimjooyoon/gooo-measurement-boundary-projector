package projector

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func RunConformance(sourcePath, corpusPath, outputDir string) (ConformanceSummary, error) {
	if err := requireOutputOutside(outputDir, filepath.Dir(sourcePath)); err != nil {
		return ConformanceSummary{}, err
	}
	ir, err := ParseSource(sourcePath)
	if err != nil {
		return ConformanceSummary{}, err
	}
	var corpus Corpus
	if err := LoadJSON(corpusPath, &corpus); err != nil {
		return ConformanceSummary{}, err
	}
	if corpus.Schema != CorpusSchema || len(corpus.Cases) < 9 {
		return ConformanceSummary{}, fmt.Errorf("corpus must use %s and contain at least nine cases", CorpusSchema)
	}
	entries := append([]CorpusEntry(nil), corpus.Cases...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Ordinal < entries[j].Ordinal })
	summary := ConformanceSummary{
		Schema: "gooo/measurement-boundary/conformance/v1", Total: len(entries), Selected: len(entries),
		Executed: len(entries), Reused: 0, Tests: make([]TestVector, 0, len(entries)),
	}
	for _, entry := range entries {
		caseDir := filepath.Join(outputDir, entry.CaseID)
		if err := os.MkdirAll(caseDir, 0o755); err != nil {
			return ConformanceSummary{}, err
		}
		fixturePath := filepath.Join(filepath.Dir(corpusPath), filepath.FromSlash(entry.Path))
		if _, err := Compile(sourcePath, filepath.Join(caseDir, "compile")); err != nil {
			return ConformanceSummary{}, fmt.Errorf("case %s compile: %w", entry.CaseID, err)
		}
		collection, _, err := CollectFixture(ir, fixturePath, filepath.Join(caseDir, "collection"))
		if err != nil {
			return ConformanceSummary{}, fmt.Errorf("case %s collect: %w", entry.CaseID, err)
		}
		evaluation, err := Evaluate(ir, collection, filepath.Join(caseDir, "evaluation.json"))
		if err != nil {
			return ConformanceSummary{}, fmt.Errorf("case %s evaluate: %w", entry.CaseID, err)
		}
		if evaluation.Decision != entry.Expected {
			return ConformanceSummary{}, fmt.Errorf("case %s observed %s, expected %s", entry.CaseID, evaluation.Decision, entry.Expected)
		}
		if err := WriteText(filepath.Join(caseDir, "human-report.md"), RenderHumanReport(evaluation)); err != nil {
			return ConformanceSummary{}, err
		}
		evaluationDigest, _, err := DigestFile(filepath.Join(caseDir, "evaluation.json"))
		if err != nil {
			return ConformanceSummary{}, err
		}
		summary.Tests = append(summary.Tests, TestVector{
			Ordinal: entry.Ordinal, CaseID: entry.CaseID, Expected: entry.Expected,
			Observed: evaluation.Decision, FixtureDigest: collection.FixtureDigest,
			EvaluationDigest: evaluationDigest,
		})
		switch evaluation.Decision {
		case Closed:
			summary.Closed++
		case Unknown:
			summary.Unknown++
		case Refuted:
			summary.Refuted++
		}
	}
	if err := WriteJSON(filepath.Join(outputDir, "semantic-ir.json"), ir); err != nil {
		return ConformanceSummary{}, err
	}
	if err := WriteJSON(filepath.Join(outputDir, "conformance-summary.json"), summary); err != nil {
		return ConformanceSummary{}, err
	}
	return summary, nil
}

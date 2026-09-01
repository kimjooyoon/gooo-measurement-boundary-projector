package projector

import (
	"encoding/json"
	"sort"
)

func DigestV2JSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	var normalized any
	if err := json.Unmarshal(data, &normalized); err != nil {
		return "", err
	}
	canonical, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return DigestBytes(canonical), nil
}

func digestV2IR(ir V2SemanticIR) (string, error) {
	without, err := cloneWithoutDigest(ir, func(value *V2SemanticIR) { value.Digest = "" })
	if err != nil {
		return "", err
	}
	return DigestV2JSON(without)
}

func digestV2Collection(collection V2Collection) (string, error) {
	without, err := cloneWithoutDigest(collection, func(value *V2Collection) { value.Digest = "" })
	if err != nil {
		return "", err
	}
	return DigestV2JSON(without)
}

func v2ExactStrings(left, right []string) bool {
	left = uniqueStrings(left)
	right = uniqueStrings(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func v2SortedObservations(values []V2CollectedObservation) {
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].MetricID != values[j].MetricID {
			return values[i].MetricID < values[j].MetricID
		}
		if values[i].Phase != values[j].Phase {
			return values[i].Phase < values[j].Phase
		}
		return values[i].SourceArtifact < values[j].SourceArtifact
	})
}

func v2AuthorityEscalated(value V2RuntimeAuthority) bool {
	return value.RepositoryWrites != 0 || value.ApplyAuthority != 0 || value.CommitAuthority != 0 ||
		value.MergeAuthority != 0 || value.TagAuthority != 0 || value.ReleaseAuthority != 0 ||
		value.LocalTestExecutions != 0 || value.CrossProjectRequiredGates != 0
}

func v2PairContextEqual(left, right V2CollectedObservation) bool {
	return left.PairID != "" && left.PairID == right.PairID &&
		left.ScenarioID == right.ScenarioID && left.InputDigest == right.InputDigest &&
		left.ContractDigest == right.ContractDigest && left.FixtureDigest == right.FixtureDigest &&
		left.Toolchain == right.Toolchain && left.Runner == right.Runner && left.Job == right.Job
}

func v2PairShape(values []V2CollectedObservation) bool {
	if len(values) != 2 || !v2PairContextEqual(values[0], values[1]) {
		return false
	}
	return (values[0].Phase == "before" && values[1].Phase == "after") ||
		(values[0].Phase == "after" && values[1].Phase == "before")
}

func v2ImprovementUnknown(spec V2MeasurementSpec, reason, class, next string, blocked []string) V2Improvement {
	frontier := &V2UnknownFrontier{
		Stage: spec.Stage, Step: spec.Step, Reason: reason, UnknownClass: class,
		NextOperation: next, BlockedBy: uniqueStrings(blocked),
	}
	return V2Improvement{State: V2Unknown, Reason: reason, Unknown: frontier}
}

package projector

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func ParseV2Source(path string) (V2SemanticIR, error) {
	digest, data, err := DigestFile(path)
	if err != nil {
		return V2SemanticIR{}, err
	}
	lines := strings.Split(string(data), "\n")
	var packageName string
	var namespace string
	measurements := make([]V2MeasurementSpec, 0)
	optionalObservations := make([]V2OptionalObservation, 0)
	var currentMeasurement *V2MeasurementSpec
	var currentOptional *V2OptionalObservation
	seenIDs := map[string]bool{}
	for lineNumber, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		tokens, err := parseTokens(line)
		if err != nil {
			return V2SemanticIR{}, fmt.Errorf("source line %d: %w", lineNumber+1, err)
		}
		if len(tokens) == 0 {
			continue
		}
		if currentMeasurement == nil && currentOptional == nil {
			switch tokens[0] {
			case "package":
				if len(tokens) != 2 {
					return V2SemanticIR{}, fmt.Errorf("source line %d: malformed package", lineNumber+1)
				}
				packageName = tokens[1]
			case "namespace":
				if len(tokens) != 2 {
					return V2SemanticIR{}, fmt.Errorf("source line %d: malformed namespace", lineNumber+1)
				}
				namespace = tokens[1]
			case "measurement":
				if len(tokens) != 3 || tokens[2] != "{" {
					return V2SemanticIR{}, fmt.Errorf("source line %d: measurement must start with an id and opening brace", lineNumber+1)
				}
				if seenIDs[tokens[1]] {
					return V2SemanticIR{}, fmt.Errorf("source line %d: duplicate measurement %q", lineNumber+1, tokens[1])
				}
				seenIDs[tokens[1]] = true
				currentMeasurement = &V2MeasurementSpec{MeasurementID: tokens[1], IdentityDigests: map[string]string{}}
			case "optional_observation":
				if len(tokens) != 3 || tokens[2] != "{" {
					return V2SemanticIR{}, fmt.Errorf("source line %d: optional_observation must start with an id and opening brace", lineNumber+1)
				}
				currentOptional = &V2OptionalObservation{ObservationID: tokens[1]}
			default:
				return V2SemanticIR{}, fmt.Errorf("source line %d: unexpected record %q", lineNumber+1, tokens[0])
			}
			continue
		}
		if len(tokens) == 1 && tokens[0] == "}" {
			if currentMeasurement != nil {
				if err := validateV2Measurement(*currentMeasurement, lineNumber+1); err != nil {
					return V2SemanticIR{}, err
				}
				measurements = append(measurements, *currentMeasurement)
				currentMeasurement = nil
				continue
			}
			if err := validateV2OptionalObservation(*currentOptional, lineNumber+1); err != nil {
				return V2SemanticIR{}, err
			}
			optionalObservations = append(optionalObservations, *currentOptional)
			currentOptional = nil
			continue
		}
		if currentMeasurement != nil {
			if err := parseV2MeasurementField(currentMeasurement, tokens, lineNumber+1); err != nil {
				return V2SemanticIR{}, err
			}
		} else if err := parseV2OptionalField(currentOptional, tokens, lineNumber+1); err != nil {
			return V2SemanticIR{}, err
		}
	}
	if currentMeasurement != nil {
		return V2SemanticIR{}, fmt.Errorf("source ended before measurement %q closed", currentMeasurement.MeasurementID)
	}
	if currentOptional != nil {
		return V2SemanticIR{}, fmt.Errorf("source ended before optional observation %q closed", currentOptional.ObservationID)
	}
	if packageName == "" || namespace == "" || len(measurements) == 0 {
		return V2SemanticIR{}, fmt.Errorf("source must declare package, namespace, and at least one v2 measurement")
	}
	ir := V2SemanticIR{
		Schema: V2IRSchema, SourcePath: filepath.ToSlash(path), SourceDigest: digest,
		Namespace: namespace, Measurements: measurements, OptionalObservations: optionalObservations,
	}
	ir.Digest, err = digestV2IR(ir)
	if err != nil {
		return V2SemanticIR{}, err
	}
	return ir, nil
}

func parseV2MeasurementField(target *V2MeasurementSpec, tokens []string, lineNumber int) error {
	if len(tokens) < 2 {
		return fmt.Errorf("source line %d: v2 measurement field is missing a value", lineNumber)
	}
	switch tokens[0] {
	case "stage_id":
		target.StageID = tokens[1]
	case "stage":
		target.Stage = tokens[1]
	case "step":
		target.Step = tokens[1]
	case "start_event":
		target.CausalEvents.Start = tokens[1]
	case "end_event":
		target.CausalEvents.End = tokens[1]
	case "include":
		if len(tokens) != 2 {
			return fmt.Errorf("source line %d: include needs one stable operation id", lineNumber)
		}
		target.CoveredOperations = append(target.CoveredOperations, tokens[1])
	case "exclude":
		if len(tokens) != 2 {
			return fmt.Errorf("source line %d: exclude needs one stable operation id", lineNumber)
		}
		target.ExcludedOperations = append(target.ExcludedOperations, tokens[1])
	case "child_process":
		if len(tokens) != 2 {
			return fmt.Errorf("source line %d: child_process needs one stable process id", lineNumber)
		}
		target.ExpectedChildProcesses = append(target.ExpectedChildProcesses, tokens[1])
	case "child_coverage":
		target.ChildProcessCoverage = tokens[1]
	case "clock":
		target.Clock = tokens[1]
	case "resolution_ms":
		value, err := strconv.ParseInt(tokens[1], 10, 64)
		if err != nil {
			return fmt.Errorf("source line %d: malformed resolution_ms: %w", lineNumber, err)
		}
		target.ResolutionMS = value
	case "rss_scope":
		target.RSSProcessTreeScope = tokens[1]
	case "input_receipt":
		target.InputReceiptDigest = tokens[1]
	case "output_receipt":
		target.OutputReceiptDigest = tokens[1]
	case "unit":
		target.Unit = tokens[1]
	case "authority":
		target.SourceAuthority = tokens[1]
	case "method":
		target.ObservationMethod = tokens[1]
	case "scope":
		target.Scope = tokens[1]
	case "identity":
		if len(tokens) != 3 {
			return fmt.Errorf("source line %d: identity needs a name and digest", lineNumber)
		}
		if target.IdentityDigests[tokens[1]] != "" {
			return fmt.Errorf("source line %d: duplicate identity %q", lineNumber, tokens[1])
		}
		target.IdentityDigests[tokens[1]] = tokens[2]
	case "direction":
		target.Direction = tokens[1]
	case "nullable":
		target.NullablePolicy = tokens[1]
	case "precedence":
		if len(tokens) != 4 {
			return fmt.Errorf("source line %d: precedence needs REFUTED UNKNOWN CLOSED", lineNumber)
		}
		target.ConflictPrecedence = []V2Decision{V2Decision(tokens[1]), V2Decision(tokens[2]), V2Decision(tokens[3])}
	default:
		return fmt.Errorf("source line %d: unknown v2 measurement field %q", lineNumber, tokens[0])
	}
	return nil
}

func parseV2OptionalField(target *V2OptionalObservation, tokens []string, lineNumber int) error {
	if len(tokens) < 2 {
		return fmt.Errorf("source line %d: optional observation field is missing a value", lineNumber)
	}
	parseInt := func(name, raw string) (int64, error) {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("source line %d: malformed %s: %w", lineNumber, name, err)
		}
		return value, nil
	}
	switch tokens[0] {
	case "source":
		target.Source = tokens[1]
	case "stage_id":
		target.StageID = tokens[1]
	case "step":
		target.Step = tokens[1]
	case "actual_main_lock_wall_ms":
		value, err := parseInt(tokens[0], tokens[1]); if err != nil { return err }; target.ActualMainLockWallMS = value
	case "product_receipt_baseline_wall_ms":
		value, err := parseInt(tokens[0], tokens[1]); if err != nil { return err }; target.ProductReceiptBaselineMS = value
	case "product_receipt_candidate_wall_ms":
		value, err := parseInt(tokens[0], tokens[1]); if err != nil { return err }; target.ProductReceiptCandidateMS = value
	case "decision":
		target.Decision = V2Decision(tokens[1])
	case "reason":
		target.Reason = tokens[1]
	case "acceptance":
		target.Acceptance = tokens[1]
	case "required_gate":
		value, err := strconv.ParseBool(tokens[1]); if err != nil { return fmt.Errorf("source line %d: malformed required_gate: %w", lineNumber, err) }; target.RequiredGate = value
	case "immutable_input":
		value, err := strconv.ParseBool(tokens[1]); if err != nil { return fmt.Errorf("source line %d: malformed immutable_input: %w", lineNumber, err) }; target.ImmutableInput = value
	default:
		return fmt.Errorf("source line %d: unknown optional observation field %q", lineNumber, tokens[0])
	}
	return nil
}

func validateV2Measurement(value V2MeasurementSpec, lineNumber int) error {
	missing := make([]string, 0)
	checks := map[string]string{
		"stage_id": value.StageID, "stage": value.Stage, "step": value.Step,
		"causal_events.start": value.CausalEvents.Start, "causal_events.end": value.CausalEvents.End,
		"child_process_coverage": value.ChildProcessCoverage, "clock": value.Clock,
		"rss_process_tree_scope": value.RSSProcessTreeScope, "input_receipt_digest": value.InputReceiptDigest,
		"output_receipt_digest": value.OutputReceiptDigest, "unit": value.Unit,
		"source_authority": value.SourceAuthority, "observation_method": value.ObservationMethod,
		"scope": value.Scope, "direction": value.Direction, "nullable_policy": value.NullablePolicy,
	}
	for name, item := range checks {
		if item == "" { missing = append(missing, name) }
	}
	if len(value.CoveredOperations) == 0 { missing = append(missing, "covered_operations") }
	if len(value.ExpectedChildProcesses) == 0 && value.ChildProcessCoverage != "none" { missing = append(missing, "expected_child_processes") }
	if len(value.IdentityDigests) == 0 { missing = append(missing, "identity_digests") }
	if value.ResolutionMS <= 0 { missing = append(missing, "resolution_ms>0") }
	if !isValidDigest(value.InputReceiptDigest) { missing = append(missing, "input_receipt_digest=sha256:<64 lowercase hex>") }
	if !isValidDigest(value.OutputReceiptDigest) { missing = append(missing, "output_receipt_digest=sha256:<64 lowercase hex>") }
	if len(value.ConflictPrecedence) != 3 || value.ConflictPrecedence[0] != V2Refuted || value.ConflictPrecedence[1] != V2Unknown || value.ConflictPrecedence[2] != V2Closed {
		missing = append(missing, "conflict_precedence=REFUTED>UNKNOWN>CLOSED")
	}
	if len(missing) > 0 { return fmt.Errorf("source line %d: v2 measurement %q missing %s", lineNumber, value.MeasurementID, strings.Join(missing, ", ")) }
	return nil
}

func validateV2OptionalObservation(value V2OptionalObservation, lineNumber int) error {
	missing := make([]string, 0)
	if value.Source == "" { missing = append(missing, "source") }
	if value.StageID == "" { missing = append(missing, "stage_id") }
	if value.Step == "" { missing = append(missing, "step") }
	if value.Decision != V2Unknown { missing = append(missing, "decision=UNKNOWN") }
	if value.Reason == "" { missing = append(missing, "reason") }
	if value.Acceptance != "optional" { missing = append(missing, "acceptance=optional") }
	if !value.ImmutableInput { missing = append(missing, "immutable_input=true") }
	if value.RequiredGate { missing = append(missing, "required_gate=false") }
	if len(missing) > 0 { return fmt.Errorf("source line %d: optional observation %q missing %s", lineNumber, value.ObservationID, strings.Join(missing, ", ")) }
	return nil
}

package projector

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func ParseSource(path string) (SemanticIR, error) {
	digest, data, err := DigestFile(path)
	if err != nil {
		return SemanticIR{}, err
	}
	lines := strings.Split(string(data), "\n")
	var packageName string
	var namespace string
	measurements := make([]MeasurementSpec, 0)
	var current *MeasurementSpec
	seenIDs := map[string]bool{}
	for lineNumber, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		tokens, err := parseTokens(line)
		if err != nil {
			return SemanticIR{}, fmt.Errorf("source line %d: %w", lineNumber+1, err)
		}
		if len(tokens) == 0 {
			continue
		}
		if current == nil {
			switch tokens[0] {
			case "package":
				if len(tokens) != 2 {
					return SemanticIR{}, fmt.Errorf("source line %d: malformed package", lineNumber+1)
				}
				packageName = tokens[1]
			case "namespace":
				if len(tokens) != 2 {
					return SemanticIR{}, fmt.Errorf("source line %d: malformed namespace", lineNumber+1)
				}
				namespace = tokens[1]
			case "measurement":
				if len(tokens) != 3 || tokens[2] != "{" {
					return SemanticIR{}, fmt.Errorf("source line %d: measurement must start with an id and opening brace", lineNumber+1)
				}
				if seenIDs[tokens[1]] {
					return SemanticIR{}, fmt.Errorf("source line %d: duplicate measurement %q", lineNumber+1, tokens[1])
				}
				seenIDs[tokens[1]] = true
				current = &MeasurementSpec{MeasurementID: tokens[1], IdentityDigests: map[string]string{}}
			default:
				return SemanticIR{}, fmt.Errorf("source line %d: unexpected record %q", lineNumber+1, tokens[0])
			}
			continue
		}
		if len(tokens) == 1 && tokens[0] == "}" {
			if err := validateMeasurement(*current, lineNumber+1); err != nil {
				return SemanticIR{}, err
			}
			measurements = append(measurements, *current)
			current = nil
			continue
		}
		if err := parseMeasurementField(current, tokens, lineNumber+1); err != nil {
			return SemanticIR{}, err
		}
	}
	if current != nil {
		return SemanticIR{}, fmt.Errorf("source ended before measurement %q closed", current.MeasurementID)
	}
	if packageName == "" || namespace == "" || len(measurements) == 0 {
		return SemanticIR{}, fmt.Errorf("source must declare package, namespace, and at least one measurement")
	}
	ir := SemanticIR{Schema: IRSchema, SourcePath: filepath.ToSlash(path), SourceDigest: digest, Measurements: measurements}
	ir.Digest, err = digestIR(ir)
	if err != nil {
		return SemanticIR{}, err
	}
	return ir, nil
}

func parseMeasurementField(target *MeasurementSpec, tokens []string, lineNumber int) error {
	if len(tokens) < 2 {
		return fmt.Errorf("source line %d: measurement field is missing a value", lineNumber)
	}
	switch tokens[0] {
	case "stage":
		target.Stage = tokens[1]
	case "step":
		target.Step = tokens[1]
	case "span":
		if len(tokens) != 3 {
			return fmt.Errorf("source line %d: span needs start and end boundaries", lineNumber)
		}
		target.Span = Span{StartBoundary: tokens[1], EndBoundary: tokens[2]}
	case "include":
		if len(tokens) != 2 {
			return fmt.Errorf("source line %d: include needs one operation", lineNumber)
		}
		target.IncludedOperations = append(target.IncludedOperations, tokens[1])
	case "exclude":
		if len(tokens) != 2 {
			return fmt.Errorf("source line %d: exclude needs one operation", lineNumber)
		}
		target.ExcludedOperations = append(target.ExcludedOperations, tokens[1])
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
		target.ConflictPrecedence = []Decision{Decision(tokens[1]), Decision(tokens[2]), Decision(tokens[3])}
	default:
		return fmt.Errorf("source line %d: unknown measurement field %q", lineNumber, tokens[0])
	}
	return nil
}

func validateMeasurement(value MeasurementSpec, lineNumber int) error {
	missing := make([]string, 0)
	checks := map[string]string{
		"stage": value.Stage, "step": value.Step, "span.start_boundary": value.Span.StartBoundary,
		"span.end_boundary": value.Span.EndBoundary, "unit": value.Unit,
		"source_authority": value.SourceAuthority, "observation_method": value.ObservationMethod,
		"scope": value.Scope, "direction": value.Direction, "nullable_policy": value.NullablePolicy,
	}
	for name, item := range checks {
		if item == "" {
			missing = append(missing, name)
		}
	}
	if len(value.IncludedOperations) == 0 {
		missing = append(missing, "included_operations")
	}
	if len(value.IdentityDigests) == 0 {
		missing = append(missing, "identity_digests")
	}
	if len(value.ConflictPrecedence) != 3 || value.ConflictPrecedence[0] != Refuted || value.ConflictPrecedence[1] != Unknown || value.ConflictPrecedence[2] != Closed {
		missing = append(missing, "conflict_precedence=REFUTED>UNKNOWN>CLOSED")
	}
	if len(missing) > 0 {
		return fmt.Errorf("source line %d: measurement %q missing %s", lineNumber, value.MeasurementID, strings.Join(missing, ", "))
	}
	return nil
}

func parseTokens(line string) ([]string, error) {
	result := make([]string, 0)
	for index := 0; index < len(line); {
		for index < len(line) && (line[index] == ' ' || line[index] == '\t') {
			index++
		}
		if index == len(line) {
			break
		}
		if line[index] == '"' {
			start := index
			index++
			escaped := false
			closed := false
			for index < len(line) {
				if escaped {
					escaped = false
					index++
					continue
				}
				if line[index] == '\\' {
					escaped = true
					index++
					continue
				}
				if line[index] == '"' {
					index++
					closed = true
					break
				}
				index++
			}
			if !closed {
				return nil, fmt.Errorf("unterminated quoted token")
			}
			value, err := strconv.Unquote(line[start:index])
			if err != nil {
				return nil, fmt.Errorf("malformed quoted token: %w", err)
			}
			result = append(result, value)
			continue
		}
		start := index
		for index < len(line) && line[index] != ' ' && line[index] != '\t' {
			index++
		}
		result = append(result, line[start:index])
	}
	return result, nil
}

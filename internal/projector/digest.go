package projector

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func DigestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func DigestFile(path string) (string, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	return DigestBytes(data), data, nil
}

func DigestJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return DigestBytes(data), nil
}

func WriteJSON(path string, value any) error {
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

func WriteText(path, value string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(value), 0o644)
}

func LoadJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func cloneWithoutDigest[T any](value T, clear func(*T)) (T, error) {
	data, err := json.Marshal(value)
	if err != nil {
		var zero T
		return zero, err
	}
	var copied T
	if err := json.Unmarshal(data, &copied); err != nil {
		var zero T
		return zero, err
	}
	clear(&copied)
	return copied, nil
}

func digestIR(ir SemanticIR) (string, error) {
	without, err := cloneWithoutDigest(ir, func(value *SemanticIR) { value.Digest = "" })
	if err != nil {
		return "", err
	}
	return DigestJSON(without)
}

func digestCollection(collection Collection) (string, error) {
	without, err := cloneWithoutDigest(collection, func(value *Collection) { value.Digest = "" })
	if err != nil {
		return "", err
	}
	return DigestJSON(without)
}

func countPhysicalLines(data []byte) int64 {
	if len(data) == 0 {
		return 0
	}
	lines := int64(1)
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if data[len(data)-1] == '\n' {
		lines--
	}
	return lines
}

func InventoryFor(root, generatedRoot string) (Inventory, error) {
	var inventory Inventory
	inventory.RootReadmeExcluded = true
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			inventory.Directories++
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if filepath.ToSlash(relative) == "README.md" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		inventory.RegularFiles++
		inventory.TreeBytes += int64(len(data))
		switch {
		case strings.HasSuffix(relative, ".go"):
			inventory.GoFiles++
			inventory.GoPhysicalLines += countPhysicalLines(data)
		case strings.HasSuffix(relative, ".gooo"):
			inventory.GoooFiles++
			inventory.GoooPhysicalLines += countPhysicalLines(data)
		}
		return nil
	})
	if err != nil {
		return Inventory{}, err
	}
	if generatedRoot == "" {
		return inventory, nil
	}
	generatedRoot, err = filepath.Abs(generatedRoot)
	if err != nil {
		return Inventory{}, err
	}
	err = filepath.WalkDir(generatedRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		inventory.GeneratedArtifacts++
		inventory.GeneratedBytes += int64(len(data))
		return nil
	})
	return inventory, err
}

func sortedStrings(values []string) []string {
	copyOf := append([]string(nil), values...)
	sort.Strings(copyOf)
	return copyOf
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func mapsEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func isValidDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	for _, character := range value[len("sha256:"):] {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}

func requireOutputOutside(output string, forbidden ...string) error {
	absoluteOutput, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	for _, candidate := range forbidden {
		absoluteCandidate, err := filepath.Abs(candidate)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(absoluteCandidate, absoluteOutput)
		if err != nil {
			return err
		}
		if relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
			return fmt.Errorf("output %q is inside read-only input %q", absoluteOutput, absoluteCandidate)
		}
	}
	return nil
}

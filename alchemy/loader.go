package alchemy

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

func LoadCSV(path string) ([]Ingredient, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	return ParseCSV(f)
}

func ParseCSV(r io.Reader) ([]Ingredient, error) {
	rec := csv.NewReader(r)
	rows, err := rec.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("csv has no data rows")
	}

	seen := map[string][]string{} // name -> sorted effect names
	var out []Ingredient

	for i, row := range rows[1:] {
		if len(row) < 15 {
			return nil, fmt.Errorf("row %d: expected 15 columns, got %d", i+2, len(row))
		}
		name := strings.TrimSpace(row[0])

		effects, err := parseEffects(row)
		if err != nil {
			return nil, fmt.Errorf("row %d (%s): %w", i+2, name, err)
		}

		sort.Strings(effects)
		if prior, ok := seen[name]; ok {
			if !equalStringSlices(prior, effects) {
				return nil, fmt.Errorf("duplicate ingredient %q with different effects", name)
			}
			continue // same effects, skip
		}
		seen[name] = effects

		weight, err := strconv.ParseFloat(strings.TrimSpace(row[1]), 64)
		if err != nil {
			return nil, fmt.Errorf("row %d (%s) weight: %w", i+2, name, err)
		}
		val, err := strconv.Atoi(strings.TrimSpace(row[2]))
		if err != nil {
			return nil, fmt.Errorf("row %d (%s) value: %w", i+2, name, err)
		}

		out = append(out, Ingredient{Name: name, Weight: weight, Value: val, EffectsNames: effects})
	}
	return out, nil
}

func parseEffects(row []string) ([]string, error) {
	var effects []string
	for slot := range 4 {
		base := 3 + slot*3
		name := strings.TrimSpace(row[base])
		if name == "" {
			continue
		}
		effects = append(effects, name)
	}
	return effects, nil
}

func effectNameList(effects []IngredientEffect) []string {
	names := make([]string, len(effects))
	for i, e := range effects {
		names[i] = e.Name
	}
	sort.Strings(names)
	return names
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

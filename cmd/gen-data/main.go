package main

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/marcelboisson/simonrim-pots-helper/alchemy"
)

type outIngredient struct {
	Name    string `json:"name"`
	Value   int    `json:"value"`
	Effects []int  `json:"effects"`
}

type outData struct {
	Effects     []string        `json:"effects"`
	Ingredients []outIngredient `json:"ingredients"`
}

func fatal(msg string, args ...any) {
	slog.Error("fatal: "+msg, args...)
	os.Exit(1)
}

func main() {
	if len(os.Args) != 2 {
		fatal("missing file to process")
	}

	inputf := os.Args[1]

	slog.Info("reading", "f", inputf)
	ingredients, err := alchemy.LoadCSV(inputf)
	if err != nil {
		fatal("load csv", "err", err)
	}
	idx := alchemy.BuildIndex(ingredients)

	effectBit := make(map[string]int, len(idx.AllEffects))
	for i, e := range idx.AllEffects {
		effectBit[e] = i
	}

	out := outData{Effects: idx.AllEffects}
	for _, ing := range idx.Ingredients {
		bits := make([]int, len(ing.EffectsNames))
		for i, e := range ing.EffectsNames {
			bits[i] = effectBit[e]
		}
		out.Ingredients = append(out.Ingredients, outIngredient{
			Name:    ing.Name,
			Value:   ing.Value,
			Effects: bits,
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fatal("encode json", "err", err)
	}
	slog.Info("done", "effects", len(out.Effects), "ingredients", len(out.Ingredients))
}

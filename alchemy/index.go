package alchemy

import (
	"slices"
	"sort"
)

type Index struct {
	Ingredients         []Ingredient
	ByName              map[string]*Ingredient
	EffectToIngredients map[string][]*Ingredient
	IngredientToEffects map[string][]string
	AllEffects          []string
	effectBit           map[string]int
}

func BuildIndex(ingredients []Ingredient) *Index {
	idx := &Index{
		Ingredients:         ingredients,
		ByName:              make(map[string]*Ingredient, len(ingredients)),
		EffectToIngredients: make(map[string][]*Ingredient),
		IngredientToEffects: make(map[string][]string, len(ingredients)),
	}

	effectSet := map[string]struct{}{}
	for i := range ingredients {
		ing := &ingredients[i]
		idx.ByName[ing.Name] = ing
		idx.IngredientToEffects[ing.Name] = ing.EffectsNames
		for _, e := range ing.EffectsNames {
			idx.EffectToIngredients[e] = append(idx.EffectToIngredients[e], ing)
			effectSet[e] = struct{}{}
		}
	}

	for name := range effectSet {
		idx.AllEffects = append(idx.AllEffects, name)
	}
	sort.Strings(idx.AllEffects)

	idx.effectBit = make(map[string]int, len(idx.AllEffects))
	for i, e := range idx.AllEffects {
		idx.effectBit[e] = i
	}
	for i := range idx.Ingredients {
		ing := &idx.Ingredients[i]
		for _, e := range ing.EffectsNames {
			bit := idx.effectBit[e]
			ing.Effects[bit/64] |= uint64(1) << uint(bit%64)
		}
	}
	return idx
}

func (idx *Index) maskFor(effects []string) EffectMask {
	var m EffectMask
	for _, e := range effects {
		if bit, ok := idx.effectBit[e]; ok {
			m[bit/64] |= uint64(1) << uint(bit%64)
		}
	}
	return m
}

func (idx *Index) effectNamesFromMask(m EffectMask) []string {
	var out []string
	for i, e := range idx.AllEffects {
		if m[i/64]>>uint(i%64)&1 == 1 {
			out = append(out, e)
		}
	}
	return out
}

// IngredientsForEffect returns all ingredients providing the named effect, sorted by name.
func (idx *Index) IngredientsForEffect(effect string) []*Ingredient {
	ings := idx.EffectToIngredients[effect]
	out := make([]*Ingredient, len(ings))
	copy(out, ings)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// IngredientsForEffects returns ingredients that have ALL of the named effects, sorted by name.
func (idx *Index) IngredientsForEffects(effects []string) []*Ingredient {
	if len(effects) == 0 {
		return nil
	}
	counts := map[string]int{}
	for _, eff := range effects {
		for _, ing := range idx.EffectToIngredients[eff] {
			counts[ing.Name]++
		}
	}
	var out []*Ingredient
	for name, count := range counts {
		if count == len(effects) {
			out = append(out, idx.ByName[name])
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (idx *Index) newPotion(ingredients []*Ingredient, effects EffectMask) *Potion {
	slices.SortFunc(ingredients, IngredientByName)
	return &Potion{
		Ingredients: ingredients,
		Effects:     idx.effectNamesFromMask(effects),
	}
}

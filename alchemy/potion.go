package alchemy

import (
	"sort"
	"strings"
)

// IsNegative reports whether an effect is harmful (poison/debuff).
func IsNegative(effect string) bool {
	if strings.HasPrefix(effect, "Damage ") ||
		strings.HasPrefix(effect, "Lingering Damage ") ||
		strings.HasPrefix(effect, "Weakness ") {
		return true
	}
	switch effect {
	case "Burden", "Calm", "Command", "Fear", "Frenzy", "Paralysis", "Silence":
		return true
	}
	return false
}

// FilterPure removes potions that mix positive and negative effects.
// Each potion kept is either all-positive (a potion) or all-negative (a poison).
func FilterPure(potions []*Potion) []*Potion {
	out := potions[:0:0]
	for _, p := range potions {
		pos, neg := false, false
		for _, e := range p.Effects {
			if IsNegative(e) {
				neg = true
			} else {
				pos = true
			}
		}
		if !pos || !neg {
			out = append(out, p)
		}
	}
	return out
}

// Brew returns all valid potions brewable from the owned ingredient names,
// sorted descending by number of shared effects, then by ingredient names.
func (idx *Index) Brew(owned []string) []*Potion {
	ings := make([]*Ingredient, 0, len(owned))
	for _, name := range owned {
		if ing, ok := idx.ByName[name]; ok {
			ings = append(ings, ing)
		}
	}

	var potions []*Potion
	n := len(ings)

	for i := range n {
		for j := i + 1; j < n; j++ {
			// 2 ingredients cover all effects
			if ij := ings[i].Effects.and(ings[j].Effects); !ij.isZero() {
				potions = append(potions, idx.newPotion([]*Ingredient{ings[i], ings[j]}, ij))
			}
		}
	}

	for i := range n {
		for j := i + 1; j < n; j++ {
			ij := ings[i].Effects.and(ings[j].Effects)
			for k := j + 1; k < n; k++ {
				ik := ings[i].Effects.and(ings[k].Effects)
				jk := ings[j].Effects.and(ings[k].Effects)
				if ij.isZero() && ik.isZero() || ij.isZero() && jk.isZero() || ik.isZero() && jk.isZero() {
					continue
				}
				potions = append(potions, idx.newPotion([]*Ingredient{ings[i], ings[j], ings[k]}, ij.or(ik).or(jk)))
			}
		}
	}

	sort.Slice(potions, func(i, j int) bool {
		li, lj := len(potions[i].Effects), len(potions[j].Effects)
		if li != lj {
			return li > lj
		}
		return potions[i].Ingredients[0].Name < potions[j].Ingredients[0].Name
	})

	return potions
}

// FindCombos returns all valid 2- and 3-ingredient combos (across all known ingredients)
// whose brewed potion contains every one of the target effects.
//
// Skyrim rule: potion effects = union of pairwise intersections across all pairs in the combo.
func (idx *Index) FindCombos(targets []string) []*Potion {
	if len(targets) == 0 {
		return nil
	}
	targetMask := idx.maskFor(targets)

	seen := make(map[*Ingredient]struct{}, 8)
	for _, t := range targets {
		for _, ing := range idx.EffectToIngredients[t] {
			seen[ing] = struct{}{}
		}
	}
	cands := make([]*Ingredient, 0, len(seen))
	for ing := range seen {
		cands = append(cands, ing)
	}

	var potions []*Potion
	n := len(cands)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			ij := cands[i].Effects.and(cands[j].Effects)
			if ij.contains(targetMask) {
				pot := idx.newPotion([]*Ingredient{cands[i], cands[j]}, ij)
				potions = append(potions, pot)
				continue
			}
			for k := j + 1; k < n; k++ {
				ik := cands[i].Effects.and(cands[k].Effects)
				jk := cands[j].Effects.and(cands[k].Effects)
				if union := ij.or(ik).or(jk); union.contains(targetMask) {
					potions = append(potions, idx.newPotion([]*Ingredient{cands[i], cands[j], cands[k]}, union))
				}
			}
		}
	}

	cost := func(ings []*Ingredient) int {
		total := 0
		for _, ing := range ings {
			total += ing.Value
		}
		return total
	}

	sort.Slice(potions, func(i, j int) bool {
		ci, cj := cost(potions[i].Ingredients), cost(potions[j].Ingredients)
		if ci != cj {
			return ci < cj
		}
		li, lj := len(potions[i].Ingredients), len(potions[j].Ingredients)
		if li != lj {
			return li < lj
		}
		return potions[i].Ingredients[0].Name < potions[j].Ingredients[0].Name
	})

	return potions
}

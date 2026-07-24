package alchemy

import (
	"strings"
)

type IngredientEffect struct {
	Name      string  `json:"n"`
	Magnitude float64 `json:"m"`
	Duration  float64 `json:"d"`
}

// EffectMask is a 128-bit bitmask over effect IDs assigned at index-build time.
// Bit i corresponds to AllEffects[i] in the containing Index.
type EffectMask [2]uint64

func (a EffectMask) and(b EffectMask) EffectMask { return EffectMask{a[0] & b[0], a[1] & b[1]} }
func (a EffectMask) or(b EffectMask) EffectMask  { return EffectMask{a[0] | b[0], a[1] | b[1]} }
func (a EffectMask) isZero() bool                { return a[0] == 0 && a[1] == 0 }
func (a EffectMask) contains(b EffectMask) bool  { return a[0]&b[0] == b[0] && a[1]&b[1] == b[1] }

type Ingredient struct {
	Name         string
	Weight       float64
	Value        int
	EffectsNames []string // kept for display; not used in hot-path calculations
	Effects      EffectMask
}

func IngredientByName(a, b *Ingredient) int {
	return strings.Compare(a.Name, b.Name)
}

// Potion is a valid combination of 2 or 3 ingredients sharing at least one effect.
// SharedEffects is the union of effects shared by any pair within the combo.
type Potion struct {
	Ingredients []*Ingredient
	Effects     []string
}

func (p Potion) String() string {
	return strings.Join(p.IngredientsNames(), " + ")
}

func (p *Potion) IngredientsNames() []string {
	names := make([]string, len(p.Ingredients))
	for i, ing := range p.Ingredients {
		names[i] = ing.Name
	}
	return names
}

package alchemy_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/marcelboisson/simonrim-pots-helper/alchemy"

	"github.com/stretchr/testify/assert"
)

const testCSV = `NAME,WEIGHT,VALUE,E1,E1M,E1D,E2,E2M,E2D,E3,E3M,E3D,E4,E4M,E4D
Alpha,1,10,Restore Health,1,10,Fortify Speed,2,60,,,,,,
Beta,1,20,Restore Health,1,10,Fortify Strength,3,60,,,,,,
Gamma,1,15,Fortify Speed,2,60,Fortify Strength,3,60,,,,,,
Delta,1,5,Night Eye,0,4,,,,,,,,,
`

func parse(t *testing.T, csv string) *alchemy.Index {
	t.Helper()
	ings, err := alchemy.ParseCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return alchemy.BuildIndex(ings)
}

func TestLoadAndIndex(t *testing.T) {
	idx := parse(t, testCSV)
	if len(idx.Ingredients) != 4 {
		t.Fatalf("want 4 ingredients, got %d", len(idx.Ingredients))
	}
	ings := idx.IngredientsForEffect("Restore Health")
	if len(ings) != 2 {
		t.Fatalf("want 2 ingredients for Restore Health, got %d", len(ings))
	}
}

func TestDuplicateSkipped(t *testing.T) {
	csv := `NAME,WEIGHT,VALUE,E1,E1M,E1D,E2,E2M,E2D,E3,E3M,E3D,E4,E4M,E4D
Alpha,1,10,Restore Health,1,10,,,,,,,,,
Alpha,1,20,Restore Health,1,10,,,,,,,,,
`
	ings, err := alchemy.ParseCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ings) != 1 {
		t.Fatalf("want 1 ingredient after dedup, got %d", len(ings))
	}
}

func TestDuplicateDifferentEffectsErrors(t *testing.T) {
	csv := `NAME,WEIGHT,VALUE,E1,E1M,E1D,E2,E2M,E2D,E3,E3M,E3D,E4,E4M,E4D
Alpha,1,10,Restore Health,1,10,,,,,,,,,
Alpha,1,10,Fortify Speed,2,60,,,,,,,,,
`
	_, err := alchemy.ParseCSV(strings.NewReader(csv))
	if err == nil {
		t.Fatal("expected error for duplicate with different effects")
	}
}

func TestBrew(t *testing.T) {
	idx := parse(t, testCSV)

	// Alpha+Beta share Restore Health; Alpha+Gamma share Fortify Speed; Beta+Gamma share Fortify Strength
	potions := idx.Brew([]string{"Alpha", "Beta", "Gamma", "Delta"})

	// Expect 3 pairs + 1 triple
	cases := map[string]bool{}
	for _, p := range potions {
		cases[strings.Join(p.IngredientsNames(), "+")] = true
	}
	for _, want := range []string{"Alpha+Beta", "Alpha+Gamma", "Beta+Gamma", "Alpha+Beta+Gamma"} {
		if !cases[want] {
			fmt.Printf("%+v", cases)
			t.Errorf("missing potion %s", want)
		}
	}
	// Delta has no shared effects with anyone
	for key := range cases {
		if strings.Contains(key, "Delta") {
			t.Errorf("unexpected potion containing Delta: %s", key)
		}
	}
}

func TestBrewRanking(t *testing.T) {
	idx := parse(t, testCSV)
	potions := idx.Brew([]string{"Alpha", "Beta", "Gamma"})
	// Triple has 3 shared effects (union), pairs have 1 each - triple should rank first
	if len(potions) == 0 {
		t.Fatal("no potions")
	}
	if len(potions[0].Ingredients) != 3 {
		t.Errorf("expected triple to rank first, got %v", potions[0].IngredientsNames())
	}
}

func TestIngredientsForEffects(t *testing.T) {
	idx := parse(t, testCSV)
	// Only Gamma has both Fortify Speed AND Fortify Strength
	ings := idx.IngredientsForEffects([]string{"Fortify Speed", "Fortify Strength"})
	if len(ings) != 1 || ings[0].Name != "Gamma" {
		t.Errorf("want [Gamma], got %v", ings)
	}
}

func TestIsNegative(t *testing.T) {
	neg := []string{
		"Damage Health", "Damage Stamina", "Damage Magicka", "Damage Weapon", "Damage Armor",
		"Lingering Damage Health", "Lingering Damage Stamina", "Lingering Damage Magicka",
		"Weakness to Fire", "Weakness to Frost", "Weakness to Shock", "Weakness to Poison",
		"Burden", "Calm", "Command", "Fear", "Frenzy", "Paralysis", "Silence",
	}
	for _, e := range neg {
		if !alchemy.IsNegative(e) {
			t.Errorf("expected %q to be negative", e)
		}
	}
	pos := []string{"Restore Health", "Fortify Speed", "Invisibility", "Reflect Damage", "Waterbreathing"}
	for _, e := range pos {
		if alchemy.IsNegative(e) {
			t.Errorf("expected %q to be positive", e)
		}
	}
}

func TestFilterPure(t *testing.T) {
	idx := parse(t, testCSV)
	// testCSV has no negative effects, so FilterPure should return all potions unchanged
	all := idx.Brew([]string{"Alpha", "Beta", "Gamma"})
	filtered := alchemy.FilterPure(all)
	if len(filtered) != len(all) {
		t.Errorf("expected %d potions, got %d", len(all), len(filtered))
	}
}

func TestFindCombos(t *testing.T) {
	// testCSV: Alpha=[Restore Health, Fortify Speed], Beta=[Restore Health, Fortify Strength],
	//          Gamma=[Fortify Speed, Fortify Strength], Delta=[Night Eye]
	idx := parse(t, testCSV)

	t.Run("single effect", func(t *testing.T) {
		// Restore Health: Alpha+Beta pair covers it; triples Alpha+Beta+Gamma also covers it
		combos := idx.FindCombos([]string{"Restore Health"})
		found := map[string]bool{}
		for _, p := range combos {
			found[strings.Join(p.IngredientsNames(), "+")] = true
		}
		if !found["Alpha+Beta"] {
			t.Error("expected Alpha+Beta (both have Restore Health)")
		}
		if found["Alpha+Gamma"] {
			t.Error("Alpha+Gamma share only Fortify Speed, not Restore Health")
		}
	})

	t.Run("two effects needing triple", func(t *testing.T) {
		// Fortify Speed + Fortify Strength: no single pair covers both
		// Alpha+Gamma covers Fortify Speed; Beta+Gamma covers Fortify Strength
		// Triple Alpha+Beta+Gamma: (A∩B)∪(A∩G)∪(B∩G) = {Restore Health}∪{Fortify Speed}∪{Fortify Strength}
		// => covers both targets
		combos := idx.FindCombos([]string{"Fortify Speed", "Fortify Strength"})
		found := map[string]bool{}
		for _, p := range combos {
			found[strings.Join(p.IngredientsNames(), "+")] = true
		}
		if !found["Alpha+Beta+Gamma"] {
			t.Error("expected Alpha+Beta+Gamma triple to cover both target effects")
		}
		// Gamma alone covers both in a pair? Gamma+? — no single other ingredient has both
		// Alpha+Gamma pair: only Fortify Speed — does not cover Fortify Strength
		if found["Alpha+Gamma"] {
			t.Error("Alpha+Gamma only covers Fortify Speed, not both targets")
		}
	})

	t.Run("no match", func(t *testing.T) {
		combos := idx.FindCombos([]string{"Night Eye"})
		// Delta has Night Eye but nothing else does, so no valid pair or triple
		if len(combos) != 0 {
			t.Errorf("expected no combos for Night Eye (only Delta has it), got %d", len(combos))
		}
	})

	t.Run("two triples sharing the same pair", func(t *testing.T) {
		// ing1+ing2 share C (not a target), so the pair alone doesn't cover {A,B}.
		// Both ing3 and ing4 independently complete the triple — both must appear in results.
		ingredients := []alchemy.Ingredient{
			{Name: "ing1", EffectsNames: []string{"A", "C"}},
			{Name: "ing2", EffectsNames: []string{"B", "C"}},
			{Name: "ing3", EffectsNames: []string{"A", "B"}},
			{Name: "ing4", EffectsNames: []string{"A", "B", "D"}},
		}
		idx2 := alchemy.BuildIndex(ingredients)
		combos := idx2.FindCombos([]string{"A", "B"})
		found := map[string]bool{}
		for _, p := range combos {
			found[strings.Join(p.IngredientsNames(), "+")] = true
		}
		assert.True(t, found["ing1+ing2+ing3"], "expected triple ing1+ing2+ing3")
		assert.True(t, found["ing1+ing2+ing4"], "expected triple ing1+ing2+ing4")
	})
}

var sinkPotions []*alchemy.Potion

func syntheticIndex(n int) *alchemy.Index {
	poolSize := min(128, max(24, 63*n/177)) // match real density (63 effects/177 ings); cap at EffectMask limit
	pool := make([]string, poolSize)
	for i := range pool {
		pool[i] = fmt.Sprintf("Effect%02d", i)
	}
	rng := rand.New(rand.NewSource(42))
	ings := make([]alchemy.Ingredient, n)
	for i := range ings {
		perm := rng.Perm(poolSize)[:4]
		efx := make([]string, 4)
		for j, p := range perm {
			efx[j] = pool[p]
		}
		ings[i] = alchemy.Ingredient{
			Name:         fmt.Sprintf("Ing%04d", i),
			Value:        rng.Intn(200) + 1,
			EffectsNames: efx,
		}
	}
	return alchemy.BuildIndex(ings)
}

func BenchmarkFindCombos(b *testing.B) {
	targets := []string{"Effect00", "Effect01"}
	for _, n := range []int{177, 500, 1000} {
		idx := syntheticIndex(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for b.Loop() {
				sinkPotions = idx.FindCombos(targets)
			}
		})
	}
}

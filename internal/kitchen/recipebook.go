package kitchen

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Ingredient is one entry of a stored recipe.
type Ingredient struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
}

// Recipe is a known recipe from the ovens mythical recipe book.
type Recipe struct {
	Name        string       `json:"name"`
	Cuisine     string       `json:"cuisine"`
	Difficulty  string       `json:"difficulty"`
	PrepMinutes int          `json:"prep_minutes"`
	BakeMinutes int          `json:"bake_minutes"`
	BakeTempC   float64      `json:"bake_temp_c"`
	Servings    int          `json:"servings"`
	Ingredients []Ingredient `json:"ingredients"`
	Steps       []string     `json:"steps"`
	Tools       []string     `json:"tools"`
}

var recipes = map[string]Recipe{
	"pancakes": {
		Name: "pancakes", Cuisine: "international", Difficulty: "easy",
		PrepMinutes: 15, BakeMinutes: 10, BakeTempC: 190, Servings: 4,
		Ingredients: []Ingredient{
			{Name: "flour", Amount: 250, Unit: "g"},
			{Name: "milk", Amount: 300, Unit: "ml"},
			{Name: "egg", Amount: 2, Unit: "count"},
			{Name: "sugar", Amount: 30, Unit: "g"},
			{Name: "butter", Amount: 50, Unit: "g"},
		},
		Steps: []string{
			"Whisk the eggs with the sugar",
			"Fold in the flour gradually while adding milk",
			"Rest the batter so the flour hydrates",
			"Fry thin rounds in a buttered pan",
		},
		Tools: []string{"whisk", "spatula"},
	},
	"cookies": {
		Name: "cookies", Cuisine: "american", Difficulty: "easy",
		PrepMinutes: 20, BakeMinutes: 12, BakeTempC: 180, Servings: 20,
		Ingredients: []Ingredient{
			{Name: "flour", Amount: 280, Unit: "g"},
			{Name: "butter", Amount: 170, Unit: "g"},
			{Name: "sugar", Amount: 150, Unit: "g"},
			{Name: "chocolate-chips", Amount: 200, Unit: "g"},
			{Name: "egg", Amount: 1, Unit: "count"},
		},
		Steps: []string{
			"Cream the butter with the sugar",
			"Mix in the egg, then the flour",
			"Scatter the chocolate chips through the dough",
			"Roll balls and flatten them on a tray",
		},
		Tools: []string{"mixer", "spatula"},
	},
	"chocolate-cake": {
		Name: "chocolate-cake", Cuisine: "international", Difficulty: "medium",
		PrepMinutes: 30, BakeMinutes: 40, BakeTempC: 175, Servings: 10,
		Ingredients: []Ingredient{
			{Name: "flour", Amount: 300, Unit: "g"},
			{Name: "cocoa", Amount: 60, Unit: "g"},
			{Name: "sugar", Amount: 220, Unit: "g"},
			{Name: "butter", Amount: 200, Unit: "g"},
			{Name: "egg", Amount: 3, Unit: "count"},
			{Name: "milk", Amount: 150, Unit: "ml"},
		},
		Steps: []string{
			"Cream the butter with the sugar until pale",
			"Whip in the eggs one at a time",
			"Alternate folds of flour-cocoa and milk",
			"Bake until a skewer comes out almost clean",
		},
		Tools: []string{"mixer", "grill"},
	},
	"banana-bread": {
		Name: "banana-bread", Cuisine: "american", Difficulty: "easy",
		PrepMinutes: 20, BakeMinutes: 55, BakeTempC: 170, Servings: 8,
		Ingredients: []Ingredient{
			{Name: "flour", Amount: 260, Unit: "g"},
			{Name: "banana", Amount: 3, Unit: "count"},
			{Name: "sugar", Amount: 150, Unit: "g"},
			{Name: "butter", Amount: 115, Unit: "g"},
			{Name: "egg", Amount: 2, Unit: "count"},
		},
		Steps: []string{
			"Mash the bananas until mostly smooth",
			"Mix in the butter and the sugar",
			"Beat in the eggs and fold in the flour",
			"Bake until the top springs back",
		},
		Tools: []string{"mixer", "spatula"},
	},
	"sourdough-bread": {
		Name: "sourdough-bread", Cuisine: "french", Difficulty: "hard",
		PrepMinutes: 240, BakeMinutes: 45, BakeTempC: 230, Servings: 8,
		Ingredients: []Ingredient{
			{Name: "flour", Amount: 500, Unit: "g"},
			{Name: "water", Amount: 350, Unit: "ml"},
			{Name: "sourdough-starter", Amount: 100, Unit: "g"},
			{Name: "salt", Amount: 10, Unit: "g"},
		},
		Steps: []string{
			"Knead the flour with the water and the starter",
			"Rest to autolyse until the dough relaxes",
			"Fold in the salt and proof overnight",
			"Steam-bake for a crackling crust",
		},
		Tools: []string{"oven", "grill"},
	},
	"pizza-dough": {
		Name: "pizza-dough", Cuisine: "italian", Difficulty: "medium",
		PrepMinutes: 90, BakeMinutes: 8, BakeTempC: 250, Servings: 4,
		Ingredients: []Ingredient{
			{Name: "flour", Amount: 400, Unit: "g"},
			{Name: "water", Amount: 240, Unit: "ml"},
			{Name: "yeast", Amount: 4, Unit: "g"},
			{Name: "salt", Amount: 8, Unit: "g"},
		},
		Steps: []string{
			"Dissolve the yeast in the water",
			"Knead with the flour and the salt",
			"Proof until doubled in size",
			"Stretch thin and bake on a hot stone",
		},
		Tools: []string{"oven", "rollingpin"},
	},
	"focaccia": {
		Name: "focaccia", Cuisine: "italian", Difficulty: "easy",
		PrepMinutes: 120, BakeMinutes: 20, BakeTempC: 220, Servings: 6,
		Ingredients: []Ingredient{
			{Name: "flour", Amount: 500, Unit: "g"},
			{Name: "water", Amount: 400, Unit: "ml"},
			{Name: "yeast", Amount: 5, Unit: "g"},
			{Name: "salt", Amount: 10, Unit: "g"},
			{Name: "olive-oil", Amount: 60, Unit: "ml"},
		},
		Steps: []string{
			"Mix a wet dough with the oil",
			"Proof in the tray, dimpling with your fingers",
			"Salt the top generously",
			"Bake until golden and oily",
		},
		Tools: []string{"oven", "spatula"},
	},
}

// List returns all known recipe names, sorted.
func List() []string {
	names := make([]string, 0, len(recipes))
	for name := range recipes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

var ErrUnknownRecipe = errors.New("recipe not found in the oven recipe book")

// Lookup fetches a recipe. Unknown names fail with the full list of
// available recipes so learners can self-correct — the provider never
// hides its cookbook.
func Lookup(name string) (Recipe, error) {
	if r, ok := recipes[strings.ToLower(name)]; ok {
		return r, nil
	}
	return Recipe{}, fmt.Errorf("%w: %q (available recipes: %s)", ErrUnknownRecipe, name, strings.Join(List(), ", "))
}

// BakeTimeMinutes scales a recipe's bake time to a serving count.
// The kitchen heuristic: half the time is fixed (oven preheat and
// carry-over), half scales linearly with servings.
func BakeTimeMinutes(recipeName string, servings int) (int, error) {
	if servings < 1 {
		return 0, fmt.Errorf("servings must be at least 1, got %d", servings)
	}
	r, err := Lookup(recipeName)
	if err != nil {
		return 0, err
	}
	base := float64(r.BakeMinutes)
	scaled := 0.5*base + 0.5*base*float64(servings)/float64(r.Servings)
	return int(scaled + 0.5), nil
}

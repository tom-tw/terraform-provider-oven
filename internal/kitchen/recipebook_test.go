package kitchen

import (
	"errors"
	"strings"
	"testing"
)

func TestRecipesBakeTime(t *testing.T) {
	pancakes, err := Lookup("pancakes")
	if err != nil {
		t.Fatalf("Lookup(pancakes): %v", err)
	}
	if pancakes.Name != "pancakes" || pancakes.BakeTempC != 190 || pancakes.PrepMinutes != 15 {
		t.Errorf("unexpected pancakes recipe: %+v", pancakes)
	}
	if len(pancakes.Ingredients) != 5 || len(pancakes.Steps) != 4 || len(pancakes.Tools) != 2 {
		t.Errorf("pancakes ingredient/step/tool counts: %d/%d/%d", len(pancakes.Ingredients), len(pancakes.Steps), len(pancakes.Tools))
	}
}

func TestRecipesBakeTimeCaseInsensitive(t *testing.T) {
	if _, err := Lookup("PANCAKES"); err != nil {
		t.Errorf("lookup should be case-insensitive: %v", err)
	}
}

func TestRecipesLookupUnknownListsAvailable(t *testing.T) {
	_, err := Lookup("birthday-cake")
	if err == nil {
		t.Fatal("unknown recipe should fail")
	}
	if !errors.Is(err, ErrUnknownRecipe) {
		t.Errorf("want ErrUnknownRecipe, got: %v", err)
	}
	for _, want := range List() {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should list %q: %v", want, err)
		}
	}
}

func TestRecipesBakeTimeScalesWithServings(t *testing.T) {
	base, err := BakeTimeMinutes("pancakes", 4)
	if err != nil {
		t.Fatalf("BakeTimeMinutes base: %v", err)
	}
	if base != 10 {
		t.Errorf("BakeTimeMinutes(pancakes, 4) = %d; want 10", base)
	}
	double, err := BakeTimeMinutes("pancakes", 8)
	if err != nil {
		t.Fatalf("BakeTimeMinutes doubled: %v", err)
	}
	if double <= base {
		t.Errorf("more servings should bake longer: %d <= %d", double, base)
	}
	if _, err := BakeTimeMinutes("pancakes", 0); err == nil {
		t.Error("zero servings must fail")
	}
	if _, err := BakeTimeMinutes("unknown-dish", 4); err == nil {
		t.Error("unknown recipe must fail")
	}
}

func TestRecipesListSorted(t *testing.T) {
	list := List()
	if len(list) < 5 {
		t.Fatalf("want at least 5 recipes, got %d", len(list))
	}
	for i := 1; i < len(list); i++ {
		if list[i-1] > list[i] {
			t.Fatalf("List not sorted: %v", list)
		}
	}
	if !slicesContains(list, "cookies") || !slicesContains(list, "sourdough-bread") {
		t.Errorf("List missing expected entries: %v", list)
	}
}

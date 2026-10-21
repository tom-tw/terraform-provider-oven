package kitchen

import (
	"errors"
	"strings"
	"testing"
)

func TestMixRatioClassifiesRatios(t *testing.T) {
	t.Parallel()
	cases := []struct {
		liquid, dry float64
		want        string
	}{
		{300, 200, "runny"},  // 200/300 = 0.67 < 0.8
		{300, 250, "batter"}, // 250/300 = 0.83
		{300, 250.1, "batter"},
		{150, 600, "dough"},
		{0, 100, "powdery"},
		{100, 0, "pool"},
		{0, 0, "empty bowl"},
	}
	for _, c := range cases {
		got, err := MixRatio(c.liquid, c.dry)
		if err != nil {
			t.Fatalf("MixRatio(%v, %v): %v", c.liquid, c.dry, err)
		}
		if got != c.want {
			t.Errorf("MixRatio(%v, %v) = %q; want %q", c.liquid, c.dry, got, c.want)
		}
	}
	if _, err := MixRatio(-1, 5); err == nil {
		t.Error("negative liquid must fail")
	}
}

func TestMixConsistencyUsesIngredientCategories(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ingredients map[string]float64
		want        string
	}{
		{map[string]float64{"milk": 300, "flour": 250, "sugar": 30}, "batter"}, // liquid 300, dry 280 → 0.93

		{map[string]float64{"flour": 500, "water": 350}, "batter"},
		{map[string]float64{"flour": 500, "water": 100}, "dough"},
		{map[string]float64{"water": 400}, "pool"},
		{map[string]float64{"flour": 400}, "powdery"},
		{map[string]float64{"mystery-powder": 50}, "powdery"},
	}
	for _, c := range cases {
		got, err := MixConsistency(c.ingredients)
		if err != nil {
			t.Fatalf("MixConsistency(%v): %v", c.ingredients, err)
		}
		if got != c.want {
			t.Errorf("MixConsistency(%v) = %q; want %q", c.ingredients, got, c.want)
		}
	}
	if _, err := MixConsistency(map[string]float64{"salt": -5}); err == nil {
		t.Error("negative amount must fail")
	}
}

func TestBakeDonenessCategories(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tempC  float64
		minute int
		mode   string
		want   string
	}{
		{75, 10, "static", "barely warmed"},
		{250, 8, "static", "golden"}, // pizza on a hot stone
		{180, 30, "static", "golden"},
		{230, 45, "static", "well-done"}, // sourdough
		{200, 90, "static", "charcoal"},
		{160, 60, "static", "well-done"},   // 1.6 * 60 = 96
		{200, 40, "static", "well-done"},   // 2.0 * 40 = 80
		{200, 40, "steam", "golden"},       // steam carries heat gently: 68
		{220, 30, "grill", "well-done"},    // grill scorches harder: 75.9
	}
	for _, c := range cases {
		got, err := BakeDoneness(c.tempC, c.minute, c.mode)
		if err != nil {
			t.Fatalf("BakeDoneness(%v, %d, %s): %v", c.tempC, c.minute, c.mode, err)
		}
		if got != c.want {
			t.Errorf("BakeDoneness(%v, %d, %s) = %q; want %q", c.tempC, c.minute, c.mode, got, c.want)
		}
	}
	_, err := BakeDoneness(180, 30, "microwave")
	if err == nil {
		t.Fatal("unknown mode must fail")
	}
	if !strings.Contains(err.Error(), "static") {
		t.Errorf("error should list known modes: %v", err)
	}
	if !errors.Is(err, ErrUnknownMode) {
		t.Errorf("want ErrUnknownMode, got: %v", err)
	}
}
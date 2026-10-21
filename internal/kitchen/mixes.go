package kitchen

import (
	"fmt"
	"sort"
)

var liquidIngredients = map[string]bool{
	"water": true, "milk": true, "oil": true, "olive-oil": true,
	"juice": true, "cream": true, "heavy-cream": true, "honey": true,
}

// ErrUnknownMode is returned for oven modes outside the factor table.
var ErrUnknownMode = fmt.Errorf("unknown oven mode")

// BakeModeFactors scales heat application per oven mode.
var BakeModeFactors = map[string]float64{
	"static": 1.0, "convection": 0.95, "steam": 0.85, "grill": 1.15,
}

// MixRatio classifies a batter from its liquid and dry mass totals:
// the classic pastry intuition that the dry/liquid ratio decides if
// you are pouring, spooning, or kneading.
func MixRatio(liquid, dry float64) (string, error) {
	if liquid < 0 || dry < 0 {
		return "", fmt.Errorf("liquid and dry amounts must not be negative")
	}
	switch {
	case liquid > 0 && dry == 0:
		return "pool", nil
	case liquid == 0 && dry > 0:
		return "powdery", nil
	case liquid == 0 && dry == 0:
		return "empty bowl", nil
	}
	ratio := dry / liquid
	switch {
	case ratio > 2:
		return "dough", nil
	case ratio < 0.8:
		return "runny", nil
	default:
		return "batter", nil
	}
}

// MixConsistency sums the liquid and dry ingredients in a mix and
// classifies the result. Unknown ingredients count as dry: the oven
// provider assumes everything in a bowl thickens it.
func MixConsistency(ingredients map[string]float64) (string, error) {
	var liquid, dry float64
	for name, amount := range ingredients {
		if amount < 0 {
			return "", fmt.Errorf("ingredient %q has a negative amount", name)
		}
		if liquidIngredients[name] {
			liquid += amount
		} else {
			dry += amount
		}
	}
	return MixRatio(liquid, dry)
}

func bakeDonenessLabel(heat float64) string {
	switch {
	case heat < 15:
		return "barely warmed"
	case heat < 75:
		return "golden"
	case heat < 110:
		return "well-done"
	case heat < 140:
		return "dark-crust"
	default:
		return "charcoal"
	}
}

// BakeDoneness estimates how a bake turns out from temperature,
// duration and oven mode. The heat index is roughly what a pastry
// book would call "oven minutes per percent of temperature".
func BakeDoneness(tempC float64, minutes int, mode string) (string, error) {
	if minutes < 0 {
		return "", fmt.Errorf("duration must not be negative, got %d", minutes)
	}
	factor, ok := BakeModeFactors[mode]
	if !ok {
		modes := make([]string, 0, len(BakeModeFactors))
		for m := range BakeModeFactors {
			modes = append(modes, m)
		}
		sort.Strings(modes)
		return "", fmt.Errorf("%w %q (known modes: %v)", ErrUnknownMode, mode, modes)
	}
	return bakeDonenessLabel((tempC / 100) * float64(minutes) * factor), nil
}

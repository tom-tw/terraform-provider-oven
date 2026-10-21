package kitchen

import "fmt"

type unitCategory int

const (
	categoryMass unitCategory = iota
	categoryVolume
)

// unitTable holds conversion factors to each category's base unit
// (grams for mass, millilitres for volume).
var unitTable = map[string]struct {
	factor   float64
	category unitCategory
}{
	"g":    {1, categoryMass},
	"kg":   {1000, categoryMass},
	"oz":   {28.349523, categoryMass},
	"lb":   {453.59237, categoryMass},
	"ml":   {1, categoryVolume},
	"l":    {1000, categoryVolume},
	"floz": {29.573530, categoryVolume},
	"cup":  {236.588236, categoryVolume},
	"tsp":  {4.928922, categoryVolume},
	"tbsp": {14.786765, categoryVolume},
}

var (
	// ErrDimensionMismatch is returned when converting between mass
	// and volume: the kitchen refuses to turn grams into cups.
	ErrDimensionMismatch = fmt.Errorf("units of different dimensions cannot be converted")
	// ErrUnknownUnit is returned for units outside the pantry table.
	ErrUnknownUnit = fmt.Errorf("unknown unit")
)

// CelsiusToFahrenheit converts oven temperatures.
func CelsiusToFahrenheit(c float64) float64 { return c*9/5 + 32 }

// FahrenheitToCelsius converts oven temperatures back.
func FahrenheitToCelsius(f float64) float64 { return (f - 32) * 5 / 9 }

// ConvertUnits converts an amount between mass units or between
// volume units. Mixing dimensions is a classic kitchen disaster.
func ConvertUnits(amount float64, from, to string) (float64, error) {
	f, ok := unitTable[from]
	if !ok {
		return 0, fmt.Errorf("%w: %q is not in the pantry (known: g, kg, oz, lb, ml, l, floz, cup, tsp, tbsp)", ErrUnknownUnit, from)
	}
	t, ok := unitTable[to]
	if !ok {
		return 0, fmt.Errorf("%w: %q is not in the pantry (known: g, kg, oz, lb, ml, l, floz, cup, tsp, tbsp)", ErrUnknownUnit, to)
	}
	if f.category != t.category {
		return 0, fmt.Errorf("%w: cannot convert %s to %s", ErrDimensionMismatch, from, to)
	}
	return amount * f.factor / t.factor, nil
}

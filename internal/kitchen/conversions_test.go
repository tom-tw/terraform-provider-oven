package kitchen

import (
	"errors"
	"math"
	"testing"
)

func floatEquals(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}

func TestTemperatureConversions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		celsius float64
		wantF   float64
	}{
		{0, 32},
		{100, 212},
		{180, 356},
		{-40, -40}, // the famous crossover point
		{37, 98.6},
	}
	for _, c := range cases {
		if got := CelsiusToFahrenheit(c.celsius); !floatEquals(got, c.wantF) {
			t.Errorf("CelsiusToFahrenheit(%v) = %v; want %v", c.celsius, got, c.wantF)
		}
		back := FahrenheitToCelsius(c.wantF)
		if !floatEquals(back, c.celsius) {
			t.Errorf("FahrenheitToCelsius(%v) = %v; want %v", c.wantF, back, c.celsius)
		}
	}
}

func TestConvertUnitsMass(t *testing.T) {
	t.Parallel()
	cases := []struct {
		amount float64
		from   string
		to     string
		want   float64
	}{
		{100, "g", "oz", 3.5273962},
		{1, "kg", "lb", 2.2046226},
		{16, "oz", "g", 453.592368},
		{5000, "g", "kg", 5},
	}
	for _, c := range cases {
		got, err := ConvertUnits(c.amount, c.from, c.to)
		if err != nil {
			t.Fatalf("ConvertUnits(%v %s -> %s): %v", c.amount, c.from, c.to, err)
		}
		if !floatEquals(got, c.want) {
			t.Errorf("ConvertUnits(%v %s -> %s) = %v; want %v", c.amount, c.from, c.to, got, c.want)
		}
	}
}

func TestConvertUnitsVolume(t *testing.T) {
	t.Parallel()
	cases := []struct {
		amount float64
		from   string
		to     string
		want   float64
	}{
		{1, "cup", "ml", 236.588236},
		{500, "ml", "l", 0.5},
		{2, "tsp", "tbsp", 0.6666667},
		{1, "l", "floz", 33.8140227},
	}
	for _, c := range cases {
		got, err := ConvertUnits(c.amount, c.from, c.to)
		if err != nil {
			t.Fatalf("ConvertUnits(%v %s -> %s): %v", c.amount, c.from, c.to, err)
		}
		if !floatEquals(got, c.want) {
			t.Errorf("ConvertUnits(%v %s -> %s) = %v; want %v", c.amount, c.from, c.to, got, c.want)
		}
	}
}

func TestConvertUnitsRejectsMassVolumeMix(t *testing.T) {
	t.Parallel()
	_, err := ConvertUnits(100, "g", "ml")
	if err == nil {
		t.Fatal("converting grams to millilitres must not be possible")
	}
	if !errors.Is(err, ErrDimensionMismatch) {
		t.Errorf("want ErrDimensionMismatch, got: %v", err)
	}
}

func TestConvertUnitsRejectsUnknownUnits(t *testing.T) {
	t.Parallel()
	if _, err := ConvertUnits(1, "bananas", "kg"); !errors.Is(err, ErrUnknownUnit) {
		t.Errorf("want ErrUnknownUnit for unknown from, got: %v", err)
	}
	if _, err := ConvertUnits(1, "g", "handfuls"); !errors.Is(err, ErrUnknownUnit) {
		t.Errorf("want ErrUnknownUnit for unknown to, got: %v", err)
	}
}

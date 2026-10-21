package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/tom-tw/terraform-provider-oven/internal/kitchen"
)

var _ function.Function = &convertTempFunction{}

// NewConvertTempFunction returns the oven::convert_temp provider
// function (OpenTofu 1.7+ provider functions).
func NewConvertTempFunction() function.Function {
	return &convertTempFunction{}
}

type convertTempFunction struct{}

func (f *convertTempFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "convert_temp"
}

func (f *convertTempFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Convert a temperature between Celsius and Fahrenheit.",
		MarkdownDescription: "Given a numeric temperature and source/target units (`C` or `F`), returns the converted value. Call from OpenTofu as `provider::oven::convert_temp(...)`.",
		Parameters: []function.Parameter{
			function.Float64Parameter{
				Name:        "value",
				Description: "Temperature value to convert.",
			},
			function.StringParameter{
				Name:        "from",
				Description: "Source unit: `C` or `F` (case-insensitive).",
			},
			function.StringParameter{
				Name:        "to",
				Description: "Target unit: `C` or `F` (case-insensitive).",
			},
		},
		Return: function.Float64Return{},
	}
}

func (f *convertTempFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value float64
	var from, to string

	resp.Error = req.Arguments.Get(ctx, &value, &from, &to)
	if resp.Error != nil {
		return
	}

	if err := checkTempUnit(from); err != nil {
		resp.Error = function.NewArgumentFuncError(1, err.Error())
		return
	}
	if err := checkTempUnit(to); err != nil {
		resp.Error = function.NewArgumentFuncError(2, err.Error())
		return
	}

	out := value
	switch {
	case from == "C" && to == "F":
		out = kitchen.CelsiusToFahrenheit(value)
	case from == "F" && to == "C":
		out = kitchen.FahrenheitToCelsius(value)
	}

	resp.Error = resp.Result.Set(ctx, out)
}

func checkTempUnit(unit string) error {
	if u := strings.ToUpper(unit); u != "C" && u != "F" {
		return fmt.Errorf("unknown temperature unit %q: use C or F", unit)
	}
	return nil
}

var _ function.Function = &convertUnitsFunction{}

// NewConvertUnitsFunction returns the oven::convert_units function.
func NewConvertUnitsFunction() function.Function {
	return &convertUnitsFunction{}
}

type convertUnitsFunction struct{}

func (f *convertUnitsFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "convert_units"
}

func (f *convertUnitsFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Convert kitchen amounts between mass or volume units.",
		MarkdownDescription: "Mass units: `g`, `kg`, `oz`, `lb`. Volume units: `ml`, `l`, `floz`, `cup`, `tsp`, `tbsp`. Converting grams into cups is impossible everywhere, including here.",
		Parameters: []function.Parameter{
			function.Float64Parameter{
				Name:        "amount",
				Description: "Amount to convert.",
			},
			function.StringParameter{
				Name:        "from",
				Description: "Source unit.",
			},
			function.StringParameter{
				Name:        "to",
				Description: "Target unit.",
			},
		},
		Return: function.Float64Return{},
	}
}

func (f *convertUnitsFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var amount float64
	var from, to string

	resp.Error = req.Arguments.Get(ctx, &amount, &from, &to)
	if resp.Error != nil {
		return
	}

	out, err := kitchen.ConvertUnits(amount, from, to)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = resp.Result.Set(ctx, out)
}

var _ function.Function = &bakeTimeFunction{}

// NewBakeTimeFunction returns the oven::bake_time function.
func NewBakeTimeFunction() function.Function {
	return &bakeTimeFunction{}
}

type bakeTimeFunction struct{}

func (f *bakeTimeFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "bake_time"
}

func (f *bakeTimeFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Estimate bake minutes for a recipe scaled to servings.",
		MarkdownDescription: "Looks up the recipe in the oven recipe book and scales its bake time to the requested servings: half the time is fixed heat-up, half scales linearly.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "recipe_name",
				Description: "Recipe from the recipe book, e.g. `pancakes`.",
			},
			function.Float64Parameter{
				Name:        "servings",
				Description: "Whole number of servings to bake for.",
			},
		},
		Return: function.Float64Return{},
	}
}

func (f *bakeTimeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var recipeName string
	var servings float64

	resp.Error = req.Arguments.Get(ctx, &recipeName, &servings)
	if resp.Error != nil {
		return
	}

	if servings != float64(int(servings)) {
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("servings must be a whole number, got %v", servings))
		return
	}

	minutes, err := kitchen.BakeTimeMinutes(recipeName, int(servings))
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = resp.Result.Set(ctx, float64(minutes))
}

var _ function.Function = &mixRatioFunction{}

// NewMixRatioFunction returns the oven::mix_ratio function.
func NewMixRatioFunction() function.Function {
	return &mixRatioFunction{}
}

type mixRatioFunction struct{}

func (f *mixRatioFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "mix_ratio"
}

func (f *mixRatioFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Classify a mix from its liquid and dry totals.",
		MarkdownDescription: "Given liquid and dry mass totals in grams, returns the pastry verdict: `pool`, `powdery`, `runny`, `batter`, `dough` or `empty bowl`.",
		Parameters: []function.Parameter{
			function.Float64Parameter{
				Name:        "liquid",
				Description: "Total liquid mass in grams.",
			},
			function.Float64Parameter{
				Name:        "dry",
				Description: "Total dry mass in grams.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *mixRatioFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var liquid, dry float64

	resp.Error = req.Arguments.Get(ctx, &liquid, &dry)
	if resp.Error != nil {
		return
	}

	verdict, err := kitchen.MixRatio(liquid, dry)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = resp.Result.Set(ctx, string(verdict))
}

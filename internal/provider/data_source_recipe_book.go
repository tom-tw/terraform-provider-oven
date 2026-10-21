package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/tom-tw/terraform-provider-oven/internal/kitchen"
)

var _ datasource.DataSource = &recipeBookDataSource{}

// NewRecipeBookDataSource returns the oven_recipe_book data source.
func NewRecipeBookDataSource() datasource.DataSource {
	return &recipeBookDataSource{}
}

type recipeBookDataSource struct {
	kitchen *Kitchen
}

type recipeIngredientEntry struct {
	Name   string  `tfsdk:"name"`
	Amount float64 `tfsdk:"amount"`
	Unit   string  `tfsdk:"unit"`
}

type recipeBookModel struct {
	ID               types.String            `tfsdk:"id"`
	RecipeName       types.String            `tfsdk:"recipe_name"`
	Cuisine          types.String            `tfsdk:"cuisine"`
	Difficulty       types.String            `tfsdk:"difficulty"`
	Servings         types.Int64             `tfsdk:"servings"`
	PrepMinutes      types.Int64             `tfsdk:"prep_minutes"`
	BakeMinutes      types.Int64             `tfsdk:"bake_minutes"`
	BakeTempC        types.Float64           `tfsdk:"bake_temp_c"`
	BakeTempF        types.Float64           `tfsdk:"bake_temp_f"`
	Ingredients      []recipeIngredientEntry `tfsdk:"ingredients"`
	Steps            []string                `tfsdk:"steps"`
	Tools            []string                `tfsdk:"tools"`
	AvailableRecipes []string                `tfsdk:"available_recipes"`
}

func (r *recipeBookDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_recipe_book"
}

func (r *recipeBookDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up one of the recipes the oven keeps in its mythical recipe book. " +
			"Unknown names fail with the full list of dishes it knows, like a friendly librarian.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Same as the recipe name.",
			},
			"recipe_name": schema.StringAttribute{
				Required:    true,
				Description: "Which dish to look up, e.g. `pancakes`.",
			},
			"cuisine": schema.StringAttribute{
				Computed:    true,
				Description: "Imaginary cuisine of origin.",
			},
			"difficulty": schema.StringAttribute{
				Computed:    true,
				Description: "One of `easy`, `medium`, `hard`.",
			},
			"servings": schema.Int64Attribute{
				Computed:    true,
				Description: "Servings the recipe book assumes.",
			},
			"prep_minutes": schema.Int64Attribute{
				Computed:    true,
				Description: "Preparation time in minutes.",
			},
			"bake_minutes": schema.Int64Attribute{
				Computed:    true,
				Description: "Baking time in minutes.",
			},
			"bake_temp_c": schema.Float64Attribute{
				Computed:    true,
				Description: "Baking temperature in Celsius.",
			},
			"bake_temp_f": schema.Float64Attribute{
				Computed:    true,
				Description: "Baking temperature in Fahrenheit, for the metrically confused.",
			},
			"ingredients": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Ingredients from the recipe book.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":   schema.StringAttribute{Computed: true},
						"amount": schema.Float64Attribute{Computed: true},
						"unit":   schema.StringAttribute{Computed: true},
					},
				},
			},
			"steps": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Ordered preparation steps.",
			},
			"tools": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Kitchen tools the recipe expects you to have.",
			},
			"available_recipes": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Every dish the recipe book knows. Always populated, very handy with unknown names.",
			},
		},
	}
}

func (r *recipeBookDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	r.kitchen = kitchenFromProviderData(req.ProviderData, "data source.Configure", &resp.Diagnostics)
}

func (r *recipeBookDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data recipeBookModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.RecipeName.ValueString()

	recipe, err := kitchen.Lookup(name)
	if err != nil {
		resp.Diagnostics.AddError("Recipe not found", err.Error())
		return
	}

	ingredients := make([]recipeIngredientEntry, 0, len(recipe.Ingredients))
	for _, ing := range recipe.Ingredients {
		ingredients = append(ingredients, recipeIngredientEntry{
			Name:   ing.Name,
			Amount: ing.Amount,
			Unit:   ing.Unit,
		})
	}

	data.ID = types.StringValue(name)
	data.Cuisine = types.StringValue(recipe.Cuisine)
	data.Difficulty = types.StringValue(recipe.Difficulty)
	data.Servings = types.Int64Value(int64(recipe.Servings))
	data.PrepMinutes = types.Int64Value(int64(recipe.PrepMinutes))
	data.BakeMinutes = types.Int64Value(int64(recipe.BakeMinutes))
	data.BakeTempC = types.Float64Value(recipe.BakeTempC)
	data.BakeTempF = types.Float64Value(kitchen.CelsiusToFahrenheit(recipe.BakeTempC))
	data.Ingredients = ingredients
	data.Steps = recipe.Steps
	data.Tools = recipe.Tools
	data.AvailableRecipes = kitchen.List()

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

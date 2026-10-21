package provider

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/tom-tw/terraform-provider-oven/internal/kitchen"
)

var _ datasource.DataSource = &ovenDataSource{}

// NewOvenDataSource returns the oven_oven data source.
func NewOvenDataSource() datasource.DataSource {
	return &ovenDataSource{}
}

type ovenDataSource struct {
	kitchen *Kitchen
}

type ovenDataSourceModel struct {
	ID             types.String  `tfsdk:"id"`
	KitchenName    types.String  `tfsdk:"kitchen_name"`
	OvenModel      types.String  `tfsdk:"oven_model"`
	MaxTemperature types.Float64 `tfsdk:"max_temperature"`
	ChaosLevel     types.Float64 `tfsdk:"chaos_level"`
	BakeModes      []string      `tfsdk:"bake_modes"`
	ToolKinds      []string      `tfsdk:"tool_kinds"`
	PrepModes      []string      `tfsdk:"prep_modes"`
}

func (r *ovenDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oven"
}

func (r *ovenDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Describes the currently configured kitchen and its imaginary oven: " +
			"useful to inspect what the provider block actually set up.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Same as the kitchen name.",
			},
			"kitchen_name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the imaginary kitchen.",
			},
			"oven_model": schema.StringAttribute{
				Computed:    true,
				Description: "Decorative oven model.",
			},
			"max_temperature": schema.Float64Attribute{
				Computed:    true,
				Description: "Highest reachable temperature in Celsius.",
			},
			"chaos_level": schema.Float64Attribute{
				Computed:    true,
				Description: "Current probability of a random kitchen accident.",
			},
			"bake_modes": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Oven modes accepted by oven_bake.",
			},
			"tool_kinds": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tool kinds accepted by oven_tool.",
			},
			"prep_modes": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Preparation modes accepted by oven_prep.",
			},
		},
	}
}

func (r *ovenDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	r.kitchen = kitchenFromProviderData(req.ProviderData, "data source.Configure", &resp.Diagnostics)
}

func (r *ovenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if r.kitchen == nil {
		resp.Diagnostics.AddError("Kitchen not configured", "The oven provider has no kitchen data; this is a provider bug.")
		return
	}

	bakeModes := make([]string, 0, len(kitchen.BakeModeFactors))
	for mode := range kitchen.BakeModeFactors {
		bakeModes = append(bakeModes, mode)
	}

	data := ovenDataSourceModel{
		ID:             types.StringValue(r.kitchen.KitchenName),
		KitchenName:    types.StringValue(r.kitchen.KitchenName),
		OvenModel:      types.StringValue(r.kitchen.OvenModel),
		MaxTemperature: types.Float64Value(r.kitchen.MaxTemperature),
		ChaosLevel:     types.Float64Value(r.kitchen.Chaos.Level),
		BakeModes:      sortedStrings(bakeModes),
		ToolKinds:      sortedStrings(ToolKinds),
		PrepModes:      sortedStrings(prepModes),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// sortedStrings returns a sorted copy of a string slice.
func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

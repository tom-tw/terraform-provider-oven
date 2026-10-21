package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/tom-tw/terraform-provider-oven/internal/kitchen"
)

var (
	_ resource.Resource                = &bakeResource{}
	_ resource.ResourceWithImportState = &bakeResource{}
)

// NewBakeResource returns the oven_bake resource.
func NewBakeResource() resource.Resource {
	return &bakeResource{}
}

type bakeResource struct {
	kitchen *Kitchen
}

type bakeResourceModel struct {
	ID              types.String  `tfsdk:"id"`
	Name            types.String  `tfsdk:"name"`
	Temperature     types.Float64 `tfsdk:"temperature"`
	DurationMinutes types.Int64   `tfsdk:"duration_minutes"`
	Mode            types.String  `tfsdk:"mode"`
	Ingredients     types.List    `tfsdk:"ingredients"`
	Doneness        types.String  `tfsdk:"doneness"`
}

func (r *bakeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bake"
}

func (r *bakeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Bakes something at a temperature for a duration in a given oven mode. " +
			"This is the most chaotic resource: ovens misbehave (gas, door peeks, power cuts) " +
			"so retries are a normal part of life here. Retrying usually succeeds by attempt 3.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Baking tray, in the form `bake:<name>`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Bake name, e.g. `batch-01-cookies`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"temperature": schema.Float64Attribute{
				Required:    true,
				Description: "Baking temperature in Celsius. Above the oven maximum it fails every time.",
				Validators:  []validator.Float64{float64validator.Between(30, 400)},
			},
			"duration_minutes": schema.Int64Attribute{
				Required:    true,
				Description: "Baking duration in minutes.",
				Validators:  []validator.Int64{int64validator.Between(1, 720)},
			},
			"mode": schema.StringAttribute{
				Required:    true,
				Description: fmt.Sprintf("Oven mode: %s.", modeNames()),
				Validators:  []validator.String{stringvalidator.OneOf(bakeModes()...)},
			},
			"ingredients": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Names of the prepared resources going into the oven together.",
			},
			"doneness": schema.StringAttribute{
				Computed:    true,
				Description: "Heat-index verdict: `barely warmed`, `golden`, `well-done`, `dark-crust` or `charcoal`. Recomputed whenever temperature, duration or mode changes.",
			},
		},
	}
}

func bakeModes() []string {
	modes := make([]string, 0, len(kitchen.BakeModeFactors))
	for mode := range kitchen.BakeModeFactors {
		modes = append(modes, mode)
	}
	return modes
}

func modeNames() string {
	return strings.Join(bakeModes(), ", ")
}

func (r *bakeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.kitchen = kitchenFromProviderData(req.ProviderData, "resource.Configure", &resp.Diagnostics)
}

func (r *bakeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bakeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.kitchen == nil {
		resp.Diagnostics.AddError("Kitchen not configured", "The oven provider has no kitchen data; this is a provider bug.")
		return
	}

	name := plan.Name.ValueString()
	temperature := plan.Temperature.ValueFloat64()
	duration := plan.DurationMinutes.ValueInt64()
	mode := plan.Mode.ValueString()

	if temperature > r.kitchen.MaxTemperature {
		resp.Diagnostics.AddError("Oven too small for this dream",
			fmt.Sprintf("You asked for %.0f°C but the %s (model %s) tops out at %.0f°C. "+
				"Retry will not help: lower the temperature or upgrade the oven_model.", temperature, r.kitchen.KitchenName, r.kitchen.OvenModel, r.kitchen.MaxTemperature))
		return
	}
	if !r.kitchen.chaosRoll("oven_bake", name, &resp.Diagnostics) {
		return
	}

	doneness, err := kitchen.BakeDoneness(temperature, int(duration), mode)
	if err != nil {
		resp.Diagnostics.AddError("Impossible bake", err.Error())
		return
	}

	plan.ID = types.StringValue("bake:" + name)
	plan.Doneness = types.StringValue(doneness)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bakeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// What is baked stays baked.
}

func (r *bakeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan bakeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.Temperature.ValueFloat64() > r.kitchen.MaxTemperature {
		resp.Diagnostics.AddError("Oven too small for this dream",
			fmt.Sprintf("You asked for %.0f°C but the %s (model %s) tops out at %.0f°C. "+
				"Retry will not help: lower the temperature or upgrade the oven_model.", plan.Temperature.ValueFloat64(), r.kitchen.KitchenName, r.kitchen.OvenModel, r.kitchen.MaxTemperature))
		return
	}
	if !r.kitchen.chaosRoll("oven_bake", plan.Name.ValueString(), &resp.Diagnostics) {
		return
	}
	doneness, err := kitchen.BakeDoneness(plan.Temperature.ValueFloat64(), int(plan.DurationMinutes.ValueInt64()), plan.Mode.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Impossible bake", err.Error())
		return
	}
	plan.Doneness = types.StringValue(doneness)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bakeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Eating the result always succeeds.
}

func (r *bakeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	var id string
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), strings.TrimPrefix(id, "bake:"))...)
}

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = &recipeResource{}
	_ resource.ResourceWithImportState = &recipeResource{}
)

// NewRecipeResource returns the oven_recipe resource.
func NewRecipeResource() resource.Resource {
	return &recipeResource{}
}

type recipeResource struct {
	kitchen *Kitchen
}

type recipeStepModel struct {
	Order       types.Int64  `tfsdk:"order"`
	Description types.String `tfsdk:"description"`
	Mode        types.String `tfsdk:"mode"`
}

type recipeResourceModel struct {
	ID              types.String      `tfsdk:"id"`
	Name            types.String      `tfsdk:"name"`
	Servings        types.Int64       `tfsdk:"servings"`
	Temperature     types.Float64     `tfsdk:"temperature"`
	DurationMinutes types.Int64       `tfsdk:"duration_minutes"`
	Steps           []recipeStepModel `tfsdk:"steps"`
	Outcome         types.String      `tfsdk:"outcome"`
}

func (r *recipeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_recipe"
}

func (r *recipeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Declares a recipe: servings, oven temperature, duration and nested `steps`. " +
			"The steps block is the classic playground for `dynamic` blocks in OpenTofu. " +
			"Asking for more heat than the oven can deliver fails deterministically.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Recipe card, in the form `recipe:<name>`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Recipe name, e.g. `birthday-cake`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"servings": schema.Int64Attribute{
				Required:    true,
				Description: "How many people will pretend to eat it.",
				Validators:  []validator.Int64{int64validator.Between(1, 500)},
			},
			"temperature": schema.Float64Attribute{
				Optional:    true,
				Description: "Oven temperature in Celsius. Must not exceed the provider's max_temperature.",
			},
			"duration_minutes": schema.Int64Attribute{
				Optional:    true,
				Description: "Total cooking time in minutes.",
				Validators:  []validator.Int64{int64validator.Between(1, 2880)},
			},
			"outcome": schema.StringAttribute{
				Computed:      true,
				Description:   "The chef's verdict: `picture-perfect`, `golden-brown`, `soggy-bottom` or `needs-practice`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
		Blocks: map[string]schema.Block{
			"steps": schema.ListNestedBlock{
				MarkdownDescription: "Ordered preparation steps.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"order": schema.Int64Attribute{
							Required:    true,
							Description: "Step position, starting at 1.",
						},
						"description": schema.StringAttribute{
							Required:    true,
							Description: "What to do in this step.",
						},
						"mode": schema.StringAttribute{
							Optional:    true,
							Description: fmt.Sprintf("Optional prep mode: %s.", strings.Join(prepModes, ", ")),
							Validators:  []validator.String{stringvalidator.OneOf(prepModes...)},
						},
					},
				},
			},
		},
	}
}

func (r *recipeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.kitchen = kitchenFromProviderData(req.ProviderData, "resource.Configure", &resp.Diagnostics)
}

func (r *recipeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan recipeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.kitchen == nil {
		resp.Diagnostics.AddError("Kitchen not configured", "The oven provider has no kitchen data; this is a provider bug.")
		return
	}

	name := plan.Name.ValueString()

	checkRecipeTemperature(&resp.Diagnostics, r.kitchen, plan.Temperature)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.kitchen.chaosRoll("oven_recipe", name, &resp.Diagnostics) {
		return
	}

	plan.ID = types.StringValue("recipe:" + name)
	plan.Outcome = types.StringValue(outcomeFor(r.kitchen.KitchenName, name))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *recipeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Recipe cards do not change on their own.
}

func (r *recipeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan recipeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	checkRecipeTemperature(&resp.Diagnostics, r.kitchen, plan.Temperature)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.kitchen.chaosRoll("oven_recipe", plan.Name.ValueString(), &resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *recipeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Erasing a recipe card always succeeds.
}

func (r *recipeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	var id string
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), strings.TrimPrefix(id, "recipe:"))...)
}

// checkRecipeTemperature is a deterministic failure: the oven simply
// cannot reach that temperature, no matter how many times you retry.
func checkRecipeTemperature(diags *diag.Diagnostics, k *Kitchen, temperature types.Float64) {
	if temperature.IsNull() || k == nil {
		return
	}
	if t := temperature.ValueFloat64(); t > k.MaxTemperature {
		diags.AddError("Oven too small for this dream",
			fmt.Sprintf("You asked for %.0f°C but the %s (model %s) tops out at %.0f°C. "+
				"Retry will not help: lower the temperature or upgrade the oven_model.", t, k.KitchenName, k.OvenModel, k.MaxTemperature))
	}
}

// outcomeFor is the deterministic playful verdict for a recipe.
func outcomeFor(kitchenName, name string) string {
	roll := kitchen.HashRoll("outcome|" + kitchenName + "|" + name)
	switch {
	case roll > 0.85:
		return "picture-perfect"
	case roll > 0.55:
		return "golden-brown"
	case roll > 0.25:
		return "soggy-bottom"
	default:
		return "needs-practice"
	}
}

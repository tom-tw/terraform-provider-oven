package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/tom-tw/terraform-provider-oven/internal/kitchen"
)

var _ resource.Resource = &mixResource{}

// NewMixResource returns the oven_mix resource.
func NewMixResource() resource.Resource {
	return &mixResource{}
}

type mixResource struct {
	kitchen *Kitchen
}

type mixResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Ingredients types.Map    `tfsdk:"ingredients"`
	Consistency types.String `tfsdk:"consistency"`
}

func (r *mixResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mix"
}

func (r *mixResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Combines ingredients (name → grams) in a bowl. The kitchen classifies the " +
			"`consistency` for you: `runny`, `batter`, `dough`, `pool`, `powdery` or `empty bowl`. " +
			"Liquids are water, milk, oil, olive-oil, juice, cream, heavy-cream and honey; everything " +
			"else counts as dry matter.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Bowl label, in the form `mix:<name>`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Bowl name, e.g. `pancake-batter`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"ingredients": schema.MapAttribute{
				Required:    true,
				ElementType: types.Float64Type,
				Description: "Ingredient name to amount in grams.",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
					mapvalidator.ValueFloat64sAre(float64validator.Between(0.01, 100000)),
				},
			},
			"consistency": schema.StringAttribute{
				Computed:    true,
				Description: "What the batter feels like right now. Recomputed whenever the ingredients change.",
			},
		},
	}
}

func (r *mixResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.kitchen = kitchenFromProviderData(req.ProviderData, "resource.Configure", &resp.Diagnostics)
}

func (r *mixResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mixResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.kitchen == nil {
		resp.Diagnostics.AddError("Kitchen not configured", "The oven provider has no kitchen data; this is a provider bug.")
		return
	}

	name := plan.Name.ValueString()

	amounts := map[string]float64{}
	resp.Diagnostics.Append(plan.Ingredients.ElementsAs(ctx, &amounts, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	consistency, err := kitchen.MixConsistency(amounts)
	if err != nil {
		resp.Diagnostics.AddError("Impossible mix", err.Error())
		return
	}

	if !r.kitchen.chaosRoll("oven_mix", name, &resp.Diagnostics) {
		return
	}

	plan.ID = types.StringValue("mix:" + name)
	plan.Consistency = types.StringValue(consistency)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *mixResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Batters do not drift; the bowl remembers.
}

func (r *mixResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mixResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	amounts := map[string]float64{}
	resp.Diagnostics.Append(plan.Ingredients.ElementsAs(ctx, &amounts, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	consistency, err := kitchen.MixConsistency(amounts)
	if err != nil {
		resp.Diagnostics.AddError("Impossible mix", err.Error())
		return
	}
	if !r.kitchen.chaosRoll("oven_mix", plan.Name.ValueString(), &resp.Diagnostics) {
		return
	}
	plan.Consistency = types.StringValue(consistency)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *mixResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Washing the bowl always succeeds.
}

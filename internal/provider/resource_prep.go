package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &prepResource{}

// NewPrepResource returns the oven_prep resource.
func NewPrepResource() resource.Resource {
	return &prepResource{}
}

// prepModes are the preparation modes the kitchen supports.
var prepModes = []string{"mix", "knead", "fold", "rest", "whip", "proof"}

// techniqueNotes is the playful summary shown after each prep.
var techniqueNotes = map[string]string{
	"mix":   "everything swirled together happily",
	"knead": "gluten networks awake and strong",
	"fold":  "air bubbles preserved like little secrets",
	"rest":  "the dough relaxed and stopped sulking",
	"whip":  "volume doubled, arms forgiven",
	"proof": "the aroma developed beautifully",
}

type prepResource struct {
	kitchen *Kitchen
}

type prepResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Mode            types.String `tfsdk:"mode"`
	DurationMinutes types.Int64  `tfsdk:"duration_minutes"`
	Ingredients     types.List   `tfsdk:"ingredients"`
	Done            types.Bool   `tfsdk:"done"`
	Technique       types.String `tfsdk:"technique"`
}

func (r *prepResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prep"
}

func (r *prepResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Performs one preparation step, like mixing or kneading. " +
			"Some mode/duration combinations are plain wrong and fail every time, " +
			"just like over-kneading does in real life.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Prep ticket, in the form `prep:<name>`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Step name, e.g. `knead-base`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"mode": schema.StringAttribute{
				Required:    true,
				Description: fmt.Sprintf("Preparation mode: %s.", strings.Join(prepModes, ", ")),
				Validators:  []validator.String{stringvalidator.OneOf(prepModes...)},
			},
			"duration_minutes": schema.Int64Attribute{
				Required:    true,
				Description: "How long the step lasts, in minutes.",
				Validators:  []validator.Int64{int64validator.Between(1, 1440)},
			},
			"ingredients": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Names of the oven_ingredient resources this step uses.",
			},
			"done": schema.BoolAttribute{
				Computed:      true,
				Description:   "True once the prep is finished, which for a simulated kitchen is immediately.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"technique": schema.StringAttribute{
				Computed:    true,
				Description: "The chef's note about how the mode went. Follows mode changes.",
			},
		},
	}
}

func (r *prepResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.kitchen = kitchenFromProviderData(req.ProviderData, "resource.Configure", &resp.Diagnostics)
}

func (r *prepResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan prepResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.kitchen == nil {
		resp.Diagnostics.AddError("Kitchen not configured", "The oven provider has no kitchen data; this is a provider bug.")
		return
	}

	name := plan.Name.ValueString()
	mode := plan.Mode.ValueString()

	validatePrepUsage(&resp.Diagnostics, mode, plan.DurationMinutes.ValueInt64())
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.kitchen.chaosRoll("oven_prep", name, &resp.Diagnostics) {
		return
	}

	plan.ID = types.StringValue("prep:" + name)
	plan.Done = types.BoolValue(true)
	plan.Technique = types.StringValue(techniqueNotes[mode])

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *prepResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Past preparations stay prepared.
}

func (r *prepResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan prepResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validatePrepUsage(&resp.Diagnostics, plan.Mode.ValueString(), plan.DurationMinutes.ValueInt64())
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.kitchen.chaosRoll("oven_prep", plan.Name.ValueString(), &resp.Diagnostics) {
		return
	}
	plan.Done = types.BoolValue(true)
	plan.Technique = types.StringValue(techniqueNotes[plan.Mode.ValueString()])
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *prepResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Cleaning the bowl always succeeds.
}

// validatePrepUsage fails deterministically for nonsensical
// mode/duration combinations. This teaches the classic Terraform
// experience of provider-side validation errors you can read.
func validatePrepUsage(diags *diag.Diagnostics, mode string, minutes int64) {
	switch {
	case mode == "knead" && minutes > 30:
		diags.AddError("Over-kneading",
			fmt.Sprintf("Kneading for %d minutes turned the dough into glue. Real kitchens fail too: keep kneading under 30 minutes, the gluten gives up past that.", minutes))
	case mode == "rest" && minutes < 15:
		diags.AddError("Useless rest",
			fmt.Sprintf("Resting for %d minutes does nothing to the dough. Give it at least 15 minutes to relax.", minutes))
	case mode == "whip" && minutes > 25:
		diags.AddError("Arm gave out",
			fmt.Sprintf("Whipping by hand for %d minutes is superhero work. Keep whips under 25 minutes or delegate to the mixer.", minutes))
	}
}

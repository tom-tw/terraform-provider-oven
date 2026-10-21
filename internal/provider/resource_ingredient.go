package provider

import (
	"context"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/tom-tw/terraform-provider-oven/internal/kitchen"
)

var (
	_ resource.Resource                = &ingredientResource{}
	_ resource.ResourceWithImportState = &ingredientResource{}
)

// NewIngredientResource returns the oven_ingredient resource.
func NewIngredientResource() resource.Resource {
	return &ingredientResource{}
}

type ingredientResource struct {
	kitchen *Kitchen
}

type ingredientResourceModel struct {
	ID            types.String  `tfsdk:"id"`
	Name          types.String  `tfsdk:"name"`
	Amount        types.Float64 `tfsdk:"amount"`
	Unit          types.String  `tfsdk:"unit"`
	ExpiresInDays types.Int64   `tfsdk:"expires_in_days"`
	StoredAt      types.String  `tfsdk:"stored_at"`
	Fresh         types.Bool    `tfsdk:"fresh"`
}

func (r *ingredientResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingredient"
}

func (r *ingredientResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Stocks an ingredient in the pantry of your imaginary kitchen. " +
			"Deleting it empties the shelf. Renaming the ingredient buys a different one, " +
			"so changing `name` forces a replacement, like real Terraform resources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Pantry barcode, in the form `ingredient:<name>`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Pantry label, e.g. `flour`, `milk`, `chocolate-chips`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"amount": schema.Float64Attribute{
				Required:    true,
				Description: "How much to stock, in the unit below.",
				Validators:  []validator.Float64{float64validator.Between(0.01, 100000)},
			},
			"unit": schema.StringAttribute{
				Required:    true,
				Description: "Pantry unit for the amount.",
				Validators: []validator.String{
					stringvalidator.OneOf("g", "kg", "ml", "l", "tsp", "tbsp", "cup", "count", "pinch"),
				},
			},
			"expires_in_days": schema.Int64Attribute{
				Optional:    true,
				Description: "How many days before the imaginary expiration date.",
				Validators:  []validator.Int64{int64validator.Between(1, 3650)},
			},
			"stored_at": schema.StringAttribute{
				Computed:      true,
				Description:   "Timestamp when the ingredient was shelved (UTC).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"fresh": schema.BoolAttribute{
				Computed:      true,
				Description:   "The pantry's deterministic judgement about freshness. Mostly true.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *ingredientResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.kitchen = kitchenFromProviderData(req.ProviderData, "resource.Configure", &resp.Diagnostics)
}

func (r *ingredientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ingredientResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.kitchen == nil {
		resp.Diagnostics.AddError("Kitchen not configured", "The oven provider has no kitchen data; this is a provider bug.")
		return
	}

	name := plan.Name.ValueString()
	if !r.kitchen.chaosRoll("oven_ingredient", name, &resp.Diagnostics) {
		return
	}

	plan.ID = types.StringValue("ingredient:" + name)
	plan.StoredAt = types.StringValue(time.Now().UTC().Format(time.RFC3339))
	plan.Fresh = types.BoolValue(kitchen.HashRoll("freshness|"+r.kitchen.KitchenName+"|"+name) > 0.15)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ingredientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// The pantry is imaginary: nothing to refresh, state stays as is.
}

func (r *ingredientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ingredientResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.kitchen.chaosRoll("oven_ingredient", plan.Name.ValueString(), &resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ingredientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Emptying a shelf always works; the chaos engine only strikes
	// while stocking.
}

func (r *ingredientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	var id string
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), strings.TrimPrefix(id, "ingredient:"))...)
}

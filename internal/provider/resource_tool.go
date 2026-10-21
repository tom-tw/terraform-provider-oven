package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &toolResource{}
	_ resource.ResourceWithImportState = &toolResource{}
)

const toolKindsDescription = "Whisk, mixer, oven, grill, knife, rollingpin, spatula"

// ToolKinds lists every implement the kitchen can stock, in the order
// documentation lists them.
var ToolKinds = []string{"whisk", "mixer", "oven", "grill", "knife", "rollingpin", "spatula"}

// NewToolResource returns the oven_tool resource.
func NewToolResource() resource.Resource {
	return &toolResource{}
}

type toolResource struct {
	kitchen *Kitchen
}

type toolResourceModel struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Kind  types.String `tfsdk:"kind"`
	Brand types.String `tfsdk:"brand"`
	Clean types.Bool   `tfsdk:"clean"`
}

func (r *toolResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tool"
}

func (r *toolResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Buys a kitchen tool. Changing `kind` means buying a different implement, " +
			"so it forces replacement (watch the plan: destroy + create). Renaming the tool is also a new tool.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Drawer sticker, in the form `tool:<name>`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Drawer label, e.g. `grandma-whisk`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"kind": schema.StringAttribute{
				Required:      true,
				Description:   toolKindsDescription + ". Changing it buys a new tool (forces replacement).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.OneOf(ToolKinds...)},
			},
			"brand": schema.StringAttribute{
				Optional:    true,
				Description: "Imaginary brand name, freely updatable.",
			},
			"clean": schema.BoolAttribute{
				Computed:      true,
				Description:   "Whether the tool drawer received it clean. Mostly, but not always.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *toolResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.kitchen = kitchenFromProviderData(req.ProviderData, "resource.Configure", &resp.Diagnostics)
}

func (r *toolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan toolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.kitchen == nil {
		resp.Diagnostics.AddError("Kitchen not configured", "The oven provider has no kitchen data; this is a provider bug.")
		return
	}

	name := plan.Name.ValueString()
	if !r.kitchen.chaosRoll("oven_tool", name, &resp.Diagnostics) {
		return
	}

	plan.ID = types.StringValue("tool:" + name)
	plan.Clean = types.BoolValue(!toolArrivedDirty(r.kitchen.KitchenName, name))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// toolArrivedDirty is deterministic per kitchen and tool name: some
// deliveries are pre-crumbled.
func toolArrivedDirty(kitchenName, name string) bool {
	return hashThreshold("cleanliness|"+kitchenName+"|"+name, 0.9)
}

func (r *toolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Tools in the drawer never wander; nothing to refresh.
}

func (r *toolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan toolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.kitchen.chaosRoll("oven_tool", plan.Name.ValueString(), &resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *toolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Donating a tool to the neighbor always succeeds.
}

func (r *toolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	var id string
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), strings.TrimPrefix(id, "tool:"))...)
}

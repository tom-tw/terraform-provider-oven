package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	ephemeralschema "github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/tom-tw/terraform-provider-oven/internal/kitchen"
)

var _ ephemeral.EphemeralResource = &tastingEphemeral{}

// NewTastingEphemeral returns the oven_tasting_spoon ephemeral resource.
//
// OpenTofu ephemeral resources (1.10+) exist only for the duration of
// one apply: values are available to other ephemeral configuration but
// are never persisted to state. Perfect for tasting spoons: after the
// apply, the spoon has been licked clean and put away.
func NewTastingEphemeral() ephemeral.EphemeralResource {
	return &tastingEphemeral{}
}

type tastingEphemeral struct {
	kitchen *Kitchen
}

type tastingModel struct {
	Name    types.String `tfsdk:"name"`
	Verdict types.String `tfsdk:"verdict"`
}

func (r *tastingEphemeral) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tasting_spoon"
}

func (r *tastingEphemeral) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = ephemeralschema.Schema{
		MarkdownDescription: "A tasting spoon. Exists only during the apply and leaves no state behind: " +
			"the perfect demonstration of ephemeral resources. The verdict is a joke for hummus-grade palates.",
		Attributes: map[string]ephemeralschema.Attribute{
			"name": ephemeralschema.StringAttribute{
				Required:    true,
				Description: "What the cook tasted, e.g. `tomato-sauce`.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"verdict": ephemeralschema.StringAttribute{
				Computed:    true,
				Description: "The cook's snap judgement.",
			},
		},
	}
}

func (r *tastingEphemeral) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	r.kitchen = kitchenFromProviderData(req.ProviderData, "ephemeral.Configure", &resp.Diagnostics)
}

func (r *tastingEphemeral) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var plan tastingModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.kitchen == nil {
		resp.Diagnostics.AddError("Kitchen not configured", "The oven provider has no kitchen data; this is a provider bug.")
		return
	}

	name := plan.Name.ValueString()

	if !r.kitchen.chaosRoll("oven_tasting_spoon", name, &resp.Diagnostics) {
		return
	}

	plan.Verdict = types.StringValue(tasteVerdict(r.kitchen.KitchenName, name))
	resp.Diagnostics.Append(resp.Result.Set(ctx, &plan)...)
}

// tasteVerdict is deterministic per kitchen and tasted thing.
func tasteVerdict(kitchenName, name string) string {
	roll := kitchen.HashRoll("taste|" + kitchenName + "|" + name)
	switch {
	case roll > 0.6:
		return "needs more sugar"
	case roll > 0.3:
		return "almost perfect"
	default:
		return "ready to serve"
	}
}

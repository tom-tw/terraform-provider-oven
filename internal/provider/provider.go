// Package provider wires the imaginary kitchen to OpenTofu. Nothing
// here touches real infrastructure: resources pretend to prepare
// food, and the chaos engine in internal/kitchen makes them fail
// from time to time so practitioners can practise real workflows.
package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/tom-tw/terraform-provider-oven/internal/kitchen"
)

// New returns a provider factory for the oven provider.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ovenProvider{version: version}
	}
}

var (
	_ provider.Provider                       = (*ovenProvider)(nil)
	_ provider.ProviderWithEphemeralResources = (*ovenProvider)(nil)
	_ provider.ProviderWithFunctions          = (*ovenProvider)(nil)
)

type ovenProvider struct {
	version string
	kitchen *Kitchen
}

func (p *ovenProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "oven"
	resp.Version = p.version
}

func (p *ovenProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The oven provider simulates a kitchen so you can practice OpenTofu " +
			"without touching any cloud. Resources represent ingredients, tools, preparations " +
			"and bakes. A chaos engine makes operations fail from time to time (the gas ran " +
			"out, the cat sat on the dough...) so you can practice reading errors and retrying. " +
			"Invalid usage always fails, which makes it perfect for learning.",
		Attributes: map[string]schema.Attribute{
			"kitchen_name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the imaginary kitchen. Namespaces chaos bookkeeping and failure messages.",
			},
			"oven_model": schema.StringAttribute{
				Optional:    true,
				Description: "Model of your imaginary oven, purely decorative but mentioned in every error.",
			},
			"max_temperature": schema.Float64Attribute{
				Optional:    true,
				Description: "Highest temperature the oven can reach, in Celsius. Bakes above it fail deterministically.",
				Validators:  []validator.Float64{float64validator.Between(50, 400)},
			},
			"chaos_level": schema.Float64Attribute{
				Optional: true,
				Description: "Probability (0-1) that a creating or updating operation randomly fails. " +
					"Attempt 3 always succeeds: retry an apply and the kitchen forgives you.",
				Validators: []validator.Float64{float64validator.Between(0, 1)},
			},
		},
	}
}

func (p *ovenProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ovenProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	kitchenName := "home-kitchen"
	if !config.KitchenName.IsNull() {
		kitchenName = config.KitchenName.ValueString()
	}
	ovenModel := "OVEN-5000"
	if !config.OvenModel.IsNull() {
		ovenModel = config.OvenModel.ValueString()
	}
	maxTemp := 250.0
	if !config.MaxTemperature.IsNull() {
		maxTemp = config.MaxTemperature.ValueFloat64()
	}
	chaosLevel := 0.12
	if !config.ChaosLevel.IsNull() {
		chaosLevel = config.ChaosLevel.ValueFloat64()
	}

	p.kitchen = &Kitchen{
		KitchenName:    kitchenName,
		OvenModel:      ovenModel,
		MaxTemperature: maxTemp,
		Chaos: kitchen.NewChaosEngine(chaosLevel, kitchenName, kitchen.NewFileStore(
			filepath.Join(ovenStateDir(), "kitchen.json"),
		)),
	}

	// Framework v1.9+ requires each component type to receive the
	// configured data explicitly.
	resp.DataSourceData = p.kitchen
	resp.EphemeralResourceData = p.kitchen
	resp.ResourceData = p.kitchen
}

// ovenStateDir resolves where chaos bookkeeping lives. OVEN_STATE_DIR
// overrides the default ~/.local/share/oven, which keeps tests hermetic.
func ovenStateDir() string {
	if dir := os.Getenv("OVEN_STATE_DIR"); dir != "" {
		return dir
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "oven")
	}
	return filepath.Join(os.TempDir(), "oven-state")
}

func (p *ovenProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewIngredientResource,
		NewToolResource,
		NewPrepResource,
		NewMixResource,
		NewRecipeResource,
		NewBakeResource,
	}
}

func (p *ovenProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewRecipeBookDataSource,
		NewOvenDataSource,
	}
}

func (p *ovenProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		NewTastingEphemeral,
	}
}

func (p *ovenProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		NewConvertTempFunction,
		NewConvertUnitsFunction,
		NewBakeTimeFunction,
		NewMixRatioFunction,
	}
}

type ovenProviderModel struct {
	KitchenName    types.String  `tfsdk:"kitchen_name"`
	OvenModel      types.String  `tfsdk:"oven_model"`
	MaxTemperature types.Float64 `tfsdk:"max_temperature"`
	ChaosLevel     types.Float64 `tfsdk:"chaos_level"`
}

// Kitchen is the configured, shared kitchen passed to every resource,
// data source and ephemeral resource through ProviderData.
type Kitchen struct {
	KitchenName    string
	OvenModel      string
	MaxTemperature float64
	Chaos          *kitchen.ChaosEngine
}

func kitchenFromProviderData(data any, where string, diags *diag.Diagnostics) *Kitchen {
	if data == nil {
		return nil
	}
	k, ok := data.(*Kitchen)
	if !ok {
		diags.AddError(fmt.Sprintf("Unexpected %s data", where), fmt.Sprintf("Expected *provider.Kitchen, got %T", data))
		return nil
	}
	return k
}

// chaosRoll consults the chaos engine for a mutating operation. It
// returns false (and adds an error diagnostic) when the kitchen
// misbehaves, which aborts the operation without touching state.
func (k *Kitchen) chaosRoll(resourceType, resourceName string, diags *diag.Diagnostics) bool {
	if k.Chaos == nil {
		return true
	}
	err := k.Chaos.Roll(resourceType, resourceName)
	if err == nil {
		return true
	}
	diags.AddError(
		"A kitchen accident happened (chaos)",
		fmt.Sprintf("%v\nThe previous state is untouched. Re-run the apply: the kitchen usually cooperates on attempt 2 or 3.", err),
	)
	return false
}

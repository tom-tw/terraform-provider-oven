package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccIngredientLifecycle(t *testing.T) {
	t.Setenv("OVEN_STATE_DIR", t.TempDir())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: chaosFreeProviderConfig + `
resource "oven_ingredient" "flour" {
  name   = "flour"
  amount = 500
  unit   = "g"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("oven_ingredient.flour", "id", "ingredient:flour"),
					resource.TestCheckResourceAttr("oven_ingredient.flour", "amount", "500"),
					resource.TestCheckResourceAttr("oven_ingredient.flour", "unit", "g"),
					resource.TestCheckResourceAttrSet("oven_ingredient.flour", "stored_at"),
					resource.TestCheckResourceAttrSet("oven_ingredient.flour", "fresh"),
				),
			},
			{
				// Updating the amount keeps the same shelf.
				Config: chaosFreeProviderConfig + `
resource "oven_ingredient" "flour" {
  name   = "flour"
  amount = 750
  unit   = "g"
}
`,
				Check: resource.TestCheckResourceAttr("oven_ingredient.flour", "amount", "750"),
			},
			{
				// Renaming the ingredient forces replacement.
				Config: chaosFreeProviderConfig + `
resource "oven_ingredient" "flour" {
  name   = "whole-wheat-flour"
  amount = 750
  unit   = "g"
}
`,
				Check: resource.TestCheckResourceAttr("oven_ingredient.flour", "id", "ingredient:whole-wheat-flour"),
			},
			{
				ResourceName:            "oven_ingredient.flour",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"amount", "unit", "expires_in_days", "stored_at", "fresh"},
			},
		},
	})
}

func TestAccIngredientRejectsEmptyName(t *testing.T) {
	t.Setenv("OVEN_STATE_DIR", t.TempDir())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: chaosFreeProviderConfig + `
resource "oven_ingredient" "bad" {
  name   = ""
  amount = 10
  unit   = "g"
}
`,
				ExpectError: regexp.MustCompile(`must be at least`),
			},
		},
	})
}

func TestAccToolLifecycle(t *testing.T) {
	t.Setenv("OVEN_STATE_DIR", t.TempDir())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: chaosFreeProviderConfig + `
resource "oven_tool" "whisk" {
  name  = "grandma-whisk"
  kind  = "whisk"
  brand = "Sofia"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("oven_tool.whisk", "id", "tool:grandma-whisk"),
					resource.TestCheckResourceAttr("oven_tool.whisk", "kind", "whisk"),
					resource.TestCheckResourceAttrSet("oven_tool.whisk", "clean"),
				),
			},
			{
				// Brand is free to change without replacement.
				Config: chaosFreeProviderConfig + `
resource "oven_tool" "whisk" {
  name  = "grandma-whisk"
  kind  = "whisk"
  brand = "BakeryPro"
}
`,
				Check: resource.TestCheckResourceAttr("oven_tool.whisk", "brand", "BakeryPro"),
			},
			{
				// New kind is a new tool: replacement with a stable id.
				Config: chaosFreeProviderConfig + `
resource "oven_tool" "whisk" {
  name  = "grandma-whisk"
  kind  = "oven"
  brand = "BakeryPro"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("oven_tool.whisk", "kind", "oven"),
					resource.TestCheckResourceAttr("oven_tool.whisk", "id", "tool:grandma-whisk"),
				),
			},
			{
				ResourceName:            "oven_tool.whisk",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"brand", "kind", "clean"},
			},
		},
	})
}

func TestAccPrepLifecycleAndMisuse(t *testing.T) {
	t.Setenv("OVEN_STATE_DIR", t.TempDir())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: chaosFreeProviderConfig + `
resource "oven_prep" "knead" {
  name             = "knead-base"
  mode             = "knead"
  duration_minutes = 10
  ingredients      = ["flour", "water"]
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("oven_prep.knead", "id", "prep:knead-base"),
					resource.TestCheckResourceAttr("oven_prep.knead", "done", "true"),
					resource.TestCheckResourceAttr("oven_prep.knead", "ingredients.#", "2"),
					resource.TestCheckResourceAttrSet("oven_prep.knead", "technique"),
				),
			},
			{
				// Over-kneading fails deterministically: retrying
				// will not fix bad baking technique.
				Config: chaosFreeProviderConfig + `
resource "oven_prep" "knead" {
  name             = "knead-base"
  mode             = "knead"
  duration_minutes = 45
}
`,
				ExpectError: regexp.MustCompile("Over-kneading"),
			},
			{
				Config: chaosFreeProviderConfig + `
resource "oven_prep" "bad-rest" {
  name             = "lazy-rest"
  mode             = "rest"
  duration_minutes = 5
}
`,
				ExpectError: regexp.MustCompile("Useless rest"),
			},
		},
	})
}

func TestAccMixConsistency(t *testing.T) {
	t.Setenv("OVEN_STATE_DIR", t.TempDir())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: chaosFreeProviderConfig + `
resource "oven_mix" "pancakes" {
  name = "pancake-batter"
  ingredients = {
    flour = 250
    sugar = 30
    milk  = 300
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("oven_mix.pancakes", "id", "mix:pancake-batter"),
					resource.TestCheckResourceAttr("oven_mix.pancakes", "consistency", "batter"),
				),
			},
			{
				// More water turns batter into dough territory.
				Config: chaosFreeProviderConfig + `
resource "oven_mix" "pancakes" {
  name = "pancake-batter"
  ingredients = {
    flour = 500
    water = 100
  }
}
`,
				Check: resource.TestCheckResourceAttr("oven_mix.pancakes", "consistency", "dough"),
			},
		},
	})
}

func TestAccRecipeLifecycle(t *testing.T) {
	t.Setenv("OVEN_STATE_DIR", t.TempDir())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: chaosFreeProviderConfig + `
resource "oven_recipe" "cake" {
  name             = "birthday-cake"
  servings         = 8
  temperature      = 175
  duration_minutes = 40

  steps {
    order       = 1
    description = "Cream butter and sugar"
    mode        = "mix"
  }

  steps {
    order       = 2
    description = "Bake until a skewer is almost clean"
    mode        = "rest"
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("oven_recipe.cake", "id", "recipe:birthday-cake"),
					resource.TestCheckResourceAttr("oven_recipe.cake", "steps.#", "2"),
					resource.TestCheckResourceAttr("oven_recipe.cake", "steps.0.description", "Cream butter and sugar"),
					resource.TestCheckResourceAttr("oven_recipe.cake", "steps.1.mode", "rest"),
					resource.TestCheckResourceAttrSet("oven_recipe.cake", "outcome"),
				),
			},
			{
				// Steps are data, so they update in place.
				Config: chaosFreeProviderConfig + `
resource "oven_recipe" "cake" {
  name             = "birthday-cake"
  servings         = 8
  temperature      = 175
  duration_minutes = 40

  steps {
    order       = 1
    description = "Cream butter and sugar until pale"
    mode        = "mix"
  }
}
`,
				Check: resource.TestCheckResourceAttr("oven_recipe.cake", "steps.#", "1"),
			},
			{
				// The provider-level oven maximum is enforced
				// deterministically. The cake resource stays so
				// later import steps can find it in state.
				Config: testAccRequiredProviders + `
provider "oven" {
  chaos_level     = 0
  max_temperature = 250
}

resource "oven_recipe" "cake" {
  name             = "birthday-cake"
  servings         = 8
  temperature      = 175
  duration_minutes = 40

  steps {
    order       = 1
    description = "Cream butter and sugar until pale"
    mode        = "mix"
  }
}

resource "oven_recipe" "torch" {
  name        = "volcano-cake"
  servings    = 4
  temperature = 300
}
`,
				ExpectError: regexp.MustCompile("Oven too small"),
			},
			{
				// Import reconstructs state from the recipe:<name> id.
				Config: chaosFreeProviderConfig + `
resource "oven_recipe" "cake" {
  name             = "birthday-cake"
  servings         = 8
  temperature      = 175
  duration_minutes = 40

  steps {
    order       = 1
    description = "Cream butter and sugar until pale"
    mode        = "mix"
  }
}
`,
				ResourceName:            "oven_recipe.cake",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"servings", "temperature", "duration_minutes", "steps", "outcome"},
			},
		},
	})
}

func TestAccBakeLifecycle(t *testing.T) {
	t.Setenv("OVEN_STATE_DIR", t.TempDir())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: chaosFreeProviderConfig + `
resource "oven_bake" "cookies" {
  name             = "batch-01"
  temperature      = 180
  duration_minutes = 12
  mode             = "static"
  ingredients      = ["flour", "sugar"]
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("oven_bake.cookies", "id", "bake:batch-01"),
					resource.TestCheckResourceAttr("oven_bake.cookies", "doneness", "golden"),
					resource.TestCheckResourceAttr("oven_bake.cookies", "ingredients.#", "2"),
				),
			},
			{
				// Changing temperature recomputes doneness in place.
				Config: chaosFreeProviderConfig + `
resource "oven_bake" "cookies" {
  name             = "batch-01"
  temperature      = 200
  duration_minutes = 90
  mode             = "static"
  ingredients      = ["flour", "sugar"]
}
`,
				Check: resource.TestCheckResourceAttr("oven_bake.cookies", "doneness", "charcoal"),
			},
			{
				// An oven that cannot reach 300°C fails no matter what.
				Config: testAccRequiredProviders + `
provider "oven" {
  chaos_level     = 0
  max_temperature = 250
}

resource "oven_bake" "pizza" {
  name             = "inferno-pizza"
  temperature      = 300
  duration_minutes = 8
  mode             = "static"
}
`,
				ExpectError: regexp.MustCompile("Oven too small"),
			},
		},
	})
}

// TestAccBakeChaosFailureAndRetry exercises the full chaos cycle with
// chaos_level = 1: the first two attempts must fail with a themed
// kitchen accident, attempt 3 always succeeds, and the failure streak
// resets so future chaos can happen again.
func TestAccBakeChaosFailureAndRetry(t *testing.T) {
	t.Setenv("OVEN_STATE_DIR", t.TempDir())

	chaosConfig := testAccRequiredProviders + `
provider "oven" {
  chaos_level = 1
}

resource "oven_bake" "cake" {
  name             = "chaos-cake"
  temperature      = 180
  duration_minutes = 30
  mode             = "static"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      chaosConfig,
				ExpectError: regexp.MustCompile("A kitchen accident happened"),
			},
			{
				Config:      chaosConfig,
				ExpectError: regexp.MustCompile("A kitchen accident happened"),
			},
			{
				Config: chaosConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("oven_bake.cake", "name", "chaos-cake"),
					resource.TestCheckResourceAttr("oven_bake.cake", "doneness", "golden"),
				),
			},
		},
	})
}

func TestAccRecipeBookDataSource(t *testing.T) {
	t.Setenv("OVEN_STATE_DIR", t.TempDir())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: chaosFreeProviderConfig + `
data "oven_recipe_book" "pancakes" {
  recipe_name = "pancakes"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.oven_recipe_book.pancakes", "id", "pancakes"),
					resource.TestCheckResourceAttr("data.oven_recipe_book.pancakes", "cuisine", "international"),
					resource.TestCheckResourceAttr("data.oven_recipe_book.pancakes", "difficulty", "easy"),
					resource.TestCheckResourceAttr("data.oven_recipe_book.pancakes", "servings", "4"),
					resource.TestCheckResourceAttr("data.oven_recipe_book.pancakes", "bake_temp_c", "190"),
					resource.TestCheckResourceAttr("data.oven_recipe_book.pancakes", "bake_temp_f", "374"),
					resource.TestCheckResourceAttr("data.oven_recipe_book.pancakes", "steps.#", "4"),
					resource.TestCheckResourceAttr("data.oven_recipe_book.pancakes", "ingredients.#", "5"),
					resource.TestCheckTypeSetElemAttr("data.oven_recipe_book.pancakes", "tools.*", "whisk"),
					resource.TestCheckResourceAttr("data.oven_recipe_book.pancakes", "available_recipes.#", "7"),
				),
			},
			{
				// Unknown recipes fail with the full catalogue so
				// learners can self-correct.
				Config: chaosFreeProviderConfig + `
data "oven_recipe_book" "mystery" {
  recipe_name = "birthday-cake"
}
`,
				ExpectError: regexp.MustCompile("Recipe not found"),
			},
		},
	})
}

func TestAccOvenDataSource(t *testing.T) {
	t.Setenv("OVEN_STATE_DIR", t.TempDir())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRequiredProviders + `
provider "oven" {
  kitchen_name    = "learning-kitchen"
  oven_model      = "TEST-100"
  max_temperature = 275
  chaos_level     = 0
}

data "oven_oven" "this" {}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.oven_oven.this", "id", "learning-kitchen"),
					resource.TestCheckResourceAttr("data.oven_oven.this", "kitchen_name", "learning-kitchen"),
					resource.TestCheckResourceAttr("data.oven_oven.this", "oven_model", "TEST-100"),
					resource.TestCheckResourceAttr("data.oven_oven.this", "max_temperature", "275"),
					resource.TestCheckResourceAttr("data.oven_oven.this", "chaos_level", "0"),
					resource.TestCheckResourceAttr("data.oven_oven.this", "bake_modes.0", "convection"),
					resource.TestCheckResourceAttr("data.oven_oven.this", "tool_kinds.0", "grill"),
					resource.TestCheckResourceAttr("data.oven_oven.this", "prep_modes.0", "fold"),
				),
			},
		},
	})
}

func TestAccTastingEphemeral(t *testing.T) {
	t.Setenv("OVEN_STATE_DIR", t.TempDir())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: chaosFreeProviderConfig + `
ephemeral "oven_tasting_spoon" "sauce" {
  name = "tomato-sauce"
}

resource "oven_ingredient" "salt" {
  name   = "salt"
  amount = 500
  unit   = "g"
}
`,
				// Ephemeral values never persist: the real assertion
				// is that the apply succeeds while the spoon exists.
				Check: resource.TestCheckResourceAttr("oven_ingredient.salt", "id", "ingredient:salt"),
			},
		},
	})
}

func TestAccProviderFunctions(t *testing.T) {
	t.Setenv("OVEN_STATE_DIR", t.TempDir())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: chaosFreeProviderConfig + `
output "boiling" {
  value = provider::oven::convert_temp(100, "C", "F")
}

output "kilogram" {
  value = provider::oven::convert_units(1, "kg", "g")
}

output "pancakes-time" {
  value = provider::oven::bake_time("pancakes", 8)
}

output "batter-verdict" {
  value = provider::oven::mix_ratio(300, 250)
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckOutput("boiling", "212"),
					resource.TestCheckOutput("kilogram", "1000"),
					resource.TestCheckOutput("pancakes-time", "15"),
					resource.TestCheckOutput("batter-verdict", "batter"),
				),
			},
			{
				// Grams to cups remains impossible, even for us.
				Config: chaosFreeProviderConfig + `
output "oops" {
  value = provider::oven::convert_units(100, "g", "ml")
}
`,
				ExpectError: regexp.MustCompile("different dimensions"),
			},
			{
				Config: chaosFreeProviderConfig + `
output "mystery-book" {
  value = provider::oven::bake_time("birthday-cake", 4)
}
`,
				ExpectError: regexp.MustCompile("recipe not found"),
			},
		},
	})
}

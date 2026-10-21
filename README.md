# terraform-provider-oven

A **fake provider for OpenTofu** (and, by construction, terraform) that
turns infrastructure-as-code into **learn-to-bake**: resources stand for
pantry ingredients, kitchen tools, preparation steps, mixes, recipes and
bakes. It exists so you can practice **real HCL / OpenTofu workflows** —
plan → apply → state → failure → retry — without ever touching a cloud
or paying a bill.

> Nothing here does anything in the real world. The oven only exists in
> your `terraform.tfstate`.

## Why

Learning HCL on AWS means learning AWS at the same time. This provider
strips the distraction: every concept of the language and tooling shows
up here, mapped onto a domain everybody understands — cooking.

Companion learning repository: [`../tf-cook-recipes`](../tf-cook-recipes)
— ten progressive lessons with `tofu test` suites.

## Provider surface

| Type | Name | Teaches |
|---|---|---|
| resource | `oven_ingredient` | basics, `RequiresReplace`, import, computed attrs |
| resource | `oven_tool` | enums, replacement on `kind`, create-before-destroy |
| resource | `oven_prep` | modes, deterministic misuse failures |
| resource | `oven_mix` | map attributes, computed classifications |
| resource | `oven_recipe` | **nested blocks** → `dynamic` blocks |
| resource | `oven_bake` | mode enums, timeouts-free determinism, chaos target |
| data source | `oven_recipe_book` | complex structures, friendly lookup errors |
| data source | `oven_oven` | reading provider config back |
| ephemeral | `oven_tasting_spoon` | OpenTofu 1.10+ ephemeral resources |
| functions | `oven::convert_temp`, `oven::convert_units`, `oven::bake_time`, `oven::mix_ratio` | OpenTofu 1.7+ provider functions |

Provider configuration:

```hcl
provider "oven" {
  kitchen_name    = "my-kitchen"   # namespaces chaos bookkeeping
  oven_model      = "OVEN-5000"    # decorative, quoted in errors
  max_temperature = 250            # °C; bakes above fail deterministically
  chaos_level     = 0.12           # random failure probability (default)
}
```

## The chaos engine

OpenTofu users must learn to **read failures and retry**, so the
provider ships a misbehaving kitchen:

- Each create/update asks the chaos engine to roll dice. With the
  default `chaos_level = 0.12` (~12%) you get themed errors such as
  _"The cat sat on the rising dough"_, _"The gas ran out in the middle
  of the preparation"_, _"The oven timer batteries died with a final
  beep"_ — 20 different excuses.
- Retrying **usually works**: the outcome is deterministic per
  attempt number, and the attempt counter is persisted in
  `~/.local/share/oven/kitchen.json` (override the directory with
  the `OVEN_STATE_DIR` env var).
- **Attempt 3 always succeeds** — a failure streak cannot lock anyone
  out; a success resets the streak.
- **Invalid usage is deterministic, not random**: temperature above
  `max_temperature`, over-kneading (> 30 min), useless rests (< 15 min),
  unknown recipes, grams → cups conversions. Retrying never fixes bad
  technique — exactly like a real provider.

```text
=== a learning moment, live ===
$ tofu apply
Error: A kitchen accident happened (chaos)
  oven_bake.demo: The cat sat on the rising dough (kitchen attempt 1: another try may succeed)
$ tofu apply        # new attempt, new dice
Error: ...
$ tofu apply        # attempt 3: the kitchen forgives you
Apply complete! Resources: 1 added.
```

## Local development

Requirements: Go 1.21+ and OpenTofu (the repo assumes `tofu`, not
`terraform`).

```sh
make build            # → bin/terraform-provider-oven
make install          # → ~/bin/terraform-provider-oven
make dev-overrides    # prints the ~/.tofurc dev_overrides snippet
```

OpenTofu resolves the provider through its registry
(`registry.opentofu.org/tom-tw/oven`), so for local use register a
**dev override** in `~/.tofurc` pointing at the **directory** holding
the binary (OpenTofu treats the `path` as a package directory):

```hcl
provider_installation {
  dev_overrides {
    "registry.opentofu.org/tom-tw/oven" = "/home/you/bin"
  }
}
```

`tofu` will warn *"development overrides are in effect"* — that is the
intended setup for a local learning provider.

## Testing (TDD)

```sh
make test        # unit tests (kitchen domain logic + providers schemas)
make lint        # golangci-lint
make test-acc    # acceptance tests against real OpenTofu
```

The acceptance suite runs every resource/data source/function through
a real `tofu` binary (12 test cases including imports, replacements,
deterministic errors and the full chaos → retry → success cycle).
OpenTofu ignores in-process provider reattach during `init`, so the
suite runs through a generated `dev_overrides` CLI config; the
environment variables follow the OpenTofu acceptance-testing guide:

```sh
TF_ACC=1 TF_ACC_TERRAFORM_PATH=$(command -v tofu) \
TF_ACC_PROVIDER_HOST=registry.opentofu.org \
TF_ACC_PROVIDER_NAMESPACE=tom-tw \
TF_CLI_CONFIG_FILE=... go test ./internal/provider/...
```

(or just `make test-acc`, which wires all of that up.)

## Repository layout

```
main.go                    plugin server (protocol 6)
internal/kitchen/          pure domain logic, heavily unit-tested
  ├── chaos.go             chaos engine + persisted failure streaks
  ├── chaos_messages.go    the 20 excuses
  ├── conversions.go       temperature/unit conversions
  ├── recipebook.go        the known recipes dataset
  └── mixes.go             consistency + doneness heuristics
internal/provider/         terraform-plugin-framework wiring
  ├── provider.go          schema + configure + registry
  ├── resource_*.go        one file per resource
  ├── data_source_*.go     recipe book + oven introspection
  ├── ephemeral_tasting.go ephemeral tasting spoon
  ├── functions.go         the four provider functions
  └── *_test.go            acceptance tests (TF_ACC)
```

## Status

Academic/learning project. No registry publishing, no real kitchen —
your bread supply stays untouched.
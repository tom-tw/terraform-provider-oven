package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/tom-tw/terraform-provider-oven/internal/provider"
)

// testAccRequiredProviders pins the provider source so OpenTofu maps
// the `oven` local name onto the dev_overrides factory below.
const testAccRequiredProviders = `
terraform {
  required_providers {
    oven = {
      source = "tom-tw/oven"
    }
  }
}

`

// chaosFreeProviderConfig disables the chaos engine so lifecycle
// assertions are deterministic. Chaos behaviour gets its own test.
const chaosFreeProviderConfig = testAccRequiredProviders + `
provider "oven" {
  chaos_level = 0
}

`

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"oven": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

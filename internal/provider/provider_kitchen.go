package provider

import "github.com/tom-tw/terraform-provider-oven/internal/kitchen"

// hashThreshold reports whether the deterministic roll for a seed
// lands above the given threshold. Small helper for fun computed
// attributes that should look random but stay stable in state.
func hashThreshold(seed string, threshold float64) bool {
	return kitchen.HashRoll(seed) > threshold
}

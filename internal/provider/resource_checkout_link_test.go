// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccCheckoutLinkResource(t *testing.T) {
	rName := fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a checkout link with a product
			{
				Config: testAccCheckoutLinkConfig(rName, "true", "false"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_checkout_link.test",
						tfjsonpath.New("label"),
						knownvalue.StringExact(rName),
					),
					statecheck.ExpectKnownValue(
						"polar_checkout_link.test",
						tfjsonpath.New("allow_discount_codes"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"polar_checkout_link.test",
						tfjsonpath.New("require_billing_address"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"polar_checkout_link.test",
						tfjsonpath.New("payment_processor"),
						knownvalue.StringExact("stripe"),
					),
					statecheck.ExpectKnownOutputValue(
						"checkout_url",
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState
			{
				ResourceName:            "polar_checkout_link.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"client_secret"},
			},
			// Update label and allow_discount_codes
			{
				Config: testAccCheckoutLinkConfig(rName+"-upd", "false", "true"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_checkout_link.test",
						tfjsonpath.New("label"),
						knownvalue.StringExact(rName+"-upd"),
					),
					statecheck.ExpectKnownValue(
						"polar_checkout_link.test",
						tfjsonpath.New("allow_discount_codes"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"polar_checkout_link.test",
						tfjsonpath.New("require_billing_address"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}

// --- Config helpers ---

func testAccCheckoutLinkConfig(label, allowDiscountCodes, requireBillingAddress string) string {
	return fmt.Sprintf(`
resource "polar_product" "test" {
  name = %q

  prices = [{
    amount_type    = "free"
  }]
}

resource "polar_checkout_link" "test" {
  product_ids            = [polar_product.test.id]
  label                  = %q
  allow_discount_codes   = %s
  require_billing_address = %s
}

output "checkout_url" {
  value = polar_checkout_link.test.url
}
`, label, label, allowDiscountCodes, requireBillingAddress)
}

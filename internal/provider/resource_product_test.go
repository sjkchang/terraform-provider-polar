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

func TestAccProductResource_oneTimeFixed(t *testing.T) {
	rName := fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create one-time product with fixed price
			{
				Config: testAccProductOneTimeFixedConfig(rName, 999),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(rName),
					),
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("prices"),
						knownvalue.ListSizeExact(1),
					),
				},
			},
			// ImportState
			{
				ResourceName:      "polar_product.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name, price, and add description
			{
				Config: testAccProductWithDescriptionConfig(rName+"-updated", "A test product", 1999),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(rName+"-updated"),
					),
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("A test product"),
					),
				},
			},
			// Delete (archive) testing automatically occurs in TestCase
		},
	})
}

func TestAccProductResource_recurring(t *testing.T) {
	rName := fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create recurring monthly product
			{
				Config: testAccProductRecurringConfig(rName, "month", 499),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(rName),
					),
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("recurring_interval"),
						knownvalue.StringExact("month"),
					),
				},
			},
			// ImportState
			{
				ResourceName:      "polar_product.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name
			{
				Config: testAccProductRecurringConfig(rName+"-upd", "month", 999),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(rName+"-upd"),
					),
				},
			},
		},
	})
}

func TestAccProductResource_metadata(t *testing.T) {
	rName := fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProductWithMetadataConfig(rName, `{ env = "test" }`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("metadata").AtMapKey("env"),
						knownvalue.StringExact("test"),
					),
				},
			},
			// Update metadata
			{
				Config: testAccProductWithMetadataConfig(rName, `{ env = "staging" }`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("metadata").AtMapKey("env"),
						knownvalue.StringExact("staging"),
					),
				},
			},
		},
	})
}

func TestAccProductResource_customPrice(t *testing.T) {
	rName := fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProductCustomPriceConfig(rName, 500, 5000, 1000),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(rName),
					),
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("prices").AtSliceIndex(0).AtMapKey("amount_type"),
						knownvalue.StringExact("custom"),
					),
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("prices").AtSliceIndex(0).AtMapKey("minimum_amount"),
						knownvalue.Int64Exact(500),
					),
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("prices").AtSliceIndex(0).AtMapKey("maximum_amount"),
						knownvalue.Int64Exact(5000),
					),
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("prices").AtSliceIndex(0).AtMapKey("preset_amount"),
						knownvalue.Int64Exact(1000),
					),
				},
			},
			// ImportState
			{
				ResourceName:      "polar_product.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update custom price amounts
			{
				Config: testAccProductCustomPriceConfig(rName, 1000, 10000, 2500),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("prices").AtSliceIndex(0).AtMapKey("preset_amount"),
						knownvalue.Int64Exact(2500),
					),
				},
			},
		},
	})
}

func TestAccProductResource_meteredUnit(t *testing.T) {
	rName := fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProductMeteredUnitConfig(rName, rName+"-meter", "0.50", 10000),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(rName),
					),
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("prices").AtSliceIndex(0).AtMapKey("amount_type"),
						knownvalue.StringExact("metered_unit"),
					),
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("prices").AtSliceIndex(0).AtMapKey("unit_amount"),
						knownvalue.StringExact("0.50"),
					),
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("prices").AtSliceIndex(0).AtMapKey("cap_amount"),
						knownvalue.Int64Exact(10000),
					),
				},
			},
			// ImportState
			{
				ResourceName:            "polar_product.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"prices"},
			},
		},
	})
}

func TestAccProductResource_withBenefits(t *testing.T) {
	rName := fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create product with a meter_credit benefit attached
			{
				Config: testAccProductWithBenefitsConfig(rName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(rName),
					),
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("benefit_ids"),
						knownvalue.SetSizeExact(1),
					),
				},
			},
			// ImportState — benefit_ids is null after import (unmanaged until configured)
			{
				ResourceName:            "polar_product.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"benefit_ids"},
			},
		},
	})
}

// --- Config helpers ---

func testAccProductOneTimeFixedConfig(name string, priceAmount int64) string {
	return fmt.Sprintf(`
resource "polar_product" "test" {
  name = %q

  prices = [{
    amount_type    = "fixed"
    price_amount   = %d
  }]
}
`, name, priceAmount)
}

func testAccProductRecurringConfig(name, interval string, priceAmount int64) string {
	return fmt.Sprintf(`
resource "polar_product" "test" {
  name               = %q
  recurring_interval = %q

  prices = [{
    amount_type    = "fixed"
    price_amount   = %d
  }]
}
`, name, interval, priceAmount)
}

func testAccProductWithDescriptionConfig(name, description string, priceAmount int64) string {
	return fmt.Sprintf(`
resource "polar_product" "test" {
  name        = %q
  description = %q

  prices = [{
    amount_type    = "fixed"
    price_amount   = %d
  }]
}
`, name, description, priceAmount)
}

func testAccProductCustomPriceConfig(name string, minAmount, maxAmount, presetAmount int64) string {
	return fmt.Sprintf(`
resource "polar_product" "test" {
  name = %q

  prices = [{
    amount_type    = "custom"
    minimum_amount = %d
    maximum_amount = %d
    preset_amount  = %d
  }]
}
`, name, minAmount, maxAmount, presetAmount)
}

func testAccProductMeteredUnitConfig(name, meterName, unitAmount string, capAmount int64) string {
	return fmt.Sprintf(`
resource "polar_meter" "test" {
  name = %q

  filter = {
    conjunction = "and"
    clauses = [{
      property = "name"
      operator = "eq"
      value    = "test"
    }]
  }

  aggregation = {
    func = "count"
  }
}

resource "polar_product" "test" {
  name               = %q
  recurring_interval = "month"

  prices = [{
    amount_type = "metered_unit"
    meter_id    = polar_meter.test.id
    unit_amount = %q
    cap_amount  = %d
  }]
}
`, meterName, name, unitAmount, capAmount)
}

func testAccProductWithMetadataConfig(name, metadata string) string {
	return fmt.Sprintf(`
resource "polar_product" "test" {
  name = %q

  prices = [{
    amount_type    = "fixed"
    price_amount   = 500
  }]

  metadata = %s
}
`, name, metadata)
}

func testAccProductWithBenefitsConfig(name string) string {
	return fmt.Sprintf(`
resource "polar_meter" "test" {
  name = "%[1]s-meter"

  filter = {
    conjunction = "and"
    clauses = [{
      property = "name"
      operator = "eq"
      value    = "test"
    }]
  }

  aggregation = {
    func = "count"
  }
}

resource "polar_benefit" "test" {
  type        = "meter_credit"
  description = "Test meter credit"

  meter_credit_properties = {
    meter_id = polar_meter.test.id
    units    = 100
    rollover = false
  }
}

resource "polar_product" "test" {
  name = %[1]q

  prices = [{
    amount_type  = "fixed"
    price_amount = 500
  }]

  benefit_ids = [polar_benefit.test.id]
}
`, name)
}

func TestAccProductResource_seatBased(t *testing.T) {
	rName := fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create recurring monthly product with seat-based pricing (2 tiers)
			{
				Config: testAccProductSeatBasedConfig(rName, 1000, 800),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(rName),
					),
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("prices").AtSliceIndex(0).AtMapKey("amount_type"),
						knownvalue.StringExact("seat_based"),
					),
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("prices").AtSliceIndex(0).AtMapKey("seat_tiers"),
						knownvalue.ListSizeExact(2),
					),
				},
			},
			// ImportState
			{
				ResourceName:      "polar_product.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update tier pricing
			{
				Config: testAccProductSeatBasedConfig(rName, 1200, 900),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_product.test",
						tfjsonpath.New("prices").AtSliceIndex(0).AtMapKey("seat_tiers"),
						knownvalue.ListSizeExact(2),
					),
				},
			},
		},
	})
}

func testAccProductSeatBasedConfig(name string, tier1Price, tier2Price int64) string {
	return fmt.Sprintf(`
resource "polar_organization" "test" {
  feature_settings = {
    member_model_enabled       = true
    seat_based_pricing_enabled = true
  }
}

resource "polar_product" "test" {
  depends_on         = [polar_organization.test]
  name               = %q
  recurring_interval = "month"

  prices = [{
    amount_type = "seat_based"
    seat_tiers = [
      {
        min_seats      = 1
        max_seats      = 10
        price_per_seat = %d
      },
      {
        min_seats      = 11
        price_per_seat = %d
      },
    ]
  }]
}
`, name, tier1Price, tier2Price)
}

// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccDiscountResource_percentage(t *testing.T) {
	rName := fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a percentage discount with once duration
			{
				Config: testAccDiscountPercentageConfig(rName, 2550, "once"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_discount.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(rName),
					),
					statecheck.ExpectKnownValue(
						"polar_discount.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("percentage"),
					),
					statecheck.ExpectKnownValue(
						"polar_discount.test",
						tfjsonpath.New("duration"),
						knownvalue.StringExact("once"),
					),
					statecheck.ExpectKnownValue(
						"polar_discount.test",
						tfjsonpath.New("basis_points"),
						knownvalue.Int64Exact(2550),
					),
				},
			},
			// ImportState
			{
				ResourceName:      "polar_discount.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name and basis_points
			{
				Config: testAccDiscountPercentageConfig(rName+"-upd", 5000, "once"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_discount.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(rName+"-upd"),
					),
					statecheck.ExpectKnownValue(
						"polar_discount.test",
						tfjsonpath.New("basis_points"),
						knownvalue.Int64Exact(5000),
					),
				},
			},
		},
	})
}

func TestAccDiscountResource_fixed(t *testing.T) {
	rName := fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a fixed discount with forever duration
			{
				Config: testAccDiscountFixedConfig(rName, 1000, "forever"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_discount.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(rName),
					),
					statecheck.ExpectKnownValue(
						"polar_discount.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("fixed"),
					),
					statecheck.ExpectKnownValue(
						"polar_discount.test",
						tfjsonpath.New("duration"),
						knownvalue.StringExact("forever"),
					),
					statecheck.ExpectKnownValue(
						"polar_discount.test",
						tfjsonpath.New("amounts"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"usd": knownvalue.Int64Exact(1000),
						}),
					),
				},
			},
			// ImportState
			{
				ResourceName:      "polar_discount.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name and amount
			{
				Config: testAccDiscountFixedConfig(rName+"-upd", 2000, "forever"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_discount.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(rName+"-upd"),
					),
					statecheck.ExpectKnownValue(
						"polar_discount.test",
						tfjsonpath.New("amounts"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"usd": knownvalue.Int64Exact(2000),
						}),
					),
				},
			},
		},
	})
}

func TestDiscountResource_validation(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// type=fixed without amounts
			{
				Config: `
resource "polar_discount" "test" {
  name     = "Test"
  type     = "fixed"
  duration = "once"
}
`,
				ExpectError: regexp.MustCompile(`amounts.*is required when.*type.*is "fixed"`),
			},
			// type=percentage without basis_points
			{
				Config: `
resource "polar_discount" "test" {
  name     = "Test"
  type     = "percentage"
  duration = "once"
}
`,
				ExpectError: regexp.MustCompile(`basis_points.*is required when.*type.*is "percentage"`),
			},
			// duration=repeating without duration_in_months
			{
				Config: `
resource "polar_discount" "test" {
  name         = "Test"
  type         = "percentage"
  duration     = "repeating"
  basis_points = 1000
}
`,
				ExpectError: regexp.MustCompile(`duration_in_months.*is required when.*duration.*is "repeating"`),
			},
		},
	})
}

// --- Config helpers ---

func testAccDiscountPercentageConfig(name string, basisPoints int64, duration string) string {
	return fmt.Sprintf(`
resource "polar_discount" "test" {
  name         = %q
  type         = "percentage"
  duration     = %q
  basis_points = %d
}
`, name, duration, basisPoints)
}

func testAccDiscountFixedConfig(name string, amount int64, duration string) string {
	return fmt.Sprintf(`
resource "polar_discount" "test" {
  name     = %q
  type     = "fixed"
  duration = %q
  amounts  = {
    usd = %d
  }
}
`, name, duration, amount)
}

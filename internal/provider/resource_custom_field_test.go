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

func TestAccCustomFieldResource_text(t *testing.T) {
	rSlug := fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a text custom field
			{
				Config: testAccCustomFieldTextConfig(rSlug, "Full Name", "Your name"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_custom_field.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("text"),
					),
					statecheck.ExpectKnownValue(
						"polar_custom_field.test",
						tfjsonpath.New("slug"),
						knownvalue.StringExact(rSlug),
					),
					statecheck.ExpectKnownValue(
						"polar_custom_field.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Full Name"),
					),
					statecheck.ExpectKnownValue(
						"polar_custom_field.test",
						tfjsonpath.New("properties").AtMapKey("form_label"),
						knownvalue.StringExact("Your name"),
					),
				},
			},
			// ImportState
			{
				ResourceName:      "polar_custom_field.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name and form_label
			{
				Config: testAccCustomFieldTextConfig(rSlug, "Display Name", "Enter name"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_custom_field.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Display Name"),
					),
					statecheck.ExpectKnownValue(
						"polar_custom_field.test",
						tfjsonpath.New("properties").AtMapKey("form_label"),
						knownvalue.StringExact("Enter name"),
					),
				},
			},
		},
	})
}

func TestAccCustomFieldResource_select(t *testing.T) {
	rSlug := fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a select custom field
			{
				Config: testAccCustomFieldSelectConfig(rSlug, "Color"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_custom_field.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("select"),
					),
					statecheck.ExpectKnownValue(
						"polar_custom_field.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Color"),
					),
				},
			},
			// ImportState
			{
				ResourceName:      "polar_custom_field.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name
			{
				Config: testAccCustomFieldSelectConfig(rSlug, "Colour"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_custom_field.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Colour"),
					),
				},
			},
		},
	})
}

func TestAccCustomFieldResource_checkbox(t *testing.T) {
	rSlug := fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomFieldCheckboxConfig(rSlug, "Agree to Terms"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_custom_field.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("checkbox"),
					),
					statecheck.ExpectKnownValue(
						"polar_custom_field.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Agree to Terms"),
					),
				},
			},
			{
				ResourceName:      "polar_custom_field.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestCustomFieldResource_invalidProperties(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// textarea on non-text type
			{
				Config: `
resource "polar_custom_field" "test" {
  type = "number"
  slug = "test-field"
  name = "Test"

  properties = {
    textarea = true
  }
}
`,
				ExpectError: regexp.MustCompile(`textarea.*can only be set when type is "text"`),
			},
			// options on non-select type
			{
				Config: `
resource "polar_custom_field" "test" {
  type = "text"
  slug = "test-field"
  name = "Test"

  properties = {
    options = [{
      value = "a"
      label = "A"
    }]
  }
}
`,
				ExpectError: regexp.MustCompile(`options.*can only be set when type is "select"`),
			},
		},
	})
}

// --- Config helpers ---

func testAccCustomFieldTextConfig(slug, name, formLabel string) string {
	return fmt.Sprintf(`
resource "polar_custom_field" "test" {
  type = "text"
  slug = %q
  name = %q

  properties = {
    form_label = %q
  }
}
`, slug, name, formLabel)
}

func testAccCustomFieldSelectConfig(slug, name string) string {
	return fmt.Sprintf(`
resource "polar_custom_field" "test" {
  type = "select"
  slug = %q
  name = %q

  properties = {
    options = [
      {
        value = "red"
        label = "Red"
      },
      {
        value = "blue"
        label = "Blue"
      }
    ]
  }
}
`, slug, name)
}

func testAccCustomFieldCheckboxConfig(slug, name string) string {
	return fmt.Sprintf(`
resource "polar_custom_field" "test" {
  type = "checkbox"
  slug = %q
  name = %q

  properties = {
    form_label = "I agree"
  }
}
`, slug, name)
}

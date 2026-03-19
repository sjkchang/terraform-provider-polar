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

func TestAccOrganizationResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Adopt the existing org with minimal config
			{
				Config: testAccOrganizationMinimal(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_organization.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"polar_organization.test",
						tfjsonpath.New("slug"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState
			{
				ResourceName:      "polar_organization.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOrganizationResource_profileFields(t *testing.T) {
	rName := fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Set profile fields
			{
				Config: testAccOrganizationProfile(rName, "https://example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_organization.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(rName),
					),
					statecheck.ExpectKnownValue(
						"polar_organization.test",
						tfjsonpath.New("website"),
						knownvalue.StringExact("https://example.com"),
					),
				},
			},
			// Update profile fields
			{
				Config: testAccOrganizationProfile(rName+"-upd", "https://updated.example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_organization.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(rName+"-upd"),
					),
					statecheck.ExpectKnownValue(
						"polar_organization.test",
						tfjsonpath.New("website"),
						knownvalue.StringExact("https://updated.example.com"),
					),
				},
			},
		},
	})
}

func TestAccOrganizationResource_featureSettings(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationFeatureSettings(false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_organization.test",
						tfjsonpath.New("feature_settings").AtMapKey("issue_funding_enabled"),
						knownvalue.Bool(false),
					),
				},
			},
			{
				Config: testAccOrganizationFeatureSettings(true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_organization.test",
						tfjsonpath.New("feature_settings").AtMapKey("issue_funding_enabled"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}

func TestAccOrganizationResource_emailAndNotificationSettings(t *testing.T) {
	// Skip: Polar API now requires subscription_renewal_reminder and
	// subscription_trial_conversion_reminder fields in customer_email_settings,
	// but the polar-go SDK v0.15.0 doesn't expose them yet.
	t.Skip("blocked on polar-go SDK adding subscription_renewal_reminder and subscription_trial_conversion_reminder fields")
}

func TestAccOrganizationResource_subscriptionSettings(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationSubscriptionSettings("prorate", 7),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_organization.test",
						tfjsonpath.New("subscription_settings").AtMapKey("proration_behavior"),
						knownvalue.StringExact("prorate"),
					),
					statecheck.ExpectKnownValue(
						"polar_organization.test",
						tfjsonpath.New("subscription_settings").AtMapKey("benefit_revocation_grace_period"),
						knownvalue.Int64Exact(7),
					),
				},
			},
			{
				Config: testAccOrganizationSubscriptionSettings("invoice", 14),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"polar_organization.test",
						tfjsonpath.New("subscription_settings").AtMapKey("proration_behavior"),
						knownvalue.StringExact("invoice"),
					),
					statecheck.ExpectKnownValue(
						"polar_organization.test",
						tfjsonpath.New("subscription_settings").AtMapKey("benefit_revocation_grace_period"),
						knownvalue.Int64Exact(14),
					),
				},
			},
		},
	})
}

// --- Config helpers ---

func testAccOrganizationMinimal() string {
	return `
resource "polar_organization" "test" {
}
`
}

func testAccOrganizationProfile(name, website string) string {
	return fmt.Sprintf(`
resource "polar_organization" "test" {
  name    = %q
  website = %q
}
`, name, website)
}

func testAccOrganizationFeatureSettings(issueFunding bool) string {
	return fmt.Sprintf(`
resource "polar_organization" "test" {
  feature_settings = {
    issue_funding_enabled       = %t
    seat_based_pricing_enabled  = true
    member_model_enabled        = true
    revops_enabled              = false
    wallets_enabled             = false
  }
}
`, issueFunding)
}

func testAccOrganizationSubscriptionSettings(prorationBehavior string, gracePeriod int) string {
	return fmt.Sprintf(`
resource "polar_organization" "test" {
  subscription_settings = {
    allow_multiple_subscriptions    = true
    allow_customer_updates          = true
    proration_behavior              = %q
    benefit_revocation_grace_period = %d
    prevent_trial_abuse             = false
  }
}
`, prorationBehavior, gracePeriod)
}

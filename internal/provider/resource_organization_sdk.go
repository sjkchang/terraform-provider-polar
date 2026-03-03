// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/polarsource/polar-go/models/components"
)

// --- Discover the single organization scoped to the access token ---
// Polar access tokens are org-scoped, so listing orgs should return exactly one.

func discoverOrganization(ctx context.Context, client *PolarProviderData) (*components.Organization, error) {
	resp, err := client.Client.Organizations.List(ctx, nil, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("listing organizations: %w", err)
	}
	if resp.ListResourceOrganization == nil || len(resp.ListResourceOrganization.Items) == 0 {
		return nil, fmt.Errorf("no organizations found for the configured access token")
	}
	if len(resp.ListResourceOrganization.Items) > 1 {
		return nil, fmt.Errorf("expected exactly 1 organization, found %d", len(resp.ListResourceOrganization.Items))
	}
	org := resp.ListResourceOrganization.Items[0]
	return &org, nil
}

// --- Build SDK OrganizationUpdate from Terraform model ---
// Only sets fields the user included in their config (non-null).

func buildOrganizationUpdate(ctx context.Context, data *OrganizationResourceModel) (*components.OrganizationUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics
	update := components.OrganizationUpdate{}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		name := data.Name.ValueString()
		update.Name = &name
	}
	if !data.AvatarURL.IsNull() && !data.AvatarURL.IsUnknown() {
		u := data.AvatarURL.ValueString()
		update.AvatarURL = &u
	}
	if !data.Email.IsNull() && !data.Email.IsUnknown() {
		e := data.Email.ValueString()
		update.Email = &e
	}
	if !data.Website.IsNull() && !data.Website.IsUnknown() {
		w := data.Website.ValueString()
		update.Website = &w
	}

	// Socials
	if !data.Socials.IsNull() && !data.Socials.IsUnknown() {
		var socials []SocialModel
		d := data.Socials.ElementsAs(ctx, &socials, false)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		sdkSocials := make([]components.OrganizationSocialLink, len(socials))
		for i, s := range socials {
			sdkSocials[i] = components.OrganizationSocialLink{
				Platform: components.OrganizationSocialPlatforms(s.Platform.ValueString()),
				URL:      s.URL.ValueString(),
			}
		}
		update.Socials = sdkSocials
	}

	// Feature settings
	if data.FeatureSettings != nil {
		fs := &components.OrganizationFeatureSettings{}
		if !data.FeatureSettings.IssueFundingEnabled.IsNull() && !data.FeatureSettings.IssueFundingEnabled.IsUnknown() {
			b := data.FeatureSettings.IssueFundingEnabled.ValueBool()
			fs.IssueFundingEnabled = &b
		}
		if !data.FeatureSettings.SeatBasedPricingEnabled.IsNull() && !data.FeatureSettings.SeatBasedPricingEnabled.IsUnknown() {
			b := data.FeatureSettings.SeatBasedPricingEnabled.ValueBool()
			fs.SeatBasedPricingEnabled = &b
		}
		if !data.FeatureSettings.RevopsEnabled.IsNull() && !data.FeatureSettings.RevopsEnabled.IsUnknown() {
			b := data.FeatureSettings.RevopsEnabled.ValueBool()
			fs.RevopsEnabled = &b
		}
		if !data.FeatureSettings.WalletsEnabled.IsNull() && !data.FeatureSettings.WalletsEnabled.IsUnknown() {
			b := data.FeatureSettings.WalletsEnabled.ValueBool()
			fs.WalletsEnabled = &b
		}
		update.FeatureSettings = fs
	}

	// Subscription settings
	if data.SubscriptionSettings != nil {
		update.SubscriptionSettings = &components.OrganizationSubscriptionSettings{
			AllowMultipleSubscriptions:   data.SubscriptionSettings.AllowMultipleSubscriptions.ValueBool(),
			AllowCustomerUpdates:         data.SubscriptionSettings.AllowCustomerUpdates.ValueBool(),
			ProrationBehavior:            components.SubscriptionProrationBehavior(data.SubscriptionSettings.ProrationBehavior.ValueString()),
			BenefitRevocationGracePeriod: data.SubscriptionSettings.BenefitRevocationGracePeriod.ValueInt64(),
			PreventTrialAbuse:            data.SubscriptionSettings.PreventTrialAbuse.ValueBool(),
		}
	}

	// Notification settings
	if data.NotificationSettings != nil {
		ns := &components.OrganizationNotificationSettings{
			NewOrder:        data.NotificationSettings.NewOrder.ValueBool(),
			NewSubscription: data.NotificationSettings.NewSubscription.ValueBool(),
		}
		update.NotificationSettings = ns
	}

	// Customer email settings
	if data.CustomerEmailSettings != nil {
		update.CustomerEmailSettings = &components.OrganizationCustomerEmailSettings{
			OrderConfirmation:            data.CustomerEmailSettings.OrderConfirmation.ValueBool(),
			SubscriptionCancellation:     data.CustomerEmailSettings.SubscriptionCancellation.ValueBool(),
			SubscriptionConfirmation:     data.CustomerEmailSettings.SubscriptionConfirmation.ValueBool(),
			SubscriptionCycled:           data.CustomerEmailSettings.SubscriptionCycled.ValueBool(),
			SubscriptionCycledAfterTrial: data.CustomerEmailSettings.SubscriptionCycledAfterTrial.ValueBool(),
			SubscriptionPastDue:          data.CustomerEmailSettings.SubscriptionPastDue.ValueBool(),
			SubscriptionRevoked:          data.CustomerEmailSettings.SubscriptionRevoked.ValueBool(),
			SubscriptionUncanceled:       data.CustomerEmailSettings.SubscriptionUncanceled.ValueBool(),
			SubscriptionUpdated:          data.CustomerEmailSettings.SubscriptionUpdated.ValueBool(),
		}
	}

	return &update, diags
}

// --- Map SDK Organization response to Terraform state ---
// Only populates fields the user opted into (non-null in state).
// This lets users manage a subset of settings without TF fighting over the rest.

func mapOrganizationResponseToState(ctx context.Context, org *components.Organization, data *OrganizationResourceModel, diags *diag.Diagnostics) {
	data.ID = types.StringValue(org.ID)
	data.Slug = types.StringValue(org.Slug)

	// Profile fields: only set if user included them in config.
	if !data.Name.IsNull() {
		data.Name = types.StringValue(org.Name)
	}
	if !data.AvatarURL.IsNull() {
		data.AvatarURL = optionalStringValue(org.AvatarURL)
	}
	if !data.Email.IsNull() {
		data.Email = optionalStringValue(org.Email)
	}
	if !data.Website.IsNull() {
		data.Website = optionalStringValue(org.Website)
	}

	// Socials: only set if user configured them
	if !data.Socials.IsNull() {
		socialModels := make([]SocialModel, len(org.Socials))
		for i, s := range org.Socials {
			socialModels[i] = SocialModel{
				Platform: types.StringValue(string(s.Platform)),
				URL:      types.StringValue(s.URL),
			}
		}
		socialList, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: socialModelAttrTypes()}, socialModels)
		diags.Append(d...)
		data.Socials = socialList
	}

	// Settings blocks: only populate if already in state (user configured them)
	if data.FeatureSettings != nil && org.FeatureSettings != nil {
		data.FeatureSettings = &FeatureSettingsModel{
			IssueFundingEnabled:     types.BoolValue(derefBool(org.FeatureSettings.IssueFundingEnabled)),
			SeatBasedPricingEnabled: types.BoolValue(derefBool(org.FeatureSettings.SeatBasedPricingEnabled)),
			RevopsEnabled:           types.BoolValue(derefBool(org.FeatureSettings.RevopsEnabled)),
			WalletsEnabled:          types.BoolValue(derefBool(org.FeatureSettings.WalletsEnabled)),
		}
	}

	if data.SubscriptionSettings != nil {
		ss := org.SubscriptionSettings
		data.SubscriptionSettings = &SubscriptionSettingsModel{
			AllowMultipleSubscriptions:   types.BoolValue(ss.AllowMultipleSubscriptions),
			AllowCustomerUpdates:         types.BoolValue(ss.AllowCustomerUpdates),
			ProrationBehavior:            types.StringValue(string(ss.ProrationBehavior)),
			BenefitRevocationGracePeriod: types.Int64Value(ss.BenefitRevocationGracePeriod),
			PreventTrialAbuse:            types.BoolValue(ss.PreventTrialAbuse),
		}
	}

	if data.NotificationSettings != nil {
		data.NotificationSettings = &NotificationSettingsModel{
			NewOrder:        types.BoolValue(org.NotificationSettings.NewOrder),
			NewSubscription: types.BoolValue(org.NotificationSettings.NewSubscription),
		}
	}

	if data.CustomerEmailSettings != nil {
		ces := org.CustomerEmailSettings
		data.CustomerEmailSettings = &CustomerEmailSettingsModel{
			OrderConfirmation:            types.BoolValue(ces.OrderConfirmation),
			SubscriptionCancellation:     types.BoolValue(ces.SubscriptionCancellation),
			SubscriptionConfirmation:     types.BoolValue(ces.SubscriptionConfirmation),
			SubscriptionCycled:           types.BoolValue(ces.SubscriptionCycled),
			SubscriptionCycledAfterTrial: types.BoolValue(ces.SubscriptionCycledAfterTrial),
			SubscriptionPastDue:          types.BoolValue(ces.SubscriptionPastDue),
			SubscriptionRevoked:          types.BoolValue(ces.SubscriptionRevoked),
			SubscriptionUncanceled:       types.BoolValue(ces.SubscriptionUncanceled),
			SubscriptionUpdated:          types.BoolValue(ces.SubscriptionUpdated),
		}
	}
}

// --- Helpers ---

func socialModelAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"platform": types.StringType,
		"url":      types.StringType,
	}
}

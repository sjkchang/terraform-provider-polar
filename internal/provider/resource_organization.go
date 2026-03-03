// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/polarsource/polar-go/models/components"
)

// Compile-time interface conformance checks.
var _ resource.Resource = &OrganizationResource{}
var _ resource.ResourceWithImportState = &OrganizationResource{}

func NewOrganizationResource() resource.Resource {
	return &OrganizationResource{}
}

// OrganizationResource uses adopt-existing lifecycle: Create discovers the org
// scoped to the access token and updates it. Delete removes it from state.
// The provider is scoped to a single org because access tokens are org-scoped;
// users managing multiple orgs should use separate provider instances.
type OrganizationResource struct {
	provider *PolarProviderData
}

// --- Terraform model types ---

type OrganizationResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Slug      types.String `tfsdk:"slug"`
	AvatarURL types.String `tfsdk:"avatar_url"`
	Email     types.String `tfsdk:"email"`
	Website   types.String `tfsdk:"website"`
	Socials   types.List   `tfsdk:"socials"`

	FeatureSettings       *FeatureSettingsModel       `tfsdk:"feature_settings"`
	SubscriptionSettings  *SubscriptionSettingsModel  `tfsdk:"subscription_settings"`
	NotificationSettings  *NotificationSettingsModel  `tfsdk:"notification_settings"`
	CustomerEmailSettings *CustomerEmailSettingsModel `tfsdk:"customer_email_settings"`
}

type SocialModel struct {
	Platform types.String `tfsdk:"platform"`
	URL      types.String `tfsdk:"url"`
}

type FeatureSettingsModel struct {
	IssueFundingEnabled     types.Bool `tfsdk:"issue_funding_enabled"`
	SeatBasedPricingEnabled types.Bool `tfsdk:"seat_based_pricing_enabled"`
	RevopsEnabled           types.Bool `tfsdk:"revops_enabled"`
	WalletsEnabled          types.Bool `tfsdk:"wallets_enabled"`
}

type SubscriptionSettingsModel struct {
	AllowMultipleSubscriptions   types.Bool   `tfsdk:"allow_multiple_subscriptions"`
	AllowCustomerUpdates         types.Bool   `tfsdk:"allow_customer_updates"`
	ProrationBehavior            types.String `tfsdk:"proration_behavior"`
	BenefitRevocationGracePeriod types.Int64  `tfsdk:"benefit_revocation_grace_period"`
	PreventTrialAbuse            types.Bool   `tfsdk:"prevent_trial_abuse"`
}

type NotificationSettingsModel struct {
	NewOrder        types.Bool `tfsdk:"new_order"`
	NewSubscription types.Bool `tfsdk:"new_subscription"`
}

type CustomerEmailSettingsModel struct {
	OrderConfirmation            types.Bool `tfsdk:"order_confirmation"`
	SubscriptionCancellation     types.Bool `tfsdk:"subscription_cancellation"`
	SubscriptionConfirmation     types.Bool `tfsdk:"subscription_confirmation"`
	SubscriptionCycled           types.Bool `tfsdk:"subscription_cycled"`
	SubscriptionCycledAfterTrial types.Bool `tfsdk:"subscription_cycled_after_trial"`
	SubscriptionPastDue          types.Bool `tfsdk:"subscription_past_due"`
	SubscriptionRevoked          types.Bool `tfsdk:"subscription_revoked"`
	SubscriptionUncanceled       types.Bool `tfsdk:"subscription_uncanceled"`
	SubscriptionUpdated          types.Bool `tfsdk:"subscription_updated"`
}

// --- Resource interface ---

func (r *OrganizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *OrganizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Polar organization's settings. The organization must already exist (created via the Polar UI). " +
			"The access token is scoped to a single organization — this resource adopts it on create and releases it from state on destroy. " +
			"Only include the settings blocks you want Terraform to manage; omitted blocks are left untouched.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The organization ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the organization.",
				Optional:            true,
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "The organization slug (read-only).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"avatar_url": schema.StringAttribute{
				MarkdownDescription: "The organization avatar URL.",
				Optional:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The organization contact email.",
				Optional:            true,
			},
			"website": schema.StringAttribute{
				MarkdownDescription: "The organization website URL.",
				Optional:            true,
			},
			"socials": schema.ListNestedAttribute{
				MarkdownDescription: "List of social links for the organization.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"platform": schema.StringAttribute{
							MarkdownDescription: "The social platform. Must be one of: `x`, `github`, `facebook`, `instagram`, `youtube`, `tiktok`, `linkedin`, `other`.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("x", "github", "facebook", "instagram", "youtube", "tiktok", "linkedin", "other"),
							},
						},
						"url": schema.StringAttribute{
							MarkdownDescription: "The URL for the social link.",
							Required:            true,
						},
					},
				},
			},
			"feature_settings": schema.SingleNestedAttribute{
				MarkdownDescription: "Feature flags for the organization. Omit to leave feature settings unmanaged. Only specified fields are updated; omitted fields keep their current values.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"issue_funding_enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether issue funding is enabled.",
						Optional:            true,
						Computed:            true,
					},
					"seat_based_pricing_enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether seat-based pricing is enabled.",
						Optional:            true,
						Computed:            true,
					},
					"revops_enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether RevOps features are enabled.",
						Optional:            true,
						Computed:            true,
					},
					"wallets_enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether wallets are enabled.",
						Optional:            true,
						Computed:            true,
					},
				},
			},
			"subscription_settings": schema.SingleNestedAttribute{
				MarkdownDescription: "Subscription behavior settings. Omit to leave subscription settings unmanaged. Only specified fields are updated; omitted fields keep their current values.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"allow_multiple_subscriptions": schema.BoolAttribute{
						MarkdownDescription: "Whether customers can hold multiple active subscriptions.",
						Optional:            true,
						Computed:            true,
					},
					"allow_customer_updates": schema.BoolAttribute{
						MarkdownDescription: "Whether customers can self-manage their subscriptions.",
						Optional:            true,
						Computed:            true,
					},
					"proration_behavior": schema.StringAttribute{
						MarkdownDescription: "How mid-cycle subscription changes are billed. Must be `invoice` or `prorate`.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("invoice", "prorate"),
						},
					},
					"benefit_revocation_grace_period": schema.Int64Attribute{
						MarkdownDescription: "Number of days before benefits are revoked after subscription cancellation.",
						Optional:            true,
						Computed:            true,
					},
					"prevent_trial_abuse": schema.BoolAttribute{
						MarkdownDescription: "Whether to prevent trial abuse by restricting repeat trials.",
						Optional:            true,
						Computed:            true,
					},
				},
			},
			"notification_settings": schema.SingleNestedAttribute{
				MarkdownDescription: "Email notification preferences for the organization. Omit to leave notification settings unmanaged. Only specified fields are updated; omitted fields keep their current values.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"new_order": schema.BoolAttribute{
						MarkdownDescription: "Whether to send notifications for new orders.",
						Optional:            true,
						Computed:            true,
					},
					"new_subscription": schema.BoolAttribute{
						MarkdownDescription: "Whether to send notifications for new subscriptions.",
						Optional:            true,
						Computed:            true,
					},
				},
			},
			"customer_email_settings": schema.SingleNestedAttribute{
				MarkdownDescription: "Controls which transactional emails are sent to customers. Omit to leave customer email settings unmanaged. Only specified fields are updated; omitted fields keep their current values.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"order_confirmation": schema.BoolAttribute{
						MarkdownDescription: "Whether to send order confirmation emails.",
						Optional:            true,
						Computed:            true,
					},
					"subscription_cancellation": schema.BoolAttribute{
						MarkdownDescription: "Whether to send subscription cancellation emails.",
						Optional:            true,
						Computed:            true,
					},
					"subscription_confirmation": schema.BoolAttribute{
						MarkdownDescription: "Whether to send subscription confirmation emails.",
						Optional:            true,
						Computed:            true,
					},
					"subscription_cycled": schema.BoolAttribute{
						MarkdownDescription: "Whether to send subscription renewal emails.",
						Optional:            true,
						Computed:            true,
					},
					"subscription_cycled_after_trial": schema.BoolAttribute{
						MarkdownDescription: "Whether to send subscription renewal emails after a trial period ends.",
						Optional:            true,
						Computed:            true,
					},
					"subscription_past_due": schema.BoolAttribute{
						MarkdownDescription: "Whether to send subscription past-due emails.",
						Optional:            true,
						Computed:            true,
					},
					"subscription_revoked": schema.BoolAttribute{
						MarkdownDescription: "Whether to send subscription revoked emails.",
						Optional:            true,
						Computed:            true,
					},
					"subscription_uncanceled": schema.BoolAttribute{
						MarkdownDescription: "Whether to send subscription uncanceled emails.",
						Optional:            true,
						Computed:            true,
					},
					"subscription_updated": schema.BoolAttribute{
						MarkdownDescription: "Whether to send subscription updated emails.",
						Optional:            true,
						Computed:            true,
					},
				},
			},
		},
	}
}

func (r *OrganizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if pd := extractProviderData(req.ProviderData, &resp.Diagnostics); pd != nil {
		r.provider = pd
	}
}

// Create adopts the existing organization rather than creating one.
// Flow: discover org via token → claim singleton → update settings → poll → save.
func (r *OrganizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data OrganizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save planned website to restore formatting after API normalizes trailing slash.
	plannedWebsite := data.Website

	// The access token is scoped to exactly one org — discover it.
	org, err := discoverOrganization(ctx, r.provider)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error discovering organization",
			fmt.Sprintf("Could not discover organization: %s", err),
		)
		return
	}

	// Singleton guard — only one polar_organization resource per provider.
	if err := r.provider.ClaimOrganization(org.ID); err != nil {
		resp.Diagnostics.AddError(
			"Duplicate organization resource",
			err.Error(),
		)
		return
	}

	update, diags := buildOrganizationUpdate(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateResult, err := r.provider.Client.Organizations.Update(ctx, org.ID, *update)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating organization",
			fmt.Sprintf("Could not update organization %s: %s", org.ID, err),
		)
		return
	}

	writeTime := latestTimestamp(updateResult.Organization)

	// Eventual consistency poll.
	consistent, err := pollForConsistency(ctx, "organization", org.ID, writeTime, func() (*components.Organization, error) {
		result, err := r.provider.Client.Organizations.Get(ctx, org.ID)
		if err != nil {
			return nil, err
		}
		return result.Organization, nil
	}, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading organization after update",
			fmt.Sprintf("Could not read organization %s: %s", org.ID, err),
		)
		return
	}

	tflog.Trace(ctx, "adopted organization", map[string]interface{}{
		"id": consistent.ID,
	})

	mapOrganizationResponseToState(ctx, consistent, &data, &resp.Diagnostics)
	preserveURLFormatting(&data.Website, plannedWebsite)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OrganizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	priorWebsite := data.Website

	result, err := r.provider.Client.Organizations.Get(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading organization",
			fmt.Sprintf("Could not read organization %s: %s", data.ID.ValueString(), err),
		)
		return
	}

	mapOrganizationResponseToState(ctx, result.Organization, &data, &resp.Diagnostics)
	preserveURLFormatting(&data.Website, priorWebsite)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data OrganizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plannedWebsite := data.Website

	update, diags := buildOrganizationUpdate(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateResult, err := r.provider.Client.Organizations.Update(ctx, data.ID.ValueString(), *update)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating organization",
			fmt.Sprintf("Could not update organization %s: %s", data.ID.ValueString(), err),
		)
		return
	}

	writeTime := latestTimestamp(updateResult.Organization)

	// Eventual consistency poll.
	consistent, err := pollForConsistency(ctx, "organization", data.ID.ValueString(), writeTime, func() (*components.Organization, error) {
		result, err := r.provider.Client.Organizations.Get(ctx, data.ID.ValueString())
		if err != nil {
			return nil, err
		}
		return result.Organization, nil
	}, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading organization after update",
			fmt.Sprintf("Could not read organization %s: %s", data.ID.ValueString(), err),
		)
		return
	}

	mapOrganizationResponseToState(ctx, consistent, &data, &resp.Diagnostics)
	preserveURLFormatting(&data.Website, plannedWebsite)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes the organization from Terraform state without deleting it.
func (r *OrganizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "organization released from Terraform management")
}

func (r *OrganizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// preserveURLFormatting keeps the user's URL formatting when the API response
// differs only by a trailing slash (e.g. "https://example.com" vs "https://example.com/").
func preserveURLFormatting(current *types.String, prior types.String) {
	if current.IsNull() || prior.IsNull() {
		return
	}
	if strings.TrimRight(current.ValueString(), "/") == strings.TrimRight(prior.ValueString(), "/") {
		*current = prior
	}
}

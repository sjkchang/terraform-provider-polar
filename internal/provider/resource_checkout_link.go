// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/polarsource/polar-go"
	"github.com/polarsource/polar-go/models/components"
)

// Compile-time interface conformance checks.
var _ resource.Resource = &CheckoutLinkResource{}
var _ resource.ResourceWithImportState = &CheckoutLinkResource{}
var _ resource.ResourceWithValidateConfig = &CheckoutLinkResource{}

func NewCheckoutLinkResource() resource.Resource {
	return &CheckoutLinkResource{}
}

type CheckoutLinkResource struct {
	client *polargo.Polar
}

// CheckoutLinkResourceModel is the Terraform state shape for polar_checkout_link.
type CheckoutLinkResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	ProductIDs            types.Set    `tfsdk:"product_ids"`
	Label                 types.String `tfsdk:"label"`
	AllowDiscountCodes    types.Bool   `tfsdk:"allow_discount_codes"`
	RequireBillingAddress types.Bool   `tfsdk:"require_billing_address"`
	DiscountID            types.String `tfsdk:"discount_id"`
	SuccessURL            types.String `tfsdk:"success_url"`
	ReturnURL             types.String `tfsdk:"return_url"`
	TrialInterval         types.String `tfsdk:"trial_interval"`
	TrialIntervalCount    types.Int64  `tfsdk:"trial_interval_count"`
	Metadata              types.Map    `tfsdk:"metadata"`
	PaymentProcessor      types.String `tfsdk:"payment_processor"`
	ClientSecret          types.String `tfsdk:"client_secret"`
	URL                   types.String `tfsdk:"url"`
}

func (r *CheckoutLinkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_checkout_link"
}

func (r *CheckoutLinkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Polar checkout link. Checkout links provide a shareable URL that customers can use to purchase products.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The checkout link ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product_ids": schema.SetAttribute{
				MarkdownDescription: "Set of product IDs available at checkout. At least one product is required.",
				Required:            true,
				ElementType:         types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"label": schema.StringAttribute{
				MarkdownDescription: "Optional internal label to distinguish links. Once set, cannot be cleared without recreating the resource.",
				Optional:            true,
			},
			"allow_discount_codes": schema.BoolAttribute{
				MarkdownDescription: "Whether to allow the customer to apply discount codes. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"require_billing_address": schema.BoolAttribute{
				MarkdownDescription: "Whether to require the customer to fill their full billing address. US customers are always required regardless. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"discount_id": schema.StringAttribute{
				MarkdownDescription: "ID of a discount to pre-apply to the checkout. Once set, cannot be cleared without recreating the resource.",
				Optional:            true,
			},
			"success_url": schema.StringAttribute{
				MarkdownDescription: "URL where the customer is redirected after a successful payment. You can add `checkout_id={CHECKOUT_ID}` as a query parameter. Once set, cannot be cleared without recreating the resource.",
				Optional:            true,
			},
			"return_url": schema.StringAttribute{
				MarkdownDescription: "When set, a back button will be shown in the checkout to return to this URL. Once set, cannot be cleared without recreating the resource.",
				Optional:            true,
			},
			"trial_interval": schema.StringAttribute{
				MarkdownDescription: "The interval unit for the trial period. Must be `day`, `week`, `month`, or `year`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("day", "week", "month", "year"),
				},
			},
			"trial_interval_count": schema.Int64Attribute{
				MarkdownDescription: "The number of interval units for the trial period.",
				Optional:            true,
			},
			"metadata": schema.MapAttribute{
				MarkdownDescription: "Key-value metadata.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
			},
			"payment_processor": schema.StringAttribute{
				MarkdownDescription: "The payment processor used. Always `stripe`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "Client secret used to access the checkout link.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The shareable checkout URL.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *CheckoutLinkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if pd := extractProviderData(req.ProviderData, &resp.Diagnostics); pd != nil {
		r.client = pd.Client
	}
}

func (r *CheckoutLinkResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data CheckoutLinkResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateMetadata(ctx, data.Metadata, &resp.Diagnostics)

	// Trial fields must be set together or not at all.
	trialIntervalSet := !data.TrialInterval.IsNull() && !data.TrialInterval.IsUnknown()
	trialCountSet := !data.TrialIntervalCount.IsNull() && !data.TrialIntervalCount.IsUnknown()
	if trialIntervalSet != trialCountSet {
		resp.Diagnostics.AddError(
			"Incomplete trial configuration",
			"`trial_interval` and `trial_interval_count` must both be set or both be omitted.",
		)
	}
}

func (r *CheckoutLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CheckoutLinkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var productIDs []string
	resp.Diagnostics.Append(data.ProductIDs.ElementsAs(ctx, &productIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	create := components.CheckoutLinkCreateProducts{
		Products: productIDs,
	}

	if !data.Label.IsNull() {
		label := data.Label.ValueString()
		create.Label = &label
	}
	allowCodes := data.AllowDiscountCodes.ValueBool()
	create.AllowDiscountCodes = &allowCodes
	requireAddr := data.RequireBillingAddress.ValueBool()
	create.RequireBillingAddress = &requireAddr
	if !data.DiscountID.IsNull() {
		discountID := data.DiscountID.ValueString()
		create.DiscountID = &discountID
	}
	if !data.SuccessURL.IsNull() {
		successURL := data.SuccessURL.ValueString()
		create.SuccessURL = &successURL
	}
	if !data.ReturnURL.IsNull() {
		returnURL := data.ReturnURL.ValueString()
		create.ReturnURL = &returnURL
	}
	if !data.TrialInterval.IsNull() {
		ti := components.TrialInterval(data.TrialInterval.ValueString())
		create.TrialInterval = &ti
	}
	if !data.TrialIntervalCount.IsNull() {
		tic := data.TrialIntervalCount.ValueInt64()
		create.TrialIntervalCount = &tic
	}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateCheckoutLinkCreateProductsMetadataStr)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		create.Metadata = m
	}

	createReq := components.CreateCheckoutLinkCreateCheckoutLinkCreateProducts(create)
	result, err := r.client.CheckoutLinks.Create(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating checkout link",
			fmt.Sprintf("Could not create checkout link: %s", err),
		)
		return
	}

	tflog.Trace(ctx, "created checkout link", map[string]interface{}{
		"id": result.CheckoutLink.ID,
	})

	// CheckoutLink directly satisfies Timestamped — no adapter needed.
	cl := result.CheckoutLink
	writeTime := latestTimestamp(cl)
	checkoutLink, err := pollForConsistency(ctx, "checkout link", cl.ID, writeTime, func() (*components.CheckoutLink, error) {
		result, err := r.client.CheckoutLinks.Get(ctx, cl.ID)
		if err != nil {
			return nil, err
		}
		return result.CheckoutLink, nil
	}, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for checkout link visibility",
			fmt.Sprintf("Checkout link %s was created but not immediately readable: %s", cl.ID, err),
		)
		return
	}

	r.mapResponseToState(ctx, checkoutLink, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CheckoutLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CheckoutLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CheckoutLinks.Get(ctx, data.ID.ValueString())
	if err != nil {
		if handleNotFoundRemove(ctx, err, "checkout link", data.ID.ValueString(), &resp.State) {
			return
		}
		resp.Diagnostics.AddError(
			"Error reading checkout link",
			fmt.Sprintf("Could not read checkout link %s: %s", data.ID.ValueString(), err),
		)
		return
	}

	r.mapResponseToState(ctx, result.CheckoutLink, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CheckoutLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CheckoutLinkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var productIDs []string
	resp.Diagnostics.Append(data.ProductIDs.ElementsAs(ctx, &productIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := components.CheckoutLinkUpdate{
		Products: productIDs,
	}

	if !data.Label.IsNull() {
		label := data.Label.ValueString()
		update.Label = &label
	}
	allowCodes := data.AllowDiscountCodes.ValueBool()
	update.AllowDiscountCodes = &allowCodes
	requireAddr := data.RequireBillingAddress.ValueBool()
	update.RequireBillingAddress = &requireAddr
	if !data.DiscountID.IsNull() {
		discountID := data.DiscountID.ValueString()
		update.DiscountID = &discountID
	}
	if !data.SuccessURL.IsNull() {
		successURL := data.SuccessURL.ValueString()
		update.SuccessURL = &successURL
	}
	if !data.ReturnURL.IsNull() {
		returnURL := data.ReturnURL.ValueString()
		update.ReturnURL = &returnURL
	}
	if !data.TrialInterval.IsNull() {
		ti := components.TrialInterval(data.TrialInterval.ValueString())
		update.TrialInterval = &ti
	}
	if !data.TrialIntervalCount.IsNull() {
		tic := data.TrialIntervalCount.ValueInt64()
		update.TrialIntervalCount = &tic
	}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateCheckoutLinkUpdateMetadataStr)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		update.Metadata = m
	}

	result, err := r.client.CheckoutLinks.Update(ctx, data.ID.ValueString(), update)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating checkout link",
			fmt.Sprintf("Could not update checkout link %s: %s", data.ID.ValueString(), err),
		)
		return
	}

	checkoutLinkID := data.ID.ValueString()
	writeTime := latestTimestamp(result.CheckoutLink)
	checkoutLink, err := pollForConsistency(ctx, "checkout link", checkoutLinkID, writeTime, func() (*components.CheckoutLink, error) {
		result, err := r.client.CheckoutLinks.Get(ctx, checkoutLinkID)
		if err != nil {
			return nil, err
		}
		return result.CheckoutLink, nil
	}, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading checkout link after update",
			fmt.Sprintf("Could not read checkout link %s: %s", checkoutLinkID, err),
		)
		return
	}

	r.mapResponseToState(ctx, checkoutLink, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CheckoutLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CheckoutLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.CheckoutLinks.Delete(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting checkout link",
			fmt.Sprintf("Could not delete checkout link %s: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Trace(ctx, "deleted checkout link", map[string]interface{}{
		"id": data.ID.ValueString(),
	})
}

func (r *CheckoutLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapResponseToState maps a CheckoutLink API response to the Terraform resource model.
func (r *CheckoutLinkResource) mapResponseToState(ctx context.Context, cl *components.CheckoutLink, data *CheckoutLinkResourceModel, diags *diag.Diagnostics) {
	data.ID = types.StringValue(cl.ID)
	data.PaymentProcessor = types.StringValue(string(cl.PaymentProcessor))
	data.ClientSecret = types.StringValue(cl.ClientSecret)
	data.URL = types.StringValue(cl.URL)
	data.Label = optionalStringValue(cl.Label)
	data.AllowDiscountCodes = types.BoolValue(cl.AllowDiscountCodes)
	data.RequireBillingAddress = types.BoolValue(cl.RequireBillingAddress)
	data.DiscountID = optionalStringValue(cl.DiscountID)
	data.SuccessURL = optionalStringValue(cl.SuccessURL)
	data.ReturnURL = optionalStringValue(cl.ReturnURL)

	if cl.TrialInterval != nil {
		data.TrialInterval = types.StringValue(string(*cl.TrialInterval))
	} else {
		data.TrialInterval = types.StringNull()
	}
	data.TrialIntervalCount = optionalInt64Value(cl.TrialIntervalCount)

	// Map product IDs from response.
	productIDs := make([]string, len(cl.Products))
	for i, p := range cl.Products {
		productIDs[i] = p.ID
	}
	productSet, d := types.SetValueFrom(ctx, types.StringType, productIDs)
	diags.Append(d...)
	data.ProductIDs = productSet

	data.Metadata = sdkMetadataToMap(ctx, cl.Metadata, extractMetadataOutput, diags)
}

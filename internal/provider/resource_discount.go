// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/polarsource/polar-go"
	"github.com/polarsource/polar-go/models/components"
)

// Compile-time interface conformance checks.
var _ resource.Resource = &DiscountResource{}
var _ resource.ResourceWithImportState = &DiscountResource{}
var _ resource.ResourceWithValidateConfig = &DiscountResource{}

func NewDiscountResource() resource.Resource {
	return &DiscountResource{}
}

type DiscountResource struct {
	client *polargo.Polar
}

// DiscountResourceModel is the Terraform state shape for polar_discount.
// The SDK uses 4 distinct create variants keyed by type+duration, but we use
// a flat schema with conditional validation.
type DiscountResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Type             types.String `tfsdk:"type"`
	Duration         types.String `tfsdk:"duration"`
	Amount           types.Int64  `tfsdk:"amount"`
	Currency         types.String `tfsdk:"currency"`
	BasisPoints      types.Int64  `tfsdk:"basis_points"`
	DurationInMonths types.Int64  `tfsdk:"duration_in_months"`
	Code             types.String `tfsdk:"code"`
	StartsAt         types.String `tfsdk:"starts_at"`
	EndsAt           types.String `tfsdk:"ends_at"`
	MaxRedemptions   types.Int64  `tfsdk:"max_redemptions"`
	Products         types.List   `tfsdk:"products"`
	Metadata         types.Map    `tfsdk:"metadata"`
}

func (r *DiscountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_discount"
}

func (r *DiscountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Polar discount. Discounts can be applied to products at checkout to reduce the price by a fixed amount or percentage.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The discount ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the discount. Displayed to the customer when the discount is applied.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The discount type. Must be `fixed` or `percentage`. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("fixed", "percentage"),
				},
			},
			"duration": schema.StringAttribute{
				MarkdownDescription: "How long the discount applies. Must be `once`, `forever`, or `repeating`. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("once", "forever", "repeating"),
				},
			},
			"amount": schema.Int64Attribute{
				MarkdownDescription: "Fixed amount to discount from the invoice total (in smallest currency unit). Required when `type` is `fixed`.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.ConflictsWith(path.MatchRoot("basis_points")),
				},
			},
			"currency": schema.StringAttribute{
				MarkdownDescription: "ISO 4217 currency code. Only applicable when `type` is `fixed`. Defaults to the organization's currency.",
				Optional:            true,
				Computed:            true,
			},
			"basis_points": schema.Int64Attribute{
				MarkdownDescription: "Discount percentage in basis points (1/100th of a percent). For example, 2550 = 25.5%. Required when `type` is `percentage`.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.ConflictsWith(path.MatchRoot("amount")),
				},
			},
			"duration_in_months": schema.Int64Attribute{
				MarkdownDescription: "Number of months the discount applies. Required when `duration` is `repeating`. For yearly pricing, multiply by 12.",
				Optional:            true,
			},
			"code": schema.StringAttribute{
				MarkdownDescription: "Code customers can use to apply the discount during checkout. Must be between 3 and 256 characters. If not provided, the discount can only be applied via the API.",
				Optional:            true,
			},
			"starts_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 timestamp after which the discount is redeemable.",
				Optional:            true,
			},
			"ends_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 timestamp after which the discount is no longer redeemable.",
				Optional:            true,
			},
			"max_redemptions": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of times the discount can be redeemed.",
				Optional:            true,
			},
			"products": schema.ListAttribute{
				MarkdownDescription: "List of product IDs this discount can be applied to. If empty, the discount applies to all products.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"metadata": schema.MapAttribute{
				MarkdownDescription: "Key-value metadata.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *DiscountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if pd := extractProviderData(req.ProviderData, &resp.Diagnostics); pd != nil {
		r.client = pd.Client
	}
}

func (r *DiscountResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data DiscountResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateMetadata(ctx, data.Metadata, &resp.Diagnostics)

	// Skip validation if type or duration are unknown during plan.
	if data.Type.IsUnknown() || data.Duration.IsUnknown() {
		return
	}

	discountType := data.Type.ValueString()

	// type=fixed requires amount, type=percentage requires basis_points.
	if discountType == "fixed" && data.Amount.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("amount"),
			"Missing required attribute",
			"`amount` is required when `type` is \"fixed\".",
		)
	}
	if discountType == "percentage" && data.BasisPoints.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("basis_points"),
			"Missing required attribute",
			"`basis_points` is required when `type` is \"percentage\".",
		)
	}

	// duration=repeating requires duration_in_months.
	if data.Duration.ValueString() == "repeating" && data.DurationInMonths.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("duration_in_months"),
			"Missing required attribute",
			"`duration_in_months` is required when `duration` is \"repeating\".",
		)
	}
}

func (r *DiscountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DiscountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq, diags := buildDiscountCreateRequest(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Discounts.Create(ctx, *createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating discount",
			fmt.Sprintf("Could not create discount: %s", err),
		)
		return
	}

	tflog.Trace(ctx, "created discount", map[string]interface{}{
		"type":     data.Type.ValueString(),
		"duration": data.Duration.ValueString(),
	})

	id := discountID(result.Discount)
	writeTime := latestTimestamp(&timestampedDiscount{result.Discount})
	wrappedDiscount, err := pollForConsistency(ctx, "discount", id, writeTime, func() (*timestampedDiscount, error) {
		result, err := r.client.Discounts.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		return &timestampedDiscount{result.Discount}, nil
	}, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for discount visibility",
			fmt.Sprintf("Discount %s was created but not immediately readable: %s", id, err),
		)
		return
	}

	mapDiscountResponseToState(ctx, wrappedDiscount.Discount, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DiscountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DiscountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Discounts.Get(ctx, data.ID.ValueString())
	if err != nil {
		if handleNotFoundRemove(ctx, err, "discount", data.ID.ValueString(), &resp.State) {
			return
		}
		resp.Diagnostics.AddError(
			"Error reading discount",
			fmt.Sprintf("Could not read discount %s: %s", data.ID.ValueString(), err),
		)
		return
	}

	mapDiscountResponseToState(ctx, result.Discount, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DiscountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DiscountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq, diags := buildDiscountUpdateRequest(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Discounts.Update(ctx, data.ID.ValueString(), *updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating discount",
			fmt.Sprintf("Could not update discount %s: %s", data.ID.ValueString(), err),
		)
		return
	}

	discountID := data.ID.ValueString()
	writeTime := latestTimestamp(&timestampedDiscount{result.Discount})
	wrappedDiscount, err := pollForConsistency(ctx, "discount", discountID, writeTime, func() (*timestampedDiscount, error) {
		result, err := r.client.Discounts.Get(ctx, discountID)
		if err != nil {
			return nil, err
		}
		return &timestampedDiscount{result.Discount}, nil
	}, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading discount after update",
			fmt.Sprintf("Could not read discount %s: %s", discountID, err),
		)
		return
	}

	mapDiscountResponseToState(ctx, wrappedDiscount.Discount, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DiscountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DiscountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Discounts.Delete(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting discount",
			fmt.Sprintf("Could not delete discount %s: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Trace(ctx, "deleted discount", map[string]interface{}{
		"id": data.ID.ValueString(),
	})
}

func (r *DiscountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// --- Product ID extraction helper ---

func extractProductIDsFromDiscount(discount *components.Discount) []string {
	var products []components.DiscountProduct
	switch {
	case discount.DiscountFixedOnceForeverDuration != nil:
		products = discount.DiscountFixedOnceForeverDuration.Products
	case discount.DiscountFixedRepeatDuration != nil:
		products = discount.DiscountFixedRepeatDuration.Products
	case discount.DiscountPercentageOnceForeverDuration != nil:
		products = discount.DiscountPercentageOnceForeverDuration.Products
	case discount.DiscountPercentageRepeatDuration != nil:
		products = discount.DiscountPercentageRepeatDuration.Products
	}
	ids := make([]string, len(products))
	for i, p := range products {
		ids[i] = p.ID
	}
	return ids
}

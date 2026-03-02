// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/polarsource/polar-go/models/components"
)

// timestampedDiscount wraps *components.Discount (a union type) to satisfy the
// Timestamped interface. The Polar SDK models discounts as a union of 4 distinct
// types keyed by type+duration, so we need this adapter to delegate to the
// active variant.
type timestampedDiscount struct{ *components.Discount }

func (d *timestampedDiscount) GetCreatedAt() time.Time {
	switch {
	case d.DiscountFixedOnceForeverDuration != nil:
		return d.DiscountFixedOnceForeverDuration.GetCreatedAt()
	case d.DiscountFixedRepeatDuration != nil:
		return d.DiscountFixedRepeatDuration.GetCreatedAt()
	case d.DiscountPercentageOnceForeverDuration != nil:
		return d.DiscountPercentageOnceForeverDuration.GetCreatedAt()
	case d.DiscountPercentageRepeatDuration != nil:
		return d.DiscountPercentageRepeatDuration.GetCreatedAt()
	}
	return time.Time{}
}

func (d *timestampedDiscount) GetModifiedAt() *time.Time {
	switch {
	case d.DiscountFixedOnceForeverDuration != nil:
		return d.DiscountFixedOnceForeverDuration.GetModifiedAt()
	case d.DiscountFixedRepeatDuration != nil:
		return d.DiscountFixedRepeatDuration.GetModifiedAt()
	case d.DiscountPercentageOnceForeverDuration != nil:
		return d.DiscountPercentageOnceForeverDuration.GetModifiedAt()
	case d.DiscountPercentageRepeatDuration != nil:
		return d.DiscountPercentageRepeatDuration.GetModifiedAt()
	}
	return nil
}

// discountID extracts the ID from whichever variant is active.
func discountID(discount *components.Discount) string {
	switch {
	case discount.DiscountFixedOnceForeverDuration != nil:
		return discount.DiscountFixedOnceForeverDuration.ID
	case discount.DiscountFixedRepeatDuration != nil:
		return discount.DiscountFixedRepeatDuration.ID
	case discount.DiscountPercentageOnceForeverDuration != nil:
		return discount.DiscountPercentageOnceForeverDuration.ID
	case discount.DiscountPercentageRepeatDuration != nil:
		return discount.DiscountPercentageRepeatDuration.ID
	}
	return ""
}

// --- Build SDK Create request ---
// Discounts are polymorphic — the TF type+duration fields determine which SDK
// union variant to construct.

func buildDiscountCreateRequest(ctx context.Context, data *DiscountResourceModel) (*components.DiscountCreate, diag.Diagnostics) {
	var diags diag.Diagnostics
	discountType := data.Type.ValueString()
	duration := data.Duration.ValueString()

	// Common fields shared across all variants.
	name := data.Name.ValueString()
	var code *string
	if !data.Code.IsNull() {
		v := data.Code.ValueString()
		code = &v
	}
	startsAt := parseOptionalTime(data.StartsAt, &diags)
	endsAt := parseOptionalTime(data.EndsAt, &diags)
	if diags.HasError() {
		return nil, diags
	}
	var maxRedemptions *int64
	if !data.MaxRedemptions.IsNull() {
		v := data.MaxRedemptions.ValueInt64()
		maxRedemptions = &v
	}
	var products []string
	if !data.Products.IsNull() {
		d := data.Products.ElementsAs(ctx, &products, false)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
	}

	switch {
	case discountType == "fixed" && (duration == "once" || duration == "forever"):
		create := components.DiscountFixedOnceForeverDurationCreate{
			Type:           components.DiscountType(discountType),
			Duration:       components.DiscountDuration(duration),
			Amount:         data.Amount.ValueInt64(),
			Name:           name,
			Code:           code,
			StartsAt:       startsAt,
			EndsAt:         endsAt,
			MaxRedemptions: maxRedemptions,
			Products:       products,
		}
		if !data.Currency.IsNull() && !data.Currency.IsUnknown() {
			currency := components.PresentmentCurrency(data.Currency.ValueString())
			create.Currency = &currency
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateDiscountFixedOnceForeverDurationCreateMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			create.Metadata = m
		}
		result := components.CreateDiscountCreateDiscountFixedOnceForeverDurationCreate(create)
		return &result, diags

	case discountType == "fixed" && duration == "repeating":
		create := components.DiscountFixedRepeatDurationCreate{
			Type:             components.DiscountType(discountType),
			Duration:         components.DiscountDuration(duration),
			Amount:           data.Amount.ValueInt64(),
			DurationInMonths: data.DurationInMonths.ValueInt64(),
			Name:             name,
			Code:             code,
			StartsAt:         startsAt,
			EndsAt:           endsAt,
			MaxRedemptions:   maxRedemptions,
			Products:         products,
		}
		if !data.Currency.IsNull() && !data.Currency.IsUnknown() {
			currency := components.PresentmentCurrency(data.Currency.ValueString())
			create.Currency = &currency
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateDiscountFixedRepeatDurationCreateMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			create.Metadata = m
		}
		result := components.CreateDiscountCreateDiscountFixedRepeatDurationCreate(create)
		return &result, diags

	case discountType == "percentage" && (duration == "once" || duration == "forever"):
		create := components.DiscountPercentageOnceForeverDurationCreate{
			Type:           components.DiscountType(discountType),
			Duration:       components.DiscountDuration(duration),
			BasisPoints:    data.BasisPoints.ValueInt64(),
			Name:           name,
			Code:           code,
			StartsAt:       startsAt,
			EndsAt:         endsAt,
			MaxRedemptions: maxRedemptions,
			Products:       products,
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateDiscountPercentageOnceForeverDurationCreateMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			create.Metadata = m
		}
		result := components.CreateDiscountCreateDiscountPercentageOnceForeverDurationCreate(create)
		return &result, diags

	case discountType == "percentage" && duration == "repeating":
		create := components.DiscountPercentageRepeatDurationCreate{
			Type:             components.DiscountType(discountType),
			Duration:         components.DiscountDuration(duration),
			BasisPoints:      data.BasisPoints.ValueInt64(),
			DurationInMonths: data.DurationInMonths.ValueInt64(),
			Name:             name,
			Code:             code,
			StartsAt:         startsAt,
			EndsAt:           endsAt,
			MaxRedemptions:   maxRedemptions,
			Products:         products,
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateDiscountPercentageRepeatDurationCreateMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			create.Metadata = m
		}
		result := components.CreateDiscountCreateDiscountPercentageRepeatDurationCreate(create)
		return &result, diags

	default:
		diags.AddError("Unsupported discount variant", "Invalid combination of type and duration.")
		return nil, diags
	}
}

// --- Build SDK Update request ---
// DiscountUpdate is a flat struct (not a union), so all variants use the same type.

func buildDiscountUpdateRequest(ctx context.Context, data *DiscountResourceModel) (*components.DiscountUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics

	name := data.Name.ValueString()
	update := components.DiscountUpdate{
		Name: &name,
	}

	if !data.Code.IsNull() {
		code := data.Code.ValueString()
		update.Code = &code
	}
	if !data.MaxRedemptions.IsNull() {
		v := data.MaxRedemptions.ValueInt64()
		update.MaxRedemptions = &v
	}

	update.StartsAt = parseOptionalTime(data.StartsAt, &diags)
	update.EndsAt = parseOptionalTime(data.EndsAt, &diags)
	if diags.HasError() {
		return nil, diags
	}

	// Type-specific fields: amount/currency for fixed, basis_points for percentage.
	if !data.Amount.IsNull() {
		v := data.Amount.ValueInt64()
		update.Amount = &v
	}
	if !data.Currency.IsNull() && !data.Currency.IsUnknown() {
		currency := components.PresentmentCurrency(data.Currency.ValueString())
		update.Currency = &currency
	}
	if !data.BasisPoints.IsNull() {
		v := data.BasisPoints.ValueInt64()
		update.BasisPoints = &v
	}
	if !data.DurationInMonths.IsNull() {
		v := data.DurationInMonths.ValueInt64()
		update.DurationInMonths = &v
	}

	if !data.Products.IsNull() {
		var products []string
		d := data.Products.ElementsAs(ctx, &products, false)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		update.Products = products
	}

	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateDiscountUpdateMetadataStr)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		update.Metadata = m
	}

	return &update, diags
}

// --- Map SDK response to Terraform state ---

func mapDiscountResponseToState(ctx context.Context, discount *components.Discount, data *DiscountResourceModel, diags *diag.Diagnostics) {
	switch {
	case discount.DiscountFixedOnceForeverDuration != nil:
		d := discount.DiscountFixedOnceForeverDuration
		data.ID = types.StringValue(d.ID)
		data.Name = types.StringValue(d.Name)
		data.Type = types.StringValue(string(d.Type))
		data.Duration = types.StringValue(string(d.Duration))
		data.Amount = types.Int64Value(d.Amount)
		data.Currency = types.StringValue(d.Currency)
		data.BasisPoints = types.Int64Null()
		data.DurationInMonths = types.Int64Null()
		setDiscountCommonOptionals(d.Code, d.StartsAt, d.EndsAt, d.MaxRedemptions, data)
		data.Metadata = sdkMetadataToMap(ctx, d.Metadata, extractMetadataOutput, diags)

	case discount.DiscountFixedRepeatDuration != nil:
		d := discount.DiscountFixedRepeatDuration
		data.ID = types.StringValue(d.ID)
		data.Name = types.StringValue(d.Name)
		data.Type = types.StringValue(string(d.Type))
		data.Duration = types.StringValue(string(d.Duration))
		data.Amount = types.Int64Value(d.Amount)
		data.Currency = types.StringValue(d.Currency)
		data.BasisPoints = types.Int64Null()
		data.DurationInMonths = types.Int64Value(d.DurationInMonths)
		setDiscountCommonOptionals(d.Code, d.StartsAt, d.EndsAt, d.MaxRedemptions, data)
		data.Metadata = sdkMetadataToMap(ctx, d.Metadata, extractMetadataOutput, diags)

	case discount.DiscountPercentageOnceForeverDuration != nil:
		d := discount.DiscountPercentageOnceForeverDuration
		data.ID = types.StringValue(d.ID)
		data.Name = types.StringValue(d.Name)
		data.Type = types.StringValue(string(d.Type))
		data.Duration = types.StringValue(string(d.Duration))
		data.Amount = types.Int64Null()
		data.Currency = types.StringNull()
		data.BasisPoints = types.Int64Value(d.BasisPoints)
		data.DurationInMonths = types.Int64Null()
		setDiscountCommonOptionals(d.Code, d.StartsAt, d.EndsAt, d.MaxRedemptions, data)
		data.Metadata = sdkMetadataToMap(ctx, d.Metadata, extractMetadataOutput, diags)

	case discount.DiscountPercentageRepeatDuration != nil:
		d := discount.DiscountPercentageRepeatDuration
		data.ID = types.StringValue(d.ID)
		data.Name = types.StringValue(d.Name)
		data.Type = types.StringValue(string(d.Type))
		data.Duration = types.StringValue(string(d.Duration))
		data.Amount = types.Int64Null()
		data.Currency = types.StringNull()
		data.BasisPoints = types.Int64Value(d.BasisPoints)
		data.DurationInMonths = types.Int64Value(d.DurationInMonths)
		setDiscountCommonOptionals(d.Code, d.StartsAt, d.EndsAt, d.MaxRedemptions, data)
		data.Metadata = sdkMetadataToMap(ctx, d.Metadata, extractMetadataOutput, diags)

	default:
		diags.AddError("Unknown discount type", "Could not determine discount type from API response.")
	}

	// Map products (extract IDs from the response's DiscountProduct objects).
	productIDs := extractProductIDsFromDiscount(discount)
	if len(productIDs) > 0 {
		productList, d := types.ListValueFrom(ctx, types.StringType, productIDs)
		diags.Append(d...)
		data.Products = productList
	} else if !data.Products.IsNull() {
		// User configured products but response has none — keep as empty list.
		productList, d := types.ListValueFrom(ctx, types.StringType, []string{})
		diags.Append(d...)
		data.Products = productList
	}
}

// --- Shared helpers ---

func setDiscountCommonOptionals(code *string, startsAt, endsAt *time.Time, maxRedemptions *int64, data *DiscountResourceModel) {
	data.Code = optionalStringValue(code)
	if startsAt != nil {
		data.StartsAt = types.StringValue(startsAt.Format(time.RFC3339))
	} else {
		data.StartsAt = types.StringNull()
	}
	if endsAt != nil {
		data.EndsAt = types.StringValue(endsAt.Format(time.RFC3339))
	} else {
		data.EndsAt = types.StringNull()
	}
	data.MaxRedemptions = optionalInt64Value(maxRedemptions)
}

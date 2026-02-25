// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/polarsource/polar-go/models/components"
)

// --- Build SDK Create request ---
// Products are polymorphic at two levels:
// 1. Recurring vs one-time (determined by recurring_interval being set)
// 2. Price type (fixed/free/custom/metered_unit per price entry)

func buildProductCreateRequest(ctx context.Context, data *ProductResourceModel) (*components.ProductCreate, diag.Diagnostics) {
	var diags diag.Diagnostics

	name := data.Name.ValueString()

	var description *string
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		d := data.Description.ValueString()
		description = &d
	}

	var medias []string
	if !data.Medias.IsNull() && !data.Medias.IsUnknown() {
		d := data.Medias.ElementsAs(ctx, &medias, false)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
	}

	if !data.RecurringInterval.IsNull() && !data.RecurringInterval.IsUnknown() {
		// Recurring product
		prices, d := pricesToRecurringCreateSDK(data.Prices)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}

		recurring := components.ProductCreateRecurring{
			Name:              name,
			Description:       description,
			Prices:            prices,
			RecurringInterval: components.SubscriptionRecurringInterval(data.RecurringInterval.ValueString()),
			Medias:            medias,
		}

		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateProductCreateRecurringMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			recurring.Metadata = m
		}

		result := components.CreateProductCreateProductCreateRecurring(recurring)
		return &result, diags
	}

	// One-time product
	prices, d := pricesToOneTimeCreateSDK(data.Prices)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	oneTime := components.ProductCreateOneTime{
		Name:        name,
		Description: description,
		Prices:      prices,
		Medias:      medias,
	}

	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateProductCreateOneTimeMetadataStr)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		oneTime.Metadata = m
	}

	result := components.CreateProductCreateProductCreateOneTime(oneTime)
	return &result, diags
}

// --- Build SDK Update request ---
// Updates receive the current prices from the API so we can match unchanged
// prices by value and send their existing IDs (avoids unnecessary recreation).

func buildProductUpdateRequest(ctx context.Context, data *ProductResourceModel, currentPrices []components.Prices) (*components.ProductUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics

	name := data.Name.ValueString()
	update := components.ProductUpdate{
		Name: &name,
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		d := data.Description.ValueString()
		update.Description = &d
	}

	prices, d := pricesToUpdateSDK(data.Prices, currentPrices)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	update.Prices = prices

	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateProductUpdateMetadataStr)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		update.Metadata = m
	}

	if !data.Medias.IsNull() && !data.Medias.IsUnknown() {
		var medias []string
		d := data.Medias.ElementsAs(ctx, &medias, false)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		update.Medias = medias
	}

	isArchived := data.IsArchived.ValueBool()
	update.IsArchived = &isArchived

	return &update, diags
}

// --- Map SDK response to Terraform state ---

// mapProductResponseToState converts the full product API response into the TF model.
func mapProductResponseToState(ctx context.Context, product *components.Product, data *ProductResourceModel, diags *diag.Diagnostics) {
	data.ID = types.StringValue(product.ID)
	data.Name = types.StringValue(product.Name)
	data.IsArchived = types.BoolValue(product.IsArchived)
	data.Description = optionalStringValue(product.Description)

	if product.RecurringInterval != nil {
		data.RecurringInterval = types.StringValue(string(*product.RecurringInterval))
	} else {
		data.RecurringInterval = types.StringNull()
	}

	// Map prices
	data.Prices = sdkPricesToModel(product.Prices, diags)

	// Map metadata
	data.Metadata = sdkMetadataToMap(ctx, product.Metadata, func(v components.MetadataOutputType) metadataFields {
		return metadataFields{Str: v.Str, Integer: v.Integer, Number: v.Number, Boolean: v.Boolean}
	}, diags)

	// Map benefit_ids — only if the user opted into TF-managed benefits (non-null).
	// If benefit_ids was omitted from config, we leave it null so Terraform
	// doesn't try to manage benefits that were attached via the dashboard.
	if !data.BenefitIDs.IsNull() {
		ids := make([]string, 0, len(product.Benefits))
		for _, b := range product.Benefits {
			if id := benefitID(b); id != "" {
				ids = append(ids, id)
			}
		}
		benefitSet, d := types.SetValueFrom(ctx, types.StringType, ids)
		diags.Append(d...)
		data.BenefitIDs = benefitSet
	}

	// Map medias — only if the user opted into TF-managed medias (non-null).
	// Same pattern as benefit_ids: if omitted from config, leave null to avoid
	// oscillation between null and [] on every plan.
	if !data.Medias.IsNull() {
		mediaIDs := make([]string, len(product.Medias))
		for i, m := range product.Medias {
			mediaIDs[i] = m.ID
		}
		mediaList, d := types.ListValueFrom(ctx, types.StringType, mediaIDs)
		diags.Append(d...)
		data.Medias = mediaList
	}
}

// --- Price conversion helpers ---

func optionalCurrency(p PriceModel) *components.PresentmentCurrency {
	if !p.PriceCurrency.IsNull() && !p.PriceCurrency.IsUnknown() {
		c := components.PresentmentCurrency(p.PriceCurrency.ValueString())
		return &c
	}
	return nil
}

func buildFixedPriceCreate(p PriceModel) components.ProductPriceFixedCreate {
	return components.ProductPriceFixedCreate{
		PriceAmount:   p.PriceAmount.ValueInt64(),
		PriceCurrency: optionalCurrency(p),
	}
}

func buildCustomPriceCreate(p PriceModel) components.ProductPriceCustomCreate {
	create := components.ProductPriceCustomCreate{
		PriceCurrency: optionalCurrency(p),
	}
	if !p.MinimumAmount.IsNull() {
		v := p.MinimumAmount.ValueInt64()
		create.MinimumAmount = &v
	}
	if !p.MaximumAmount.IsNull() {
		v := p.MaximumAmount.ValueInt64()
		create.MaximumAmount = &v
	}
	if !p.PresetAmount.IsNull() {
		v := p.PresetAmount.ValueInt64()
		create.PresetAmount = &v
	}
	return create
}

func buildMeteredUnitPriceCreate(p PriceModel) components.ProductPriceMeteredUnitCreate {
	create := components.ProductPriceMeteredUnitCreate{
		MeterID:       p.MeterID.ValueString(),
		UnitAmount:    components.CreateUnitAmountStr(p.UnitAmount.ValueString()),
		PriceCurrency: optionalCurrency(p),
	}
	if !p.CapAmount.IsNull() {
		v := p.CapAmount.ValueInt64()
		create.CapAmount = &v
	}
	return create
}

func pricesToRecurringCreateSDK(prices []PriceModel) ([]components.ProductCreateRecurringPrices, diag.Diagnostics) {
	var diags diag.Diagnostics
	result := make([]components.ProductCreateRecurringPrices, len(prices))
	for i, p := range prices {
		switch p.AmountType.ValueString() {
		case "fixed":
			result[i] = components.CreateProductCreateRecurringPricesFixed(buildFixedPriceCreate(p))
		case "free":
			result[i] = components.CreateProductCreateRecurringPricesFree(components.ProductPriceFreeCreate{})
		case "custom":
			result[i] = components.CreateProductCreateRecurringPricesCustom(buildCustomPriceCreate(p))
		case "metered_unit":
			result[i] = components.CreateProductCreateRecurringPricesMeteredUnit(buildMeteredUnitPriceCreate(p))
		default:
			diags.AddError("Unsupported price type", "Price amount_type must be one of: fixed, free, custom, metered_unit.")
			return nil, diags
		}
	}
	return result, diags
}

func pricesToOneTimeCreateSDK(prices []PriceModel) ([]components.ProductCreateOneTimePrices, diag.Diagnostics) {
	var diags diag.Diagnostics
	result := make([]components.ProductCreateOneTimePrices, len(prices))
	for i, p := range prices {
		switch p.AmountType.ValueString() {
		case "fixed":
			result[i] = components.CreateProductCreateOneTimePricesFixed(buildFixedPriceCreate(p))
		case "free":
			result[i] = components.CreateProductCreateOneTimePricesFree(components.ProductPriceFreeCreate{})
		case "custom":
			result[i] = components.CreateProductCreateOneTimePricesCustom(buildCustomPriceCreate(p))
		case "metered_unit":
			result[i] = components.CreateProductCreateOneTimePricesMeteredUnit(buildMeteredUnitPriceCreate(p))
		default:
			diags.AddError("Unsupported price type", "Price amount_type must be one of: fixed, free, custom, metered_unit.")
			return nil, diags
		}
	}
	return result, diags
}

// --- Existing price matching for updates ---
// When updating a product, Polar expects either an existing price ID (to keep it)
// or a new price definition (to create it). We match planned prices against
// current API prices by value to determine which can be reused.

type existingPrice struct {
	id   string
	data PriceModel
	used bool // prevents the same API price from being matched twice
}

// extractExistingPrices converts the API price response into matchable structs.
func extractExistingPrices(apiPrices []components.Prices) []*existingPrice {
	var result []*existingPrice
	for _, p := range apiPrices {
		if p.ProductPrice == nil {
			continue
		}
		model := sdkProductPriceToModel(p.ProductPrice)
		if model == nil {
			continue
		}
		var id string
		switch {
		case p.ProductPrice.ProductPriceFixed != nil:
			id = p.ProductPrice.ProductPriceFixed.ID
		case p.ProductPrice.ProductPriceFree != nil:
			id = p.ProductPrice.ProductPriceFree.ID
		case p.ProductPrice.ProductPriceCustom != nil:
			id = p.ProductPrice.ProductPriceCustom.ID
		case p.ProductPrice.ProductPriceMeteredUnit != nil:
			id = p.ProductPrice.ProductPriceMeteredUnit.ID
		}
		result = append(result, &existingPrice{id: id, data: *model})
	}
	return result
}

// pricesMatch compares the user-specified fields of two price models.
func pricesMatch(planned, existing PriceModel) bool {
	if planned.AmountType.ValueString() != existing.AmountType.ValueString() {
		return false
	}
	switch planned.AmountType.ValueString() {
	case "fixed":
		if planned.PriceAmount.ValueInt64() != existing.PriceAmount.ValueInt64() {
			return false
		}
		if !planned.PriceCurrency.IsNull() && !planned.PriceCurrency.IsUnknown() &&
			planned.PriceCurrency.ValueString() != existing.PriceCurrency.ValueString() {
			return false
		}
		return true
	case "free":
		return true
	case "custom":
		return optionalInt64Equal(planned.MinimumAmount, existing.MinimumAmount) &&
			optionalInt64Equal(planned.MaximumAmount, existing.MaximumAmount) &&
			optionalInt64Equal(planned.PresetAmount, existing.PresetAmount)
	case "metered_unit":
		if planned.MeterID.ValueString() != existing.MeterID.ValueString() {
			return false
		}
		if !numericStringsEqual(planned.UnitAmount.ValueString(), existing.UnitAmount.ValueString()) {
			return false
		}
		return optionalInt64Equal(planned.CapAmount, existing.CapAmount)
	}
	return false
}

func optionalInt64Equal(a, b types.Int64) bool {
	if a.IsNull() && b.IsNull() {
		return true
	}
	if a.IsNull() != b.IsNull() {
		return false
	}
	return a.ValueInt64() == b.ValueInt64()
}

// pricesToUpdateSDK builds the update price list. For each planned price:
// 1. Try to find an existing API price with matching values → reuse its ID
// 2. If no match → create a new price definition.
func pricesToUpdateSDK(planned []PriceModel, currentPrices []components.Prices) ([]components.ProductUpdatePrices, diag.Diagnostics) {
	var diags diag.Diagnostics
	existing := extractExistingPrices(currentPrices)
	result := make([]components.ProductUpdatePrices, len(planned))

	for i, p := range planned {
		// Try to reuse an existing price if values match.
		var matched bool
		for _, ep := range existing {
			if !ep.used && pricesMatch(p, ep.data) {
				ep.used = true
				result[i] = components.CreateProductUpdatePricesExistingProductPrice(
					components.ExistingProductPrice{ID: ep.id},
				)
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		// Create new price
		switch p.AmountType.ValueString() {
		case "fixed":
			result[i] = components.CreateProductUpdatePricesTwo(
				components.CreateTwoFixed(buildFixedPriceCreate(p)),
			)
		case "free":
			result[i] = components.CreateProductUpdatePricesTwo(
				components.CreateTwoFree(components.ProductPriceFreeCreate{}),
			)
		case "custom":
			result[i] = components.CreateProductUpdatePricesTwo(
				components.CreateTwoCustom(buildCustomPriceCreate(p)),
			)
		case "metered_unit":
			result[i] = components.CreateProductUpdatePricesTwo(
				components.CreateTwoMeteredUnit(buildMeteredUnitPriceCreate(p)),
			)
		default:
			diags.AddError("Unsupported price type", "Price amount_type must be one of: fixed, free, custom, metered_unit.")
			return nil, diags
		}
	}
	return result, diags
}

// --- Benefit helpers ---

// extractBenefitIDsFromSet converts a types.Set of benefit IDs to a []string.
func extractBenefitIDsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []string {
	var ids []string
	d := set.ElementsAs(ctx, &ids, false)
	diags.Append(d...)
	if ids == nil {
		ids = []string{}
	}
	return ids
}

// benefitID extracts the ID string from a components.Benefit union type.
func benefitID(b components.Benefit) string {
	switch {
	case b.BenefitCustom != nil:
		return b.BenefitCustom.ID
	case b.BenefitDiscord != nil:
		return b.BenefitDiscord.ID
	case b.BenefitGitHubRepository != nil:
		return b.BenefitGitHubRepository.ID
	case b.BenefitDownloadables != nil:
		return b.BenefitDownloadables.ID
	case b.BenefitLicenseKeys != nil:
		return b.BenefitLicenseKeys.ID
	case b.BenefitMeterCredit != nil:
		return b.BenefitMeterCredit.ID
	default:
		return ""
	}
}

// --- Response mapping (API → TF state) ---

func sdkPricesToModel(prices []components.Prices, diags *diag.Diagnostics) []PriceModel {
	result := make([]PriceModel, 0, len(prices))
	for _, p := range prices {
		if p.ProductPrice != nil {
			model := sdkProductPriceToModel(p.ProductPrice)
			if model != nil {
				result = append(result, *model)
			} else {
				diags.AddWarning(
					"Unknown price type",
					"The API returned a price type not recognized by this provider version. The price was omitted from state.",
				)
			}
		}
	}
	return result
}

// nullPriceModel creates a PriceModel with all optional fields set to null.
// The caller then fills in only the fields relevant to the amount_type.
func nullPriceModel(amountType string, currency types.String) PriceModel {
	return PriceModel{
		AmountType:    types.StringValue(amountType),
		PriceCurrency: currency,
		PriceAmount:   types.Int64Null(),
		MinimumAmount: types.Int64Null(),
		MaximumAmount: types.Int64Null(),
		PresetAmount:  types.Int64Null(),
		MeterID:       types.StringNull(),
		UnitAmount:    types.StringNull(),
		CapAmount:     types.Int64Null(),
	}
}

// sdkProductPriceToModel converts one SDK price union variant → flat PriceModel.
func sdkProductPriceToModel(price *components.ProductPrice) *PriceModel {
	switch {
	case price.ProductPriceFixed != nil:
		pp := price.ProductPriceFixed
		m := nullPriceModel("fixed", types.StringValue(pp.PriceCurrency))
		m.PriceAmount = types.Int64Value(pp.PriceAmount)
		return &m

	case price.ProductPriceFree != nil:
		m := nullPriceModel("free", types.StringNull())
		return &m

	case price.ProductPriceCustom != nil:
		pp := price.ProductPriceCustom
		m := nullPriceModel("custom", types.StringValue(pp.PriceCurrency))
		m.MinimumAmount = types.Int64Value(pp.MinimumAmount)
		m.MaximumAmount = optionalInt64Value(pp.MaximumAmount)
		m.PresetAmount = optionalInt64Value(pp.PresetAmount)
		return &m

	case price.ProductPriceMeteredUnit != nil:
		pp := price.ProductPriceMeteredUnit
		m := nullPriceModel("metered_unit", types.StringValue(pp.PriceCurrency))
		m.MeterID = types.StringValue(pp.MeterID)
		m.UnitAmount = types.StringValue(normalizeDecimalString(pp.UnitAmount))
		m.CapAmount = optionalInt64Value(pp.CapAmount)
		return &m

	default:
		return nil
	}
}

// normalizeDecimalString strips trailing zeros from a decimal string so that
// API values like "0.500000000000" become "0.5".
func normalizeDecimalString(s string) string {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// numericStringsEqual returns true if two numeric strings represent the same
// value (e.g. "0.50" and "0.5" and "0.500000000000" are all equal).
func numericStringsEqual(a, b string) bool {
	af, aErr := strconv.ParseFloat(a, 64)
	bf, bErr := strconv.ParseFloat(b, 64)
	return aErr == nil && bErr == nil && af == bf
}

// preserveUnitAmountFormatting restores the user's original unit_amount
// formatting when the API value is numerically equivalent. This prevents
// Terraform from detecting spurious diffs due to trailing-zero differences
// (e.g. user writes "0.50", API returns "0.500000000000", normalized to "0.5").
//
// Prices are matched by content (via pricesMatch) rather than by index,
// so reordering, inserting, or removing prices won't mis-apply formatting.
func preserveUnitAmountFormatting(prices []PriceModel, priorPrices []PriceModel) {
	used := make([]bool, len(priorPrices))
	for i := range prices {
		if prices[i].UnitAmount.IsNull() {
			continue
		}
		for j := range priorPrices {
			if used[j] || priorPrices[j].UnitAmount.IsNull() {
				continue
			}
			if pricesMatch(prices[i], priorPrices[j]) {
				prices[i].UnitAmount = priorPrices[j].UnitAmount
				used[j] = true
				break
			}
		}
	}
}

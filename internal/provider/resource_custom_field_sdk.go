// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/polarsource/polar-go/models/components"
)

// timestampedCustomField wraps *components.CustomField (a union type) to satisfy
// the Timestamped interface.
type timestampedCustomField struct{ *components.CustomField }

func (c *timestampedCustomField) GetCreatedAt() time.Time {
	switch {
	case c.CustomFieldText != nil:
		return c.CustomFieldText.GetCreatedAt()
	case c.CustomFieldNumber != nil:
		return c.CustomFieldNumber.GetCreatedAt()
	case c.CustomFieldDate != nil:
		return c.CustomFieldDate.GetCreatedAt()
	case c.CustomFieldCheckbox != nil:
		return c.CustomFieldCheckbox.GetCreatedAt()
	case c.CustomFieldSelect != nil:
		return c.CustomFieldSelect.GetCreatedAt()
	}
	return time.Time{}
}

func (c *timestampedCustomField) GetModifiedAt() *time.Time {
	switch {
	case c.CustomFieldText != nil:
		return c.CustomFieldText.GetModifiedAt()
	case c.CustomFieldNumber != nil:
		return c.CustomFieldNumber.GetModifiedAt()
	case c.CustomFieldDate != nil:
		return c.CustomFieldDate.GetModifiedAt()
	case c.CustomFieldCheckbox != nil:
		return c.CustomFieldCheckbox.GetModifiedAt()
	case c.CustomFieldSelect != nil:
		return c.CustomFieldSelect.GetModifiedAt()
	}
	return nil
}

// customFieldID extracts the ID from whichever variant is active.
func customFieldID(cf *components.CustomField) string {
	switch {
	case cf.CustomFieldText != nil:
		return cf.CustomFieldText.ID
	case cf.CustomFieldNumber != nil:
		return cf.CustomFieldNumber.ID
	case cf.CustomFieldDate != nil:
		return cf.CustomFieldDate.ID
	case cf.CustomFieldCheckbox != nil:
		return cf.CustomFieldCheckbox.ID
	case cf.CustomFieldSelect != nil:
		return cf.CustomFieldSelect.ID
	}
	return ""
}

// --- Build SDK Create request ---

func buildCustomFieldCreateRequest(ctx context.Context, data *CustomFieldResourceModel) (*components.CustomFieldCreate, diag.Diagnostics) {
	var diags diag.Diagnostics
	fieldType := data.Type.ValueString()
	slug := data.Slug.ValueString()
	name := data.Name.ValueString()

	switch fieldType {
	case "text":
		props := buildTextProperties(data.Properties)
		create := components.CustomFieldCreateText{
			Slug:       slug,
			Name:       name,
			Properties: props,
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateCustomFieldCreateTextMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			create.Metadata = m
		}
		result := components.CreateCustomFieldCreateText(create)
		return &result, diags

	case "number":
		props := buildNumberProperties(data.Properties)
		create := components.CustomFieldCreateNumber{
			Slug:       slug,
			Name:       name,
			Properties: props,
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateCustomFieldCreateNumberMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			create.Metadata = m
		}
		result := components.CreateCustomFieldCreateNumber(create)
		return &result, diags

	case "date":
		props := buildDateProperties(data.Properties)
		create := components.CustomFieldCreateDate{
			Slug:       slug,
			Name:       name,
			Properties: props,
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateCustomFieldCreateDateMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			create.Metadata = m
		}
		result := components.CreateCustomFieldCreateDate(create)
		return &result, diags

	case "checkbox":
		props := buildCheckboxProperties(data.Properties)
		create := components.CustomFieldCreateCheckbox{
			Slug:       slug,
			Name:       name,
			Properties: props,
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateCustomFieldCreateCheckboxMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			create.Metadata = m
		}
		result := components.CreateCustomFieldCreateCheckbox(create)
		return &result, diags

	case "select":
		props := buildSelectProperties(data.Properties)
		create := components.CustomFieldCreateSelect{
			Slug:       slug,
			Name:       name,
			Properties: props,
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateCustomFieldCreateSelectMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			create.Metadata = m
		}
		result := components.CreateCustomFieldCreateSelect(create)
		return &result, diags

	default:
		diags.AddError("Unsupported custom field type", "Type "+fieldType+" is not supported.")
		return nil, diags
	}
}

// --- Build SDK Update request ---

func buildCustomFieldUpdateRequest(ctx context.Context, data *CustomFieldResourceModel) (*components.CustomFieldUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics
	fieldType := data.Type.ValueString()
	name := data.Name.ValueString()
	slug := data.Slug.ValueString()

	switch fieldType {
	case "text":
		update := components.CustomFieldUpdateText{
			Name: &name,
			Slug: &slug,
		}
		if data.Properties != nil {
			props := buildTextProperties(data.Properties)
			update.Properties = &props
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateCustomFieldUpdateTextMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			update.Metadata = m
		}
		result := components.CreateCustomFieldUpdateText(update)
		return &result, diags

	case "number":
		update := components.CustomFieldUpdateNumber{
			Name: &name,
			Slug: &slug,
		}
		if data.Properties != nil {
			props := buildNumberProperties(data.Properties)
			update.Properties = &props
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateCustomFieldUpdateNumberMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			update.Metadata = m
		}
		result := components.CreateCustomFieldUpdateNumber(update)
		return &result, diags

	case "date":
		update := components.CustomFieldUpdateDate{
			Name: &name,
			Slug: &slug,
		}
		if data.Properties != nil {
			props := buildDateProperties(data.Properties)
			update.Properties = &props
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateCustomFieldUpdateDateMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			update.Metadata = m
		}
		result := components.CreateCustomFieldUpdateDate(update)
		return &result, diags

	case "checkbox":
		update := components.CustomFieldUpdateCheckbox{
			Name: &name,
			Slug: &slug,
		}
		if data.Properties != nil {
			props := buildCheckboxProperties(data.Properties)
			update.Properties = &props
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateCustomFieldUpdateCheckboxMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			update.Metadata = m
		}
		result := components.CreateCustomFieldUpdateCheckbox(update)
		return &result, diags

	case "select":
		update := components.CustomFieldUpdateSelect{
			Name: &name,
			Slug: &slug,
		}
		if data.Properties != nil {
			props := buildSelectProperties(data.Properties)
			update.Properties = &props
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateCustomFieldUpdateSelectMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			update.Metadata = m
		}
		result := components.CreateCustomFieldUpdateSelect(update)
		return &result, diags

	default:
		diags.AddError("Unsupported custom field type", "Type "+fieldType+" is not supported.")
		return nil, diags
	}
}

// --- Map SDK response to Terraform state ---

func mapCustomFieldResponseToState(ctx context.Context, cf *components.CustomField, data *CustomFieldResourceModel, diags *diag.Diagnostics) {
	switch {
	case cf.CustomFieldText != nil:
		f := cf.CustomFieldText
		data.ID = types.StringValue(f.ID)
		data.Type = types.StringValue("text")
		data.Slug = types.StringValue(f.Slug)
		data.Name = types.StringValue(f.Name)
		data.Properties = mapTextPropertiesToModel(&f.Properties)
		data.Metadata = sdkMetadataToMap(ctx, f.Metadata, extractMetadataOutput, diags)

	case cf.CustomFieldNumber != nil:
		f := cf.CustomFieldNumber
		data.ID = types.StringValue(f.ID)
		data.Type = types.StringValue("number")
		data.Slug = types.StringValue(f.Slug)
		data.Name = types.StringValue(f.Name)
		data.Properties = mapNumberPropertiesToModel(&f.Properties)
		data.Metadata = sdkMetadataToMap(ctx, f.Metadata, extractMetadataOutput, diags)

	case cf.CustomFieldDate != nil:
		f := cf.CustomFieldDate
		data.ID = types.StringValue(f.ID)
		data.Type = types.StringValue("date")
		data.Slug = types.StringValue(f.Slug)
		data.Name = types.StringValue(f.Name)
		data.Properties = mapDatePropertiesToModel(&f.Properties)
		data.Metadata = sdkMetadataToMap(ctx, f.Metadata, extractMetadataOutput, diags)

	case cf.CustomFieldCheckbox != nil:
		f := cf.CustomFieldCheckbox
		data.ID = types.StringValue(f.ID)
		data.Type = types.StringValue("checkbox")
		data.Slug = types.StringValue(f.Slug)
		data.Name = types.StringValue(f.Name)
		data.Properties = mapCheckboxPropertiesToModel(&f.Properties)
		data.Metadata = sdkMetadataToMap(ctx, f.Metadata, extractMetadataOutput, diags)

	case cf.CustomFieldSelect != nil:
		f := cf.CustomFieldSelect
		data.ID = types.StringValue(f.ID)
		data.Type = types.StringValue("select")
		data.Slug = types.StringValue(f.Slug)
		data.Name = types.StringValue(f.Name)
		data.Properties = mapSelectPropertiesToModel(&f.Properties)
		data.Metadata = sdkMetadataToMap(ctx, f.Metadata, extractMetadataOutput, diags)

	default:
		diags.AddError("Unknown custom field type", "Could not determine custom field type from API response.")
	}
}

// --- Properties builders (TF model → SDK) ---

func buildTextProperties(props *CustomFieldPropertiesModel) components.CustomFieldTextProperties {
	if props == nil {
		return components.CustomFieldTextProperties{}
	}
	result := components.CustomFieldTextProperties{}
	if !props.FormLabel.IsNull() {
		v := props.FormLabel.ValueString()
		result.FormLabel = &v
	}
	if !props.FormHelpText.IsNull() {
		v := props.FormHelpText.ValueString()
		result.FormHelpText = &v
	}
	if !props.FormPlaceholder.IsNull() {
		v := props.FormPlaceholder.ValueString()
		result.FormPlaceholder = &v
	}
	if !props.Textarea.IsNull() {
		v := props.Textarea.ValueBool()
		result.Textarea = &v
	}
	if !props.MinLength.IsNull() {
		v := props.MinLength.ValueInt64()
		result.MinLength = &v
	}
	if !props.MaxLength.IsNull() {
		v := props.MaxLength.ValueInt64()
		result.MaxLength = &v
	}
	return result
}

func buildNumberProperties(props *CustomFieldPropertiesModel) components.CustomFieldNumberProperties {
	if props == nil {
		return components.CustomFieldNumberProperties{}
	}
	result := components.CustomFieldNumberProperties{}
	if !props.FormLabel.IsNull() {
		v := props.FormLabel.ValueString()
		result.FormLabel = &v
	}
	if !props.FormHelpText.IsNull() {
		v := props.FormHelpText.ValueString()
		result.FormHelpText = &v
	}
	if !props.FormPlaceholder.IsNull() {
		v := props.FormPlaceholder.ValueString()
		result.FormPlaceholder = &v
	}
	if !props.Ge.IsNull() {
		v := props.Ge.ValueInt64()
		result.Ge = &v
	}
	if !props.Le.IsNull() {
		v := props.Le.ValueInt64()
		result.Le = &v
	}
	return result
}

func buildDateProperties(props *CustomFieldPropertiesModel) components.CustomFieldDateProperties {
	if props == nil {
		return components.CustomFieldDateProperties{}
	}
	result := components.CustomFieldDateProperties{}
	if !props.FormLabel.IsNull() {
		v := props.FormLabel.ValueString()
		result.FormLabel = &v
	}
	if !props.FormHelpText.IsNull() {
		v := props.FormHelpText.ValueString()
		result.FormHelpText = &v
	}
	if !props.FormPlaceholder.IsNull() {
		v := props.FormPlaceholder.ValueString()
		result.FormPlaceholder = &v
	}
	if !props.Ge.IsNull() {
		v := props.Ge.ValueInt64()
		result.Ge = &v
	}
	if !props.Le.IsNull() {
		v := props.Le.ValueInt64()
		result.Le = &v
	}
	return result
}

func buildCheckboxProperties(props *CustomFieldPropertiesModel) components.CustomFieldCheckboxProperties {
	if props == nil {
		return components.CustomFieldCheckboxProperties{}
	}
	result := components.CustomFieldCheckboxProperties{}
	if !props.FormLabel.IsNull() {
		v := props.FormLabel.ValueString()
		result.FormLabel = &v
	}
	if !props.FormHelpText.IsNull() {
		v := props.FormHelpText.ValueString()
		result.FormHelpText = &v
	}
	if !props.FormPlaceholder.IsNull() {
		v := props.FormPlaceholder.ValueString()
		result.FormPlaceholder = &v
	}
	return result
}

func buildSelectProperties(props *CustomFieldPropertiesModel) components.CustomFieldSelectProperties {
	if props == nil {
		return components.CustomFieldSelectProperties{}
	}
	result := components.CustomFieldSelectProperties{}
	if !props.FormLabel.IsNull() {
		v := props.FormLabel.ValueString()
		result.FormLabel = &v
	}
	if !props.FormHelpText.IsNull() {
		v := props.FormHelpText.ValueString()
		result.FormHelpText = &v
	}
	if !props.FormPlaceholder.IsNull() {
		v := props.FormPlaceholder.ValueString()
		result.FormPlaceholder = &v
	}
	if props.Options != nil {
		options := make([]components.CustomFieldSelectOption, len(props.Options))
		for i, o := range props.Options {
			options[i] = components.CustomFieldSelectOption{
				Value: o.Value.ValueString(),
				Label: o.Label.ValueString(),
			}
		}
		result.Options = options
	}
	return result
}

// --- Properties mappers (SDK → TF model) ---

func mapTextPropertiesToModel(props *components.CustomFieldTextProperties) *CustomFieldPropertiesModel {
	return &CustomFieldPropertiesModel{
		FormLabel:       optionalStringValue(props.FormLabel),
		FormHelpText:    optionalStringValue(props.FormHelpText),
		FormPlaceholder: optionalStringValue(props.FormPlaceholder),
		Textarea:        optionalBoolValue(props.Textarea),
		MinLength:       optionalInt64Value(props.MinLength),
		MaxLength:       optionalInt64Value(props.MaxLength),
		Ge:              types.Int64Null(),
		Le:              types.Int64Null(),
		Options:         nil,
	}
}

func mapNumberPropertiesToModel(props *components.CustomFieldNumberProperties) *CustomFieldPropertiesModel {
	return &CustomFieldPropertiesModel{
		FormLabel:       optionalStringValue(props.FormLabel),
		FormHelpText:    optionalStringValue(props.FormHelpText),
		FormPlaceholder: optionalStringValue(props.FormPlaceholder),
		Textarea:        types.BoolNull(),
		MinLength:       types.Int64Null(),
		MaxLength:       types.Int64Null(),
		Ge:              optionalInt64Value(props.Ge),
		Le:              optionalInt64Value(props.Le),
		Options:         nil,
	}
}

func mapDatePropertiesToModel(props *components.CustomFieldDateProperties) *CustomFieldPropertiesModel {
	return &CustomFieldPropertiesModel{
		FormLabel:       optionalStringValue(props.FormLabel),
		FormHelpText:    optionalStringValue(props.FormHelpText),
		FormPlaceholder: optionalStringValue(props.FormPlaceholder),
		Textarea:        types.BoolNull(),
		MinLength:       types.Int64Null(),
		MaxLength:       types.Int64Null(),
		Ge:              optionalInt64Value(props.Ge),
		Le:              optionalInt64Value(props.Le),
		Options:         nil,
	}
}

func mapCheckboxPropertiesToModel(props *components.CustomFieldCheckboxProperties) *CustomFieldPropertiesModel {
	return &CustomFieldPropertiesModel{
		FormLabel:       optionalStringValue(props.FormLabel),
		FormHelpText:    optionalStringValue(props.FormHelpText),
		FormPlaceholder: optionalStringValue(props.FormPlaceholder),
		Textarea:        types.BoolNull(),
		MinLength:       types.Int64Null(),
		MaxLength:       types.Int64Null(),
		Ge:              types.Int64Null(),
		Le:              types.Int64Null(),
		Options:         nil,
	}
}

func mapSelectPropertiesToModel(props *components.CustomFieldSelectProperties) *CustomFieldPropertiesModel {
	var options []CustomFieldOptionModel
	if len(props.Options) > 0 {
		options = make([]CustomFieldOptionModel, len(props.Options))
		for i, o := range props.Options {
			options[i] = CustomFieldOptionModel{
				Value: types.StringValue(o.Value),
				Label: types.StringValue(o.Label),
			}
		}
	}
	return &CustomFieldPropertiesModel{
		FormLabel:       optionalStringValue(props.FormLabel),
		FormHelpText:    optionalStringValue(props.FormHelpText),
		FormPlaceholder: optionalStringValue(props.FormPlaceholder),
		Textarea:        types.BoolNull(),
		MinLength:       types.Int64Null(),
		MaxLength:       types.Int64Null(),
		Ge:              types.Int64Null(),
		Le:              types.Int64Null(),
		Options:         options,
	}
}

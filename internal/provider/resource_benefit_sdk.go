// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/polarsource/polar-go/models/components"
	"github.com/polarsource/polar-go/models/operations"
)

// timestampedBenefit wraps *components.Benefit (a union type) to satisfy the
// Timestamped interface. The Polar SDK models benefits as a union of distinct
// types (BenefitCustom, BenefitDiscord, etc.) rather than a single type with
// variant properties, so we need this adapter to delegate to the active variant.
type timestampedBenefit struct{ *components.Benefit }

func (b *timestampedBenefit) GetCreatedAt() time.Time {
	switch {
	case b.BenefitCustom != nil:
		return b.BenefitCustom.GetCreatedAt()
	case b.BenefitDiscord != nil:
		return b.BenefitDiscord.GetCreatedAt()
	case b.BenefitGitHubRepository != nil:
		return b.BenefitGitHubRepository.GetCreatedAt()
	case b.BenefitDownloadables != nil:
		return b.BenefitDownloadables.GetCreatedAt()
	case b.BenefitLicenseKeys != nil:
		return b.BenefitLicenseKeys.GetCreatedAt()
	case b.BenefitMeterCredit != nil:
		return b.BenefitMeterCredit.GetCreatedAt()
	}
	return time.Time{}
}

func (b *timestampedBenefit) GetModifiedAt() *time.Time {
	switch {
	case b.BenefitCustom != nil:
		return b.BenefitCustom.GetModifiedAt()
	case b.BenefitDiscord != nil:
		return b.BenefitDiscord.GetModifiedAt()
	case b.BenefitGitHubRepository != nil:
		return b.BenefitGitHubRepository.GetModifiedAt()
	case b.BenefitDownloadables != nil:
		return b.BenefitDownloadables.GetModifiedAt()
	case b.BenefitLicenseKeys != nil:
		return b.BenefitLicenseKeys.GetModifiedAt()
	case b.BenefitMeterCredit != nil:
		return b.BenefitMeterCredit.GetModifiedAt()
	}
	return nil
}

// --- Build SDK Create request ---
// Benefits are polymorphic — the TF `type` field determines which SDK union
// variant to construct. Each builder function handles one benefit type.

func buildBenefitCreateRequest(ctx context.Context, data *BenefitResourceModel) (*components.BenefitCreate, diag.Diagnostics) {
	var diags diag.Diagnostics
	benefitType := data.Type.ValueString()
	description := data.Description.ValueString()

	switch benefitType {
	case "custom":
		return buildCustomCreate(ctx, description, data, &diags)
	case "discord":
		return buildDiscordCreate(ctx, description, data, &diags)
	case "github_repository":
		return buildGitHubRepositoryCreate(ctx, description, data, &diags)
	case "downloadables":
		return buildDownloadablesCreate(ctx, description, data, &diags)
	case "license_keys":
		return buildLicenseKeysCreate(ctx, description, data, &diags)
	case "meter_credit":
		return buildMeterCreditCreate(ctx, description, data, &diags)
	default:
		diags.AddError("Unsupported benefit type", fmt.Sprintf("Benefit type %q is not supported.", benefitType))
		return nil, diags
	}
}

func buildCustomCreate(ctx context.Context, description string, data *BenefitResourceModel, diags *diag.Diagnostics) (*components.BenefitCreate, diag.Diagnostics) {
	props := components.BenefitCustomCreateProperties{}
	if data.CustomProperties != nil && !data.CustomProperties.Note.IsNull() {
		note := data.CustomProperties.Note.ValueString()
		props.Note = &note
	}
	create := components.BenefitCustomCreate{
		Description: description,
		Properties:  props,
	}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateBenefitCustomCreateMetadataStr)
		diags.Append(d...)
		if diags.HasError() {
			return nil, *diags
		}
		create.Metadata = m
	}
	result := components.CreateBenefitCreateCustom(create)
	return &result, *diags
}

func buildDiscordCreate(ctx context.Context, description string, data *BenefitResourceModel, diags *diag.Diagnostics) (*components.BenefitCreate, diag.Diagnostics) {
	if data.DiscordProperties == nil {
		diags.AddError("Missing properties", "discord_properties is required when type is discord.")
		return nil, *diags
	}
	create := components.BenefitDiscordCreate{
		Description: description,
		Properties: components.BenefitDiscordCreateProperties{
			GuildToken: data.DiscordProperties.GuildToken.ValueString(),
			RoleID:     data.DiscordProperties.RoleID.ValueString(),
			KickMember: data.DiscordProperties.KickMember.ValueBool(),
		},
	}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateBenefitDiscordCreateMetadataStr)
		diags.Append(d...)
		if diags.HasError() {
			return nil, *diags
		}
		create.Metadata = m
	}
	result := components.CreateBenefitCreateDiscord(create)
	return &result, *diags
}

func buildGitHubRepositoryCreate(ctx context.Context, description string, data *BenefitResourceModel, diags *diag.Diagnostics) (*components.BenefitCreate, diag.Diagnostics) {
	if data.GitHubRepositoryProperties == nil {
		diags.AddError("Missing properties", "github_repository_properties is required when type is github_repository.")
		return nil, *diags
	}
	create := components.BenefitGitHubRepositoryCreate{
		Description: description,
		Properties: components.BenefitGitHubRepositoryCreateProperties{
			RepositoryOwner: data.GitHubRepositoryProperties.RepositoryOwner.ValueString(),
			RepositoryName:  data.GitHubRepositoryProperties.RepositoryName.ValueString(),
			Permission:      components.BenefitGitHubRepositoryCreatePropertiesPermission(data.GitHubRepositoryProperties.Permission.ValueString()),
		},
	}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateBenefitGitHubRepositoryCreateMetadataStr)
		diags.Append(d...)
		if diags.HasError() {
			return nil, *diags
		}
		create.Metadata = m
	}
	result := components.CreateBenefitCreateGithubRepository(create)
	return &result, *diags
}

func buildDownloadablesCreate(ctx context.Context, description string, data *BenefitResourceModel, diags *diag.Diagnostics) (*components.BenefitCreate, diag.Diagnostics) {
	if data.DownloadablesProperties == nil {
		diags.AddError("Missing properties", "downloadables_properties is required when type is downloadables.")
		return nil, *diags
	}
	var files []string
	d := data.DownloadablesProperties.Files.ElementsAs(ctx, &files, false)
	diags.Append(d...)
	if diags.HasError() {
		return nil, *diags
	}
	create := components.BenefitDownloadablesCreate{
		Description: description,
		Properties: components.BenefitDownloadablesCreateProperties{
			Files: files,
		},
	}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateBenefitDownloadablesCreateMetadataStr)
		diags.Append(d...)
		if diags.HasError() {
			return nil, *diags
		}
		create.Metadata = m
	}
	result := components.CreateBenefitCreateDownloadables(create)
	return &result, *diags
}

func buildLicenseKeysCreate(ctx context.Context, description string, data *BenefitResourceModel, diags *diag.Diagnostics) (*components.BenefitCreate, diag.Diagnostics) {
	if data.LicenseKeysProperties == nil {
		diags.AddError("Missing properties", "license_keys_properties is required when type is license_keys.")
		return nil, *diags
	}
	props := licenseKeysPropsToSDK(data.LicenseKeysProperties)
	create := components.BenefitLicenseKeysCreate{
		Description: description,
		Properties:  props,
	}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateBenefitLicenseKeysCreateMetadataStr)
		diags.Append(d...)
		if diags.HasError() {
			return nil, *diags
		}
		create.Metadata = m
	}
	result := components.CreateBenefitCreateLicenseKeys(create)
	return &result, *diags
}

func buildMeterCreditCreate(ctx context.Context, description string, data *BenefitResourceModel, diags *diag.Diagnostics) (*components.BenefitCreate, diag.Diagnostics) {
	if data.MeterCreditProperties == nil {
		diags.AddError("Missing properties", "meter_credit_properties is required when type is meter_credit.")
		return nil, *diags
	}
	create := components.BenefitMeterCreditCreate{
		Description: description,
		Properties: components.BenefitMeterCreditCreateProperties{
			MeterID:  data.MeterCreditProperties.MeterID.ValueString(),
			Units:    data.MeterCreditProperties.Units.ValueInt64(),
			Rollover: data.MeterCreditProperties.Rollover.ValueBool(),
		},
	}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateBenefitMeterCreditCreateMetadataStr)
		diags.Append(d...)
		if diags.HasError() {
			return nil, *diags
		}
		create.Metadata = m
	}
	result := components.CreateBenefitCreateMeterCredit(create)
	return &result, *diags
}

// --- Build SDK Update request ---
// Same polymorphic dispatch as create. The update types use a different SDK
// union (BenefitsUpdateBenefitUpdate) with slightly different field types.

func buildBenefitUpdateRequest(ctx context.Context, data *BenefitResourceModel) (*operations.BenefitsUpdateBenefitUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics
	benefitType := data.Type.ValueString()
	description := data.Description.ValueString()

	switch benefitType {
	case "custom":
		update := components.BenefitCustomUpdate{Description: &description}
		if data.CustomProperties != nil {
			props := components.BenefitCustomProperties{}
			if !data.CustomProperties.Note.IsNull() {
				note := data.CustomProperties.Note.ValueString()
				props.Note = &note
			}
			update.Properties = &props
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateBenefitCustomUpdateMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			update.Metadata = m
		}
		result := operations.CreateBenefitsUpdateBenefitUpdateBenefitCustomUpdate(update)
		return &result, diags

	case "discord":
		update := components.BenefitDiscordUpdate{Description: &description}
		if data.DiscordProperties != nil {
			update.Properties = &components.BenefitDiscordCreateProperties{
				GuildToken: data.DiscordProperties.GuildToken.ValueString(),
				RoleID:     data.DiscordProperties.RoleID.ValueString(),
				KickMember: data.DiscordProperties.KickMember.ValueBool(),
			}
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateBenefitDiscordUpdateMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			update.Metadata = m
		}
		result := operations.CreateBenefitsUpdateBenefitUpdateBenefitDiscordUpdate(update)
		return &result, diags

	case "github_repository":
		update := components.BenefitGitHubRepositoryUpdate{Description: &description}
		if data.GitHubRepositoryProperties != nil {
			update.Properties = &components.BenefitGitHubRepositoryCreateProperties{
				RepositoryOwner: data.GitHubRepositoryProperties.RepositoryOwner.ValueString(),
				RepositoryName:  data.GitHubRepositoryProperties.RepositoryName.ValueString(),
				Permission:      components.BenefitGitHubRepositoryCreatePropertiesPermission(data.GitHubRepositoryProperties.Permission.ValueString()),
			}
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateBenefitGitHubRepositoryUpdateMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			update.Metadata = m
		}
		result := operations.CreateBenefitsUpdateBenefitUpdateBenefitGitHubRepositoryUpdate(update)
		return &result, diags

	case "downloadables":
		update := components.BenefitDownloadablesUpdate{Description: &description}
		if data.DownloadablesProperties != nil {
			var files []string
			d := data.DownloadablesProperties.Files.ElementsAs(ctx, &files, false)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			update.Properties = &components.BenefitDownloadablesCreateProperties{
				Files: files,
			}
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateBenefitDownloadablesUpdateMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			update.Metadata = m
		}
		result := operations.CreateBenefitsUpdateBenefitUpdateBenefitDownloadablesUpdate(update)
		return &result, diags

	case "license_keys":
		update := components.BenefitLicenseKeysUpdate{Description: &description}
		if data.LicenseKeysProperties != nil {
			props := licenseKeysPropsToSDK(data.LicenseKeysProperties)
			update.Properties = &props
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateBenefitLicenseKeysUpdateMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			update.Metadata = m
		}
		result := operations.CreateBenefitsUpdateBenefitUpdateBenefitLicenseKeysUpdate(update)
		return &result, diags

	case "meter_credit":
		update := components.BenefitMeterCreditUpdate{Description: &description}
		if data.MeterCreditProperties != nil {
			update.Properties = &components.BenefitMeterCreditCreateProperties{
				MeterID:  data.MeterCreditProperties.MeterID.ValueString(),
				Units:    data.MeterCreditProperties.Units.ValueInt64(),
				Rollover: data.MeterCreditProperties.Rollover.ValueBool(),
			}
		}
		if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
			m, d := metadataToCreateSDK(ctx, data.Metadata, components.CreateBenefitMeterCreditUpdateMetadataStr)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			update.Metadata = m
		}
		result := operations.CreateBenefitsUpdateBenefitUpdateBenefitMeterCreditUpdate(update)
		return &result, diags

	default:
		diags.AddError("Unsupported benefit type", fmt.Sprintf("Benefit type %q is not supported.", benefitType))
		return nil, diags
	}
}

// --- Map SDK response to Terraform state ---
// The API response is a union with one active variant. We clear all properties
// blocks first, then populate only the matching one based on which variant is set.

func mapBenefitResponseToState(ctx context.Context, benefit *components.Benefit, data *BenefitResourceModel, diags *diag.Diagnostics) {
	// Reset all type-specific properties — the switch below sets only the active one.
	data.CustomProperties = nil
	data.DiscordProperties = nil
	data.GitHubRepositoryProperties = nil
	data.DownloadablesProperties = nil
	data.LicenseKeysProperties = nil
	data.MeterCreditProperties = nil

	switch {
	case benefit.BenefitCustom != nil:
		b := benefit.BenefitCustom
		setBenefitCommonFields(b.ID, "custom", b.Description, data)
		data.CustomProperties = &BenefitCustomPropertiesModel{
			Note: optionalStringValue(b.Properties.Note),
		}
		data.Metadata = sdkMetadataToMap(ctx, b.Metadata, func(v components.MetadataOutputType) metadataFields {
			return metadataFields{Str: v.Str, Integer: v.Integer, Number: v.Number, Boolean: v.Boolean}
		}, diags)

	case benefit.BenefitDiscord != nil:
		b := benefit.BenefitDiscord
		setBenefitCommonFields(b.ID, "discord", b.Description, data)
		data.DiscordProperties = &BenefitDiscordPropertiesModel{
			GuildToken: types.StringValue(b.Properties.GuildToken),
			RoleID:     types.StringValue(b.Properties.RoleID),
			KickMember: types.BoolValue(b.Properties.KickMember),
			GuildID:    types.StringValue(b.Properties.GuildID),
		}
		data.Metadata = sdkMetadataToMap(ctx, b.Metadata, func(v components.MetadataOutputType) metadataFields {
			return metadataFields{Str: v.Str, Integer: v.Integer, Number: v.Number, Boolean: v.Boolean}
		}, diags)

	case benefit.BenefitGitHubRepository != nil:
		b := benefit.BenefitGitHubRepository
		setBenefitCommonFields(b.ID, "github_repository", b.Description, data)
		data.GitHubRepositoryProperties = &BenefitGitHubRepositoryPropertiesModel{
			RepositoryOwner: types.StringValue(b.Properties.RepositoryOwner),
			RepositoryName:  types.StringValue(b.Properties.RepositoryName),
			Permission:      types.StringValue(string(b.Properties.Permission)),
		}
		data.Metadata = sdkMetadataToMap(ctx, b.Metadata, func(v components.MetadataOutputType) metadataFields {
			return metadataFields{Str: v.Str, Integer: v.Integer, Number: v.Number, Boolean: v.Boolean}
		}, diags)

	case benefit.BenefitDownloadables != nil:
		b := benefit.BenefitDownloadables
		setBenefitCommonFields(b.ID, "downloadables", b.Description, data)
		filesList, d := types.ListValueFrom(ctx, types.StringType, b.Properties.Files)
		diags.Append(d...)
		data.DownloadablesProperties = &BenefitDownloadablesPropertiesModel{
			Files: filesList,
		}
		data.Metadata = sdkMetadataToMap(ctx, b.Metadata, func(v components.MetadataOutputType) metadataFields {
			return metadataFields{Str: v.Str, Integer: v.Integer, Number: v.Number, Boolean: v.Boolean}
		}, diags)

	case benefit.BenefitLicenseKeys != nil:
		b := benefit.BenefitLicenseKeys
		setBenefitCommonFields(b.ID, "license_keys", b.Description, data)
		data.LicenseKeysProperties = sdkLicenseKeysPropsToModel(&b.Properties)
		data.Metadata = sdkMetadataToMap(ctx, b.Metadata, func(v components.MetadataOutputType) metadataFields {
			return metadataFields{Str: v.Str, Integer: v.Integer, Number: v.Number, Boolean: v.Boolean}
		}, diags)

	case benefit.BenefitMeterCredit != nil:
		b := benefit.BenefitMeterCredit
		setBenefitCommonFields(b.ID, "meter_credit", b.Description, data)
		data.MeterCreditProperties = &BenefitMeterCreditPropertiesModel{
			MeterID:  types.StringValue(b.Properties.MeterID),
			Units:    types.Int64Value(b.Properties.Units),
			Rollover: types.BoolValue(b.Properties.Rollover),
		}
		data.Metadata = sdkMetadataToMap(ctx, b.Metadata, func(v components.MetadataOutputType) metadataFields {
			return metadataFields{Str: v.Str, Integer: v.Integer, Number: v.Number, Boolean: v.Boolean}
		}, diags)

	default:
		diags.AddError("Unknown benefit type", "Could not determine benefit type from API response.")
	}
}

// --- Shared helpers ---

func setBenefitCommonFields(id, benefitType, description string, data *BenefitResourceModel) {
	data.ID = types.StringValue(id)
	data.Type = types.StringValue(benefitType)
	data.Description = types.StringValue(description)
}

// licenseKeysPropsToSDK converts the TF model to SDK create properties (used for both create and update).
func licenseKeysPropsToSDK(lk *BenefitLicenseKeysPropertiesModel) components.BenefitLicenseKeysCreateProperties {
	props := components.BenefitLicenseKeysCreateProperties{}
	if !lk.Prefix.IsNull() {
		prefix := lk.Prefix.ValueString()
		props.Prefix = &prefix
	}
	if !lk.LimitUsage.IsNull() {
		limitUsage := lk.LimitUsage.ValueInt64()
		props.LimitUsage = &limitUsage
	}
	if lk.Expires != nil {
		props.Expires = &components.BenefitLicenseKeyExpirationProperties{
			TTL:       lk.Expires.TTL.ValueInt64(),
			Timeframe: components.Timeframe(lk.Expires.Timeframe.ValueString()),
		}
	}
	if lk.Activations != nil {
		props.Activations = &components.BenefitLicenseKeyActivationCreateProperties{
			Limit:               lk.Activations.Limit.ValueInt64(),
			EnableCustomerAdmin: lk.Activations.EnableCustomerAdmin.ValueBool(),
		}
	}
	return props
}

// sdkLicenseKeysPropsToModel converts SDK response properties to the TF model.
func sdkLicenseKeysPropsToModel(props *components.BenefitLicenseKeysProperties) *BenefitLicenseKeysPropertiesModel {
	model := &BenefitLicenseKeysPropertiesModel{
		Prefix:     optionalStringValue(props.Prefix),
		LimitUsage: optionalInt64Value(props.LimitUsage),
	}
	if props.Expires != nil {
		model.Expires = &BenefitLicenseKeyExpirationModel{
			TTL:       types.Int64Value(props.Expires.TTL),
			Timeframe: types.StringValue(string(props.Expires.Timeframe)),
		}
	}
	if props.Activations != nil {
		model.Activations = &BenefitLicenseKeyActivationModel{
			Limit:               types.Int64Value(props.Activations.Limit),
			EnableCustomerAdmin: types.BoolValue(props.Activations.EnableCustomerAdmin),
		}
	}
	return model
}

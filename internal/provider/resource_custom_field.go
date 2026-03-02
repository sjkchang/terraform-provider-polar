// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

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
)

// Compile-time interface conformance checks.
var _ resource.Resource = &CustomFieldResource{}
var _ resource.ResourceWithImportState = &CustomFieldResource{}
var _ resource.ResourceWithValidateConfig = &CustomFieldResource{}

func NewCustomFieldResource() resource.Resource {
	return &CustomFieldResource{}
}

type CustomFieldResource struct {
	client *polargo.Polar
}

// CustomFieldResourceModel is the Terraform state shape for polar_custom_field.
type CustomFieldResourceModel struct {
	ID         types.String              `tfsdk:"id"`
	Type       types.String              `tfsdk:"type"`
	Slug       types.String              `tfsdk:"slug"`
	Name       types.String              `tfsdk:"name"`
	Metadata   types.Map                 `tfsdk:"metadata"`
	Properties *CustomFieldPropertiesModel `tfsdk:"properties"`
}

// CustomFieldPropertiesModel is a unified properties block for all custom field types.
// Fields that don't apply to the current type are simply null.
type CustomFieldPropertiesModel struct {
	FormLabel       types.String               `tfsdk:"form_label"`
	FormHelpText    types.String               `tfsdk:"form_help_text"`
	FormPlaceholder types.String               `tfsdk:"form_placeholder"`
	Textarea        types.Bool                 `tfsdk:"textarea"`
	MinLength       types.Int64                `tfsdk:"min_length"`
	MaxLength       types.Int64                `tfsdk:"max_length"`
	Ge              types.Int64                `tfsdk:"ge"`
	Le              types.Int64                `tfsdk:"le"`
	Options         []CustomFieldOptionModel   `tfsdk:"options"`
}

type CustomFieldOptionModel struct {
	Value types.String `tfsdk:"value"`
	Label types.String `tfsdk:"label"`
}

func (r *CustomFieldResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_field"
}

func (r *CustomFieldResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Polar custom field. Custom fields allow you to collect additional information from customers during checkout.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The custom field ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The custom field type. Must be `text`, `number`, `date`, `checkbox`, or `select`. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("text", "number", "date", "checkbox", "select"),
				},
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "Identifier of the custom field. Used as the key when storing the value. Must contain only lowercase letters, numbers, hyphens, and underscores.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the custom field.",
				Required:            true,
			},
			"metadata": schema.MapAttribute{
				MarkdownDescription: "Key-value metadata.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
			},
			"properties": schema.SingleNestedAttribute{
				MarkdownDescription: "Type-specific properties. Fields that don't apply to the current type should be omitted.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"form_label": schema.StringAttribute{
						MarkdownDescription: "Label shown on the form.",
						Optional:            true,
					},
					"form_help_text": schema.StringAttribute{
						MarkdownDescription: "Help text shown below the field.",
						Optional:            true,
					},
					"form_placeholder": schema.StringAttribute{
						MarkdownDescription: "Placeholder text for the input.",
						Optional:            true,
					},
					"textarea": schema.BoolAttribute{
						MarkdownDescription: "Whether to render as a textarea (text type only).",
						Optional:            true,
					},
					"min_length": schema.Int64Attribute{
						MarkdownDescription: "Minimum length (text type only).",
						Optional:            true,
					},
					"max_length": schema.Int64Attribute{
						MarkdownDescription: "Maximum length (text type only).",
						Optional:            true,
					},
					"ge": schema.Int64Attribute{
						MarkdownDescription: "Minimum value (number and date types only).",
						Optional:            true,
					},
					"le": schema.Int64Attribute{
						MarkdownDescription: "Maximum value (number and date types only).",
						Optional:            true,
					},
					"options": schema.ListNestedAttribute{
						MarkdownDescription: "List of selectable options (select type only).",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"value": schema.StringAttribute{
									MarkdownDescription: "The option value.",
									Required:            true,
								},
								"label": schema.StringAttribute{
									MarkdownDescription: "The display label for the option.",
									Required:            true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *CustomFieldResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if pd := extractProviderData(req.ProviderData, &resp.Diagnostics); pd != nil {
		r.client = pd.Client
	}
}

func (r *CustomFieldResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data CustomFieldResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateMetadata(ctx, data.Metadata, &resp.Diagnostics)

	if data.Type.IsUnknown() || data.Properties == nil {
		return
	}

	fieldType := data.Type.ValueString()

	// options is only valid for select type.
	if fieldType != "select" && data.Properties.Options != nil && len(data.Properties.Options) > 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("properties").AtName("options"),
			"Invalid property for type",
			fmt.Sprintf("`options` can only be set when type is \"select\", got %q.", fieldType),
		)
	}

	// textarea, min_length, max_length are only valid for text type.
	if fieldType != "text" {
		if !data.Properties.Textarea.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("properties").AtName("textarea"),
				"Invalid property for type",
				fmt.Sprintf("`textarea` can only be set when type is \"text\", got %q.", fieldType),
			)
		}
		if !data.Properties.MinLength.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("properties").AtName("min_length"),
				"Invalid property for type",
				fmt.Sprintf("`min_length` can only be set when type is \"text\", got %q.", fieldType),
			)
		}
		if !data.Properties.MaxLength.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("properties").AtName("max_length"),
				"Invalid property for type",
				fmt.Sprintf("`max_length` can only be set when type is \"text\", got %q.", fieldType),
			)
		}
	}

	// ge, le are only valid for number and date types.
	if fieldType != "number" && fieldType != "date" {
		if !data.Properties.Ge.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("properties").AtName("ge"),
				"Invalid property for type",
				fmt.Sprintf("`ge` can only be set when type is \"number\" or \"date\", got %q.", fieldType),
			)
		}
		if !data.Properties.Le.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("properties").AtName("le"),
				"Invalid property for type",
				fmt.Sprintf("`le` can only be set when type is \"number\" or \"date\", got %q.", fieldType),
			)
		}
	}
}

func (r *CustomFieldResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CustomFieldResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq, diags := buildCustomFieldCreateRequest(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CustomFields.Create(ctx, *createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating custom field",
			fmt.Sprintf("Could not create custom field: %s", err),
		)
		return
	}

	tflog.Trace(ctx, "created custom field", map[string]interface{}{
		"type": data.Type.ValueString(),
		"slug": data.Slug.ValueString(),
	})

	id := customFieldID(result.CustomField)
	writeTime := latestTimestamp(&timestampedCustomField{result.CustomField})
	wrappedCF, err := pollForConsistency(ctx, "custom field", id, writeTime, func() (*timestampedCustomField, error) {
		result, err := r.client.CustomFields.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		return &timestampedCustomField{result.CustomField}, nil
	}, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for custom field visibility",
			fmt.Sprintf("Custom field %s was created but not immediately readable: %s", id, err),
		)
		return
	}

	mapCustomFieldResponseToState(ctx, wrappedCF.CustomField, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CustomFieldResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CustomFieldResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CustomFields.Get(ctx, data.ID.ValueString())
	if err != nil {
		if handleNotFoundRemove(ctx, err, "custom field", data.ID.ValueString(), &resp.State) {
			return
		}
		resp.Diagnostics.AddError(
			"Error reading custom field",
			fmt.Sprintf("Could not read custom field %s: %s", data.ID.ValueString(), err),
		)
		return
	}

	mapCustomFieldResponseToState(ctx, result.CustomField, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CustomFieldResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CustomFieldResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq, diags := buildCustomFieldUpdateRequest(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CustomFields.Update(ctx, data.ID.ValueString(), *updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating custom field",
			fmt.Sprintf("Could not update custom field %s: %s", data.ID.ValueString(), err),
		)
		return
	}

	cfID := data.ID.ValueString()
	writeTime := latestTimestamp(&timestampedCustomField{result.CustomField})
	wrappedCF, err := pollForConsistency(ctx, "custom field", cfID, writeTime, func() (*timestampedCustomField, error) {
		result, err := r.client.CustomFields.Get(ctx, cfID)
		if err != nil {
			return nil, err
		}
		return &timestampedCustomField{result.CustomField}, nil
	}, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading custom field after update",
			fmt.Sprintf("Could not read custom field %s: %s", cfID, err),
		)
		return
	}

	mapCustomFieldResponseToState(ctx, wrappedCF.CustomField, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CustomFieldResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CustomFieldResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.CustomFields.Delete(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting custom field",
			fmt.Sprintf("Could not delete custom field %s: %s", data.ID.ValueString(), err),
		)
		return
	}

	tflog.Trace(ctx, "deleted custom field", map[string]interface{}{
		"id": data.ID.ValueString(),
	})
}

func (r *CustomFieldResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

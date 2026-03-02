// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/polarsource/polar-go/models/components"
)

// --- Metadata conversion ---
// Terraform stores metadata as map[string]string (all values are strings).
// The Polar SDK uses typed union values (string | int | float | bool) per key.
// These helpers bridge the two representations.

// metadataFields is a common shape to extract union values from any SDK metadata type.
type metadataFields struct {
	Str     *string
	Integer *int64
	Number  *float64
	Boolean *bool
}

// metadataToCreateSDK converts TF map[string]string → SDK metadata map.
// The factory creates the type-specific SDK union variant from a string value.
func metadataToCreateSDK[T any](ctx context.Context, metadata types.Map, factory func(string) T) (map[string]T, diag.Diagnostics) {
	var stringMap map[string]string
	diags := metadata.ElementsAs(ctx, &stringMap, false)
	if diags.HasError() {
		return nil, diags
	}
	result := make(map[string]T, len(stringMap))
	for k, v := range stringMap {
		result[k] = factory(v)
	}
	return result, diags
}

// sdkMetadataToMap converts SDK metadata → TF map[string]string.
// The extract function pulls the union value pointers from the SDK type.
// Always returns a non-null map (empty {} for nil input) so Optional+Computed
// metadata attributes don't oscillate between null and {} across plan/apply.
func sdkMetadataToMap[T any](ctx context.Context, metadata map[string]T, extract func(T) metadataFields, diags *diag.Diagnostics) types.Map {
	if len(metadata) == 0 {
		result, d := types.MapValueFrom(ctx, types.StringType, map[string]string{})
		diags.Append(d...)
		return result
	}
	stringMap := make(map[string]string, len(metadata))
	for k, v := range metadata {
		f := extract(v)
		switch {
		case f.Str != nil:
			stringMap[k] = *f.Str
		case f.Integer != nil:
			stringMap[k] = strconv.FormatInt(*f.Integer, 10)
		case f.Number != nil:
			stringMap[k] = strconv.FormatFloat(*f.Number, 'f', -1, 64)
		case f.Boolean != nil:
			stringMap[k] = strconv.FormatBool(*f.Boolean)
		}
	}
	result, d := types.MapValueFrom(ctx, types.StringType, stringMap)
	diags.Append(d...)
	return result
}

// extractMetadataOutput is the standard extractor for MetadataOutputType used
// across resources that return metadata in API responses.
func extractMetadataOutput(v components.MetadataOutputType) metadataFields {
	return metadataFields{Str: v.Str, Integer: v.Integer, Number: v.Number, Boolean: v.Boolean}
}

// validateMetadata checks plan-phase constraints for metadata fields:
//   - Max 50 key-value pairs
//   - Key length ≤ 40 characters
//   - Value length ≤ 500 characters
func validateMetadata(ctx context.Context, metadata types.Map, diags *diag.Diagnostics) {
	if metadata.IsNull() || metadata.IsUnknown() {
		return
	}

	var stringMap map[string]string
	d := metadata.ElementsAs(ctx, &stringMap, false)
	diags.Append(d...)
	if d.HasError() {
		return
	}

	metaPath := path.Root("metadata")

	if len(stringMap) > 50 {
		diags.AddAttributeError(
			metaPath,
			"Too many metadata entries",
			fmt.Sprintf("Metadata supports a maximum of 50 key-value pairs, got %d.", len(stringMap)),
		)
	}

	for k, v := range stringMap {
		if len(k) > 40 {
			diags.AddAttributeError(
				metaPath.AtMapKey(k),
				"Metadata key too long",
				fmt.Sprintf("Metadata key %q has %d characters, maximum allowed is 40.", k, len(k)),
			)
		}
		if len(v) > 500 {
			diags.AddAttributeError(
				metaPath.AtMapKey(k),
				"Metadata value too long",
				fmt.Sprintf("Metadata value for key %q has %d characters, maximum allowed is 500.", k, len(v)),
			)
		}
	}
}

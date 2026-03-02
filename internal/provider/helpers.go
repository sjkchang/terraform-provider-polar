// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/polarsource/polar-go/models/apierrors"
)

// --- Eventual consistency infrastructure ---
// Polar's API is eventually consistent. After a write succeeds, subsequent
// reads may still return stale data. We solve this by extracting the write
// timestamp from the API response and polling GET until it catches up.

// Timestamped is satisfied by any Polar API resource that exposes
// created_at / modified_at timestamps.
type Timestamped interface {
	GetCreatedAt() time.Time
	GetModifiedAt() *time.Time
}

// latestTimestamp returns ModifiedAt if present, otherwise CreatedAt.
// Both timestamps come from Polar's servers (no local clock dependency).
func latestTimestamp(t Timestamped) time.Time {
	if mod := t.GetModifiedAt(); mod != nil {
		return *mod
	}
	return t.GetCreatedAt()
}

// Polling constants for eventual consistency handling.
const (
	pollMaxAttempts = 10
	pollInterval    = 500 * time.Millisecond
)

// isNotFound checks if an error is a Polar API 404 ResourceNotFound error.
func isNotFound(err error) bool {
	var notFound *apierrors.ResourceNotFound
	return errors.As(err, &notFound)
}

// isTransient checks if an error is a transient API error (429 rate limit or
// 5xx server error) that the SDK's built-in retry already exhausted. Polling
// should keep trying rather than treating these as permanent failures.
func isTransient(err error) bool {
	var apiErr *apierrors.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 429 || apiErr.StatusCode >= 500
	}
	return false
}

// extractProviderData extracts the *PolarProviderData from provider configuration.
// Returns nil without error when providerData is nil (early provider lifecycle).
func extractProviderData(providerData any, diags *diag.Diagnostics) *PolarProviderData {
	if providerData == nil {
		return nil
	}
	pd, ok := providerData.(*PolarProviderData)
	if !ok {
		diags.AddError(
			"Unexpected Configure Type",
			fmt.Sprintf("Expected *PolarProviderData, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil
	}
	return pd
}

// handleNotFoundRemove checks if err is a 404 ResourceNotFound error and, if so,
// logs it and removes the resource from Terraform state. Returns true if the error
// was a 404 (caller should return early), false otherwise.
//
// Safe against transient infrastructure failures: isNotFound only matches the SDK's
// typed ResourceNotFound (a structured JSON response from Polar's API). Generic
// 404s from CDN/DNS/load balancers return a different error type and won't match,
// so outages won't cause resources to be dropped from state.
func handleNotFoundRemove(ctx context.Context, err error, resourceType, id string, state *tfsdk.State) bool {
	if !isNotFound(err) {
		return false
	}
	tflog.Trace(ctx, resourceType+" not found, removing from state", map[string]interface{}{"id": id})
	state.RemoveResource(ctx)
	return true
}

// --- Nil-safe pointer → Terraform type converters ---

func optionalStringValue(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

// optionalInt64Value safely converts a *int64 to types.Int64,
// returning types.Int64Null() if the pointer is nil.
func optionalInt64Value(i *int64) types.Int64 {
	if i == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*i)
}

// parseOptionalTime parses an RFC3339 timestamp from a types.String, returning
// nil if the value is null or unknown.
func parseOptionalTime(s types.String, diags *diag.Diagnostics) *time.Time {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s.ValueString())
	if err != nil {
		diags.AddError("Invalid timestamp", "Could not parse timestamp: "+err.Error())
		return nil
	}
	return &t
}

// optionalBoolValue safely converts a *bool to types.Bool,
// returning types.BoolNull() if the pointer is nil.
func optionalBoolValue(b *bool) types.Bool {
	if b == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*b)
}

// derefBool safely dereferences a *bool, returning false if nil.
func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// pollForConsistency polls fetch until it returns a result whose timestamp is
// at or after writeTimestamp. Retries on ResourceNotFound, transient errors
// (429/5xx), and stale reads. If polling exhausts all attempts and at least
// one successful read was obtained, the last result is returned with a warning
// diagnostic. If no successful read was ever obtained (e.g. persistent 404),
// a hard error is returned.
func pollForConsistency[T Timestamped](ctx context.Context, resourceType, id string, writeTimestamp time.Time, fetch func() (T, error), diags *diag.Diagnostics) (T, error) {
	var last T
	var hasResult bool
	var lastRejectReason string

	for i := 0; i < pollMaxAttempts; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				var zero T
				return zero, ctx.Err()
			case <-time.After(pollInterval):
			}
		}
		result, err := fetch()
		if err != nil {
			if isNotFound(err) {
				lastRejectReason = "resource not found"
				continue
			}
			if isTransient(err) {
				lastRejectReason = fmt.Sprintf("transient error: %s", err)
				continue
			}
			var zero T
			return zero, err
		}
		last = result
		hasResult = true
		if latestTimestamp(result).Before(writeTimestamp) {
			lastRejectReason = "stale read (timestamp before write)"
			continue
		}
		tflog.Trace(ctx, "read-after-write consistent", map[string]interface{}{
			"resource": resourceType,
			"id":       id,
			"polls":    i + 1,
		})
		return result, nil
	}

	// If we never got a successful read, return a hard error.
	if !hasResult {
		var zero T
		msg := fmt.Sprintf("%s %s not readable after %d polls", resourceType, id, pollMaxAttempts)
		if lastRejectReason != "" {
			msg += ": " + lastRejectReason
		}
		return zero, fmt.Errorf("%s", msg)
	}

	// We got at least one read but it was stale — return it with a warning.
	diags.AddWarning(
		"Eventual consistency timeout",
		fmt.Sprintf(
			"%s %s read-back did not converge after %d polls (%s). "+
				"The state may not reflect the latest changes. Run terraform refresh to re-sync.",
			resourceType, id, pollMaxAttempts, lastRejectReason,
		),
	)
	return last, nil
}

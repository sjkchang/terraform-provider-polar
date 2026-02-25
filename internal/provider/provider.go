// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	polargo "github.com/polarsource/polar-go"
	"github.com/polarsource/polar-go/retry"
)

// Compile-time interface conformance check.
var _ provider.Provider = &PolarProvider{}

// PolarProvider defines the provider implementation.
type PolarProvider struct {
	version string // set by goreleaser on release, "dev" locally, "test" in acceptance tests
}

// PolarProviderModel maps the HCL provider block into Go via `tfsdk` struct tags.
type PolarProviderModel struct {
	AccessToken types.String `tfsdk:"access_token"`
	Server      types.String `tfsdk:"server"`
}

// PolarProviderData is passed to every resource/datasource via Configure().
type PolarProviderData struct {
	Client *polargo.Polar

	// Singleton guard: only one polar_organization resource per provider.
	orgOnce sync.Once
	orgID   string
}

// ClaimOrganization enforces that at most one polar_organization resource
// exists per provider. The first call succeeds and records the org ID;
// subsequent calls with a different ID return an error.
func (pd *PolarProviderData) ClaimOrganization(id string) error {
	pd.orgOnce.Do(func() {
		pd.orgID = id
	})
	if pd.orgID != id {
		return fmt.Errorf(
			"only one polar_organization resource is allowed per provider (already managing %s)",
			pd.orgID,
		)
	}
	return nil
}

func (p *PolarProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "polar"
	resp.Version = p.version
}

func (p *PolarProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Polar provider enables Terraform to manage [Polar.sh](https://polar.sh) resources such as products, meters, benefits, and webhook endpoints.",
		Attributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				MarkdownDescription: "Polar organization access token. Can also be set with the `POLAR_ACCESS_TOKEN` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"server": schema.StringAttribute{
				MarkdownDescription: "The Polar environment to use. Must be `production` or `sandbox`. Can also be set with the `POLAR_SERVER` environment variable.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("production", "sandbox"),
				},
			},
		},
	}
}

// Configure is called once by Terraform after schema validation.
// It resolves auth + server settings, builds the SDK client, and stores
// the resulting PolarProviderData so resources/datasources can use it.
func (p *PolarProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Deserialize the provider HCL block into our model struct.
	var data PolarProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve access token: config value takes precedence over env var.
	accessToken := data.AccessToken.ValueString()
	if accessToken == "" {
		accessToken = os.Getenv("POLAR_ACCESS_TOKEN")
	}
	if accessToken == "" {
		resp.Diagnostics.AddError(
			"Missing Polar Access Token",
			"The provider requires a Polar organization access token. "+
				"Set it in the provider configuration or via the POLAR_ACCESS_TOKEN environment variable.",
		)
		return
	}

	opts := []polargo.SDKOption{
		polargo.WithSecurity(accessToken),
	}

	// Resolve server: config value takes precedence over env var.
	serverStr := data.Server.ValueString()
	if serverStr == "" {
		serverStr = os.Getenv("POLAR_SERVER")
	}
	if serverStr == "" {
		resp.Diagnostics.AddError(
			"Missing Polar Server",
			"The provider requires a server environment (`production` or `sandbox`). "+
				"Set it in the provider configuration or via the POLAR_SERVER environment variable.",
		)
		return
	}
	if serverStr != "production" && serverStr != "sandbox" {
		resp.Diagnostics.AddError(
			"Invalid Polar Server",
			"The server must be `production` or `sandbox`, got: "+serverStr,
		)
		return
	}

	server := polargo.ServerSandbox
	if serverStr == "production" {
		server = polargo.ServerProduction
	}
	opts = append(opts, polargo.WithServer(server))

	// Built-in retry for rate limits (429) and server errors (5xx).
	opts = append(opts, polargo.WithRetryConfig(retry.Config{
		Strategy: "backoff",
		Backoff: &retry.BackoffStrategy{
			InitialInterval: 500,
			MaxInterval:     30000,
			Exponent:        1.5,
			MaxElapsedTime:  120000,
		},
		RetryConnectionErrors: false,
	}))

	client := polargo.New(opts...)

	// Package everything into PolarProviderData and hand it to Terraform.
	// Resources receive this via resp.ResourceData, datasources via resp.DataSourceData.
	providerData := &PolarProviderData{
		Client: client,
	}
	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

// Resources returns constructors for all managed resources.
// Terraform calls each constructor to get a fresh resource instance.
func (p *PolarProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewWebhookEndpointResource,
		NewMeterResource,
		NewBenefitResource,
		NewProductResource,
		NewOrganizationResource,
	}
}

// DataSources returns constructors for all read-only data sources.
func (p *PolarProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewMeterDataSource,
		NewBenefitDataSource,
	}
}

// New returns a factory function that Terraform calls to create the provider.
// The version string is injected by goreleaser at build time.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PolarProvider{
			version: version,
		}
	}
}

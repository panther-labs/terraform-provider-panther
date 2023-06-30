// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"terraform-provider-panther/internal/client/panther"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure PantherProvider satisfies various provider interfaces.
var _ provider.Provider = &PantherProvider{}

// PantherProvider defines the provider implementation.
type PantherProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// PantherProviderModel describes the provider data model.
type PantherProviderModel struct {
	Url   types.String `tfsdk:"url"`
	Token types.String `tfsdk:"token"`
}

func (p *PantherProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "panther"
	resp.Version = p.version
}

func (p *PantherProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Description: "The GraphQL API URL for the target Panther instance.",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "The API token for the Panther GraphQL API.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *PantherProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data PantherProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Url.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("url"),
			"API URL Invalid",
			"The Panther GraphQL API URL is invalid.",
		)
	}

	if data.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("url"),
			"API Token Invalid",
			"The API Token for Panther GraphQL API is invalid.",
		)
	}

	url := os.Getenv("PANTHER_API_URL")
	token := os.Getenv("PANTHER_API_TOKEN")

	if !data.Url.IsNull() {
		url = data.Url.ValueString()
	}

	if !data.Token.IsNull() {
		token = data.Token.ValueString()
	}

	if url == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("url"),
			"Missing Panther API URL",
			"Panther GraphQL API URL must be provided.",
		)
	}

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing Panther API Token",
			"Panther GraphQL API Token must be provided.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.ResourceData = panther.NewClient(url, token)

}

func (p *PantherProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewS3SourceResource,
	}
}

func (p *PantherProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PantherProvider{
			version: version,
		}
	}
}

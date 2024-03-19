// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	dtrack "github.com/futurice/dependency-track-client-go"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &DependencyTrackProvider{}
var _ provider.ProviderWithFunctions = &DependencyTrackProvider{}

// DependencyTrackProvider defines the provider implementation.
type DependencyTrackProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// DependencyTrackProviderModel describes the provider data model.
type DependencyTrackProviderModel struct {
	Host   types.String `tfsdk:"host"`
	APIKey types.String `tfsdk:"api_key"`
}

func (p *DependencyTrackProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "dependencytrack"
	resp.Version = p.version
}

func (p *DependencyTrackProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Required: true,
			},
			"api_key": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
			},
		},
	}
}

func (p *DependencyTrackProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data DependencyTrackProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	if data.Host.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("host"),
			"Missing Dependency Track API Host",
			"The provider cannot create the Dependency Track API client as there is a missing or empty value for the Dependency Track API host.",
		)
	}

	if data.APIKey.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"),
			"Missing Dependency Track API Key",
			"The provider cannot create the Dependency Track API client as there is a missing or empty value for the Dependency Track API key.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := dtrack.NewClient(data.Host.ValueString(),
		dtrack.WithAPIKey(data.APIKey.ValueString()),
		dtrack.WithDebug(true),
	)
	if err != nil {
		resp.Diagnostics.AddError("TODO: Client creation error", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *DependencyTrackProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewTeamResource,
		NewTeamPermissionResource,
		NewProjectResource,
		NewACLMappingResource,
	}
}

func (p *DependencyTrackProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewTeamDataSource,
	}
}

func (p *DependencyTrackProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &DependencyTrackProvider{
			version: version,
		}
	}
}

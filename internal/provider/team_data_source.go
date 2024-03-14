// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &TeamDataSource{}

func NewTeamDataSource() datasource.DataSource {
	return &TeamDataSource{}
}

// TeamDataSource defines the data source implementation.
type TeamDataSource struct {
	client *dtrack.Client
}

// TeamDataSourceModel describes the data source data model.
type TeamDataSourceModel struct {
	ID               types.String   `tfsdk:"id"`
	Name             types.String   `tfsdk:"name"`
	Permissions      []types.String `tfsdk:"permissions"`
	MappedOIDCGroups []types.String `tfsdk:"mapped_oidc_groups"`
}

func (d *TeamDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (d *TeamDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "TODO data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Team UUID",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the team",
				Computed:            true,
			},
			"permissions": schema.ListAttribute{
				ElementType:         basetypes.StringType{},
				MarkdownDescription: "Permissions given to the team",
				Computed:            true,
			},
			"mapped_oidc_groups": schema.ListAttribute{
				ElementType:         basetypes.StringType{},
				MarkdownDescription: "OIDC groups mapped to the team",
				Computed:            true,
			},
		},
	}
}

func (d *TeamDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*dtrack.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *dtrack.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *TeamDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TeamDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	teamID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("id"),
			"Invalid team ID",
			"Team ID has to be a valid UUID.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	team, err := d.client.Team.Get(ctx, teamID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read team, got error: %s", err))
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("read team: %+v", team))

	data.Name = types.StringValue(team.Name)

	data.Permissions = make([]types.String, len(team.Permissions))
	for i, permission := range team.Permissions {
		data.Permissions[i] = types.StringValue(permission.Name)
	}

	data.MappedOIDCGroups = make([]types.String, len(team.MappedOIDCGroups))
	for i, group := range team.MappedOIDCGroups {
		data.MappedOIDCGroups[i] = types.StringValue(group.UUID.String())
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "read a data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

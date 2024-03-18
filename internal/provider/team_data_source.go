// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/google/uuid"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Permissions      types.Set    `tfsdk:"permissions"`
	MappedOIDCGroups types.Set    `tfsdk:"mapped_oidc_groups"`
}

func (d *TeamDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (d *TeamDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
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
			"permissions": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Permissions given to the team",
				Computed:            true,
			},
			"mapped_oidc_groups": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "OIDC groups mapped to the team",
				Computed:            true,
			},
		},
	}
}

func (d *TeamDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	var model TeamDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)

	teamID, err := uuid.Parse(model.ID.ValueString())
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

	model.Name = types.StringValue(team.Name)

	tfPermissions := make([]string, len(team.Permissions))
	for i, p := range team.Permissions {
		tfPermissions[i] = p.Name
	}
	model.Permissions, _ = types.SetValueFrom(ctx, types.StringType, tfPermissions)

	tfOIDCGroups := make([]string, len(team.MappedOIDCGroups))
	for i, p := range team.MappedOIDCGroups {
		tfOIDCGroups[i] = p.UUID.String()
	}
	model.MappedOIDCGroups, _ = types.SetValueFrom(ctx, types.StringType, tfOIDCGroups)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package teampermission

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"strings"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/futurice/terraform-provider-dependencytrack/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TeamPermissionResource{}
var _ resource.ResourceWithImportState = &TeamPermissionResource{}

func NewTeamPermissionResource() resource.Resource {
	return &TeamPermissionResource{}
}

// TeamPermissionResource defines the resource implementation.
type TeamPermissionResource struct {
	client *dtrack.Client
}

// TeamPermissionResourceModel describes the resource data model.
type TeamPermissionResourceModel struct {
	ID     types.String `tfsdk:"id"`
	TeamID types.String `tfsdk:"team_id"`
	Name   types.String `tfsdk:"name"`
}

func (r *TeamPermissionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_permission"
}

func (r *TeamPermissionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Team permission",

		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				MarkdownDescription: "ID of the team",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the permission",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic permission ID in the form of team_id/permission_name",
				Computed:            true,
			},
		},
	}
}

func (r *TeamPermissionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*dtrack.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *dtrack.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *TeamPermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, state TeamPermissionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID, teamIDDiags := utils.ParseAttributeUUID(plan.TeamID.ValueString(), "team_id")
	resp.Diagnostics.Append(teamIDDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	permission := dtrack.Permission{
		Name: plan.Name.ValueString(),
	}

	_, err := r.client.Permission.AddPermissionToTeam(ctx, permission, teamID)
	if err != nil {
		if apiErr, ok := err.(*dtrack.APIError); ok {
			switch apiErr.StatusCode {
			case 304:
				resp.Diagnostics.AddError("Client Error", "The permission already existed on the team")
			case 404:
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("The permission '%s' not found", permission.Name))
			}
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create permission, got error: %s", err))
		}
		return
	}

	state.ID = types.StringValue(makePermissionID(teamID, permission.Name))
	state.TeamID = types.StringValue(teamID.String())
	state.Name = types.StringValue(permission.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamPermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TeamPermissionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID, teamIDDiags := utils.ParseAttributeUUID(state.TeamID.ValueString(), "team_id")
	resp.Diagnostics.Append(teamIDDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	respTeam, err := r.client.Team.Get(ctx, teamID)
	if err != nil {
		if apiErr, ok := err.(*dtrack.APIError); ok && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read team, got error: %s", err))
		return
	}

	found := false
	for _, perm := range respTeam.Permissions {
		if perm.Name == state.Name.ValueString() {
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamPermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TeamPermissionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	newTeamID, newTeamIDDiags := utils.ParseAttributeUUID(plan.TeamID.ValueString(), "team_id")
	resp.Diagnostics.Append(newTeamIDDiags...)

	oldTeamID, oldTeamIDDiags := utils.ParseAttributeUUID(state.TeamID.ValueString(), "team_id")
	resp.Diagnostics.Append(oldTeamIDDiags...)

	if resp.Diagnostics.HasError() {
		return
	}

	newPermission := dtrack.Permission{
		Name: plan.Name.ValueString(),
	}

	_, err := r.client.Permission.AddPermissionToTeam(ctx, newPermission, newTeamID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create new permission, got error: %s", err))
		return
	}

	oldPermission := dtrack.Permission{
		Name: state.Name.ValueString(),
	}

	_, err = r.client.Permission.RemovePermissionFromTeam(ctx, oldPermission, oldTeamID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete old permission, got error: %s", err))
		return
	}

	state.ID = types.StringValue(makePermissionID(newTeamID, newPermission.Name))
	state.TeamID = types.StringValue(newTeamID.String())
	state.Name = types.StringValue(newPermission.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamPermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TeamPermissionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID, teamIDDiags := utils.ParseAttributeUUID(state.TeamID.ValueString(), "team_id")
	resp.Diagnostics.Append(teamIDDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	permission := dtrack.Permission{
		Name: state.Name.ValueString(),
	}

	_, err := r.client.Permission.RemovePermissionFromTeam(ctx, permission, teamID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete permission, got error: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *TeamPermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected ID in the format 'team_id/permission_name', got [%s]", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}

func makePermissionID(teamID uuid.UUID, permission string) string {
	return fmt.Sprintf("%s/%s", teamID.String(), permission)
}

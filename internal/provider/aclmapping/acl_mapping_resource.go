// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package aclmapping

import (
	"context"
	"fmt"
	"strings"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/futurice/terraform-provider-dependencytrack/internal/utils"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ACLMappingResource{}
var _ resource.ResourceWithImportState = &ACLMappingResource{}

func NewACLMappingResource() resource.Resource {
	return &ACLMappingResource{}
}

// ACLMappingResource defines the resource implementation.
type ACLMappingResource struct {
	client *dtrack.Client
}

// ACLResourceModel describes the resource data model.
type ACLResourceModel struct {
	ID        types.String `tfsdk:"id"`
	TeamID    types.String `tfsdk:"team_id"`
	ProjectID types.String `tfsdk:"project_id"`
}

func (r *ACLMappingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acl_mapping"
}

func (r *ACLMappingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "ACL mapping",

		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				MarkdownDescription: "Team UUID",
				Required:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Project UUID",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic ACL mapping ID in the form of team_id/project_id",
				Computed:            true,
			},
		},
	}
}

func (r *ACLMappingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ACLMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, state ACLResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID, teamIDDiags := utils.ParseAttributeUUID(plan.TeamID.ValueString(), "team_id")
	resp.Diagnostics.Append(teamIDDiags...)

	projectID, projectIDDiags := utils.ParseAttributeUUID(plan.ProjectID.ValueString(), "project_id")
	resp.Diagnostics.Append(projectIDDiags...)

	if resp.Diagnostics.HasError() {
		return
	}

	mapping := dtrack.ACLMappingRequest{
		Team:    teamID,
		Project: projectID,
	}

	err := r.client.ACL.AddProjectMapping(ctx, mapping)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create ACL mapping, got error: %s", err))
		return
	}

	state.ID = types.StringValue(makeACLMappingID(teamID, projectID))
	state.TeamID = types.StringValue(teamID.String())
	state.ProjectID = types.StringValue(projectID.String())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ACLMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ACLResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID, teamIDDiags := utils.ParseAttributeUUID(state.TeamID.ValueString(), "team_id")
	resp.Diagnostics.Append(teamIDDiags...)

	projectID, projectIDDiags := utils.ParseAttributeUUID(state.ProjectID.ValueString(), "project_id")
	resp.Diagnostics.Append(projectIDDiags...)

	if resp.Diagnostics.HasError() {
		return
	}

	projects, err := r.client.ACL.GetAllProjects(ctx, teamID, dtrack.PageOptions{})
	if err != nil {
		if apiErr, ok := err.(*dtrack.APIError); ok && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read ACL mapping, got error: %s", err))
		return
	}

	found := false
	for _, project := range projects.Items {
		if project.UUID == projectID {
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

func (r *ACLMappingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ACLResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	oldTeamID, oldTeamIDDiags := utils.ParseAttributeUUID(state.TeamID.ValueString(), "team_id")
	resp.Diagnostics.Append(oldTeamIDDiags...)

	oldProjectID, oldProjectIDDiags := utils.ParseAttributeUUID(state.ProjectID.ValueString(), "project_id")
	resp.Diagnostics.Append(oldProjectIDDiags...)

	newTeamID, newTeamIDDiags := utils.ParseAttributeUUID(plan.TeamID.ValueString(), "team_id")
	resp.Diagnostics.Append(newTeamIDDiags...)

	newProjectID, newProjectIDDiags := utils.ParseAttributeUUID(plan.ProjectID.ValueString(), "project_id")
	resp.Diagnostics.Append(newProjectIDDiags...)

	if resp.Diagnostics.HasError() {
		return
	}

	newMapping := dtrack.ACLMappingRequest{
		Team:    newTeamID,
		Project: newProjectID,
	}

	err := r.client.ACL.AddProjectMapping(ctx, newMapping)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create new ACL mapping, got error: %s", err))
		return
	}

	err = r.client.ACL.RemoveProjectMapping(ctx, oldTeamID, oldProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete old ACL mapping, got error: %s", err))
		return
	}

	state.ID = types.StringValue(makeACLMappingID(newTeamID, newProjectID))
	state.TeamID = types.StringValue(newTeamID.String())
	state.ProjectID = types.StringValue(newProjectID.String())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ACLMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ACLResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID, teamIDDiags := utils.ParseAttributeUUID(state.TeamID.ValueString(), "team_id")
	resp.Diagnostics.Append(teamIDDiags...)

	projectID, projectIDDiags := utils.ParseAttributeUUID(state.ProjectID.ValueString(), "project_id")
	resp.Diagnostics.Append(projectIDDiags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.ACL.RemoveProjectMapping(ctx, teamID, projectID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete ACL mapping, got error: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *ACLMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected ID in the format 'team_id/project_id', got [%s]", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[1])...)
}

func makeACLMappingID(teamID uuid.UUID, projectID uuid.UUID) string {
	return fmt.Sprintf("%s/%s", teamID.String(), projectID.String())
}

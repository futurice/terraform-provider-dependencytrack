// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	dtrack "github.com/futurice/dependency-track-client-go"
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
	var plan ACLResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	mapping := dtrack.ACLMapping{
		Team:    uuid.MustParse(plan.TeamID.ValueString()),
		Project: uuid.MustParse(plan.ProjectID.ValueString()),
	}

	err := r.client.ACLMapping.Create(ctx, mapping)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create ACL mapping, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ACLMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ACLResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	projectMappings, err := r.client.ACLMapping.Get(ctx, uuid.MustParse(state.TeamID.ValueString()))
	if err != nil {
		if apiErr, ok := err.(*dtrack.APIError); ok && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read ACL mapping, got error: %s", err))
		return
	}

	projectID := uuid.MustParse(state.ProjectID.ValueString())
	found := false
	for _, project := range projectMappings {
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

	newMapping := dtrack.ACLMapping{
		Team:    uuid.MustParse(plan.TeamID.ValueString()),
		Project: uuid.MustParse(plan.ProjectID.ValueString()),
	}

	oldMapping := dtrack.ACLMapping{
		Team:    uuid.MustParse(state.TeamID.ValueString()),
		Project: uuid.MustParse(state.ProjectID.ValueString()),
	}

	err := r.client.ACLMapping.Create(ctx, newMapping)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update ACL mapping, got error: %s", err))
		return
	}

	err = r.client.ACLMapping.Delete(ctx, oldMapping)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update ACL mapping, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ACLMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ACLResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	mapping := dtrack.ACLMapping{
		Team:    uuid.MustParse(state.TeamID.ValueString()),
		Project: uuid.MustParse(state.ProjectID.ValueString()),
	}

	err := r.client.ACLMapping.Delete(ctx, mapping)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete ACL mapping, got error: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *ACLMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

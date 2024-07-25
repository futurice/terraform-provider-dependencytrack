// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package teamapikey

import (
	"context"
	"fmt"
	"strings"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/google/uuid"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TeamAPIKeyResource{}
var _ resource.ResourceWithImportState = &TeamAPIKeyResource{}

func NewTeamAPIKeyResource() resource.Resource {
	return &TeamAPIKeyResource{}
}

// TeamAPIKeyResource defines the resource implementation.
type TeamAPIKeyResource struct {
	client *dtrack.Client
}

// TeamAPIKeyResourceModel describes the resource data model.
type TeamAPIKeyResourceModel struct {
	TeamID types.String `tfsdk:"team_id"`
	Value  types.String `tfsdk:"value"`
}

func (r *TeamAPIKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_api_key"
}

func (r *TeamAPIKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "API Key for a team",

		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				MarkdownDescription: "ID of the team",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Value of the API key",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

func (r *TeamAPIKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TeamAPIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TeamAPIKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiKey, err := r.client.Team.GenerateAPIKey(ctx, uuid.MustParse(plan.TeamID.String()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create API key, got error: %s", err))
		return
	}

	plan.Value = types.StringValue(apiKey)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TeamAPIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TeamAPIKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// NOTE: API only returns the API keys for the team when fetching all the teams
	teams, err := r.client.Team.GetAll(ctx, dtrack.PageOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read team, got error: %s", err))
		return
	}

	found := false
	for _, team := range teams.Items {
		if team.UUID.String() != state.TeamID.ValueString() {
			continue
		}

		for _, key := range team.APIKeys {
			if key.Key == state.Value.ValueString() {
				found = true
				break
			}
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamAPIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Internal Error", "API Key resource is immutable")
}

func (r *TeamAPIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TeamAPIKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Team.DeleteAPIKey(ctx, state.Value.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete API key, got error: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *TeamAPIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected ID in the format 'team_id/api_key', got [%s]", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("value"), parts[1])...)
}

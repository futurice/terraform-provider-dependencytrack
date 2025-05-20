// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package teamapikey

import (
	"context"
	"fmt"
	"github.com/futurice/terraform-provider-dependencytrack/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"strings"

	dtrack "github.com/futurice/dependency-track-client-go"
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
	ID      types.String `tfsdk:"id"`
	TeamID  types.String `tfsdk:"team_id"`
	Value   types.String `tfsdk:"value"`
	Comment types.String `tfsdk:"comment"`
	Legacy  types.Bool   `tfsdk:"legacy"`
}

func (r *TeamAPIKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_api_key"
}

func (r *TeamAPIKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "API Key for a team",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Generated ID of the API key, the public ID returned by the API",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
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
			"legacy": schema.BoolAttribute{
				MarkdownDescription: "Whether the key is legacy or not",
				Computed:            true,
			},
			"comment": schema.StringAttribute{
				MarkdownDescription: "The API key comment",
				Optional:            true,
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

	teamID, teamIDDiags := utils.ParseAttributeUUID(plan.TeamID.ValueString(), "team_id")
	resp.Diagnostics.Append(teamIDDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey, err := r.client.Team.GenerateAPIKey(ctx, teamID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create API key, got error: %s", err))
		return
	}

	if plan.Comment.ValueString() != "" {
		_, err := r.client.Team.UpdateAPIKeyComment(ctx, apiKey.PublicId, plan.Comment.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update API key comment, got error: %s", err))
			return
		}
	}

	plan.ID = types.StringValue(apiKey.PublicId)
	plan.Value = types.StringValue(apiKey.Key)
	plan.Legacy = types.BoolValue(apiKey.Legacy)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TeamAPIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TeamAPIKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiKey, diags = r.getTeamAPIKey(ctx, state.TeamID.ValueString(), state.ID.ValueString(), state.Value.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if apiKey == (dtrack.APIKey{}) {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamAPIKeyResource) getTeamAPIKey(ctx context.Context, teamID string, id string, value string) (dtrack.APIKey, diag.Diagnostics) {
	var diags diag.Diagnostics

	// NOTE: API only returns the API keys for the team when fetching all the teams
	teams, err := r.client.Team.GetAll(ctx, dtrack.PageOptions{})
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to read team, got error: %s", err))
		return dtrack.APIKey{}, diags
	}

	for _, team := range teams.Items {
		if team.UUID.String() != teamID {
			continue
		}

		for _, key := range team.APIKeys {
			if key.Legacy {
				if key.Key == value {
					return key, diags
				}
			} else {
				if key.PublicId == id {
					return key, diags
				}
			}
		}
	}

	return dtrack.APIKey{}, diags
}

func (r *TeamAPIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TeamAPIKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Team.UpdateAPIKeyComment(ctx, state.ID.String(), plan.Comment.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update API key comment, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TeamAPIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TeamAPIKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idOrKey := state.ID.ValueString()
	if state.Legacy.ValueBool() {
		idOrKey = state.Value.ValueString()
	}
	err := r.client.Team.DeleteAPIKey(ctx, idOrKey)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete API key, got error: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *TeamAPIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected ID in the format 'team_id/publicID', got [%s]", req.ID))
		return
	}

	teamID := parts[0]
	publicID := parts[1]

	apiKey, diags := r.getTeamAPIKey(ctx, teamID, publicID, "")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if apiKey == (dtrack.APIKey{}) {
		resp.Diagnostics.AddError("API Key Not Found", fmt.Sprintf("No API key found for team ID [%s] and public ID [%s]", teamID, publicID))
		return
	}

	tfApiKey, diags := DTAPIKeyToTFAPIKey(ctx, apiKey, parts[0])
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &tfApiKey)...)
}

func DTAPIKeyToTFAPIKey(ctx context.Context, dtAPIKey dtrack.APIKey, teamID string) (TeamAPIKeyResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	APIKey := TeamAPIKeyResourceModel{
		ID:      types.StringValue(dtAPIKey.PublicId),
		TeamID:  types.StringValue(teamID),
		Comment: types.StringValue(dtAPIKey.Comment),
		Legacy:  types.BoolValue(dtAPIKey.Legacy),
	}

	return APIKey, diags
}

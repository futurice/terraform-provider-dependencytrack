// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package team

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/futurice/terraform-provider-dependencytrack/internal/utils"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TeamResource{}
var _ resource.ResourceWithImportState = &TeamResource{}

func NewTeamResource() resource.Resource {
	return &TeamResource{}
}

// TeamResource defines the resource implementation.
type TeamResource struct {
	client *dtrack.Client
}

// TeamResourceModel describes the resource data model.
type TeamResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	OIDCGroupIds      types.List   `tfsdk:"oidc_group_ids"`
	OIDCGroupMappings types.Map    `tfsdk:"oidc_group_mappings"` // [groupID] -> mappingID
}

func (r *TeamResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (r *TeamResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Team",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the team",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Team UUID",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"oidc_group_ids": schema.ListAttribute{
				MarkdownDescription: "OIDC group IDs",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"oidc_group_mappings": schema.MapAttribute{
				MarkdownDescription: "Internal map of Group UUIDs to Mapping UUIDs.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *TeamResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TeamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, state TeamResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dtTeam, diags := TFTeamToDTTeam(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createdTeam, err := r.client.Team.Create(ctx, dtTeam)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create team, got error: %s", err))
		return
	}

	emptyState := TeamResourceModel{OIDCGroupMappings: types.MapNull(types.StringType)}
	resp.Diagnostics.Append(r.syncOIDCMappings(ctx, plan, emptyState, createdTeam.UUID)...)

	finalTeam, err := r.client.Team.Get(ctx, createdTeam.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read team after create/sync, got error: %s", err))
		return
	}

	state, diags = DTTeamToTFTeam(ctx, finalTeam)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TeamResourceModel
	var diags diag.Diagnostics

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID, teamIDDiags := utils.ParseAttributeUUID(state.ID.ValueString(), "id")
	resp.Diagnostics.Append(teamIDDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	respTeam, err := r.client.Team.Get(ctx, teamID)
	if err != nil {
		var apiErr *dtrack.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read team, got error: %s", err))
		return
	}

	state, diags = DTTeamToTFTeam(ctx, respTeam)
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TeamResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set plan id here or otherwise the check in TFTeamToDTTeam will always have empty id
	plan.ID = state.ID

	dtTeam, diags := TFTeamToDTTeam(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updatedTeam, err := r.client.Team.Update(ctx, dtTeam)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update team, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(r.syncOIDCMappings(ctx, plan, state, updatedTeam.UUID)...)
	finalTeam, err := r.client.Team.Get(ctx, updatedTeam.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read team after update/sync, got error: %s", err))
		return
	}

	finalState, diags := DTTeamToTFTeam(ctx, finalTeam)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &finalState)...)
}

func (r *TeamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TeamResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete OIDC mappings
	var actualMappings map[string]string
	if !state.OIDCGroupMappings.IsNull() {
		diags := state.OIDCGroupMappings.ElementsAs(ctx, &actualMappings, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for groupID, mappingID := range actualMappings {
			mappingIDUUID, err := uuid.Parse(mappingID)
			if err != nil {
				resp.Diagnostics.AddError("Internal Error", fmt.Sprintf("Invalid mapping UUID %s from state for group %s", mappingID, groupID))
				continue
			}
			err = r.client.OIDC.RemoveTeamMapping(ctx, mappingIDUUID)
			if err != nil {
				// Don't stop; try to delete the team anyway
				resp.Diagnostics.AddWarning("Client Error", fmt.Sprintf("Unable to remove OIDC mapping for group %s: %s. Continuing with team deletion.", groupID, err))
			}
		}
	}

	dtTeam, diags := TFTeamToDTTeam(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Team.Delete(ctx, dtTeam)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete team, got error: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *TeamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func DTTeamToTFTeam(ctx context.Context, dtTeam dtrack.Team) (TeamResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	groupIds := make([]string, 0, len(dtTeam.MappedOIDCGroups))
	mappingMap := make(map[string]string)

	for _, mapping := range dtTeam.MappedOIDCGroups {
		groupID := mapping.Group.UUID.String()
		mappingID := mapping.UUID.String()
		groupIds = append(groupIds, groupID)
		mappingMap[groupID] = mappingID
	}

	oidcGroupIdsList, listDiags := types.ListValueFrom(ctx, types.StringType, groupIds)
	diags.Append(listDiags...)

	oidcGroupMappingsMap, mapDiags := types.MapValueFrom(ctx, types.StringType, mappingMap)
	diags.Append(mapDiags...)

	if diags.HasError() {
		return TeamResourceModel{}, diags
	}

	team := TeamResourceModel{
		ID:                types.StringValue(dtTeam.UUID.String()),
		Name:              types.StringValue(dtTeam.Name),
		OIDCGroupIds:      oidcGroupIdsList,
		OIDCGroupMappings: oidcGroupMappingsMap,
	}

	return team, diags
}

func TFTeamToDTTeam(ctx context.Context, tfTeam TeamResourceModel) (dtrack.Team, diag.Diagnostics) {
	var diags diag.Diagnostics
	team := dtrack.Team{
		Name: tfTeam.Name.ValueString(),
	}

	if tfTeam.ID.ValueString() != "" {
		teamID, teamIDDiags := utils.ParseAttributeUUID(tfTeam.ID.ValueString(), "id")
		team.UUID = teamID
		diags.Append(teamIDDiags...)
	}

	return team, diags
}

// Convert types.List in plan to a better searchable map.
func (r *TeamResource) formatDesiredGroups(ctx context.Context, plan TeamResourceModel) (map[string]bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	var desiredGroupIDs []types.String
	if !plan.OIDCGroupIds.IsNull() {
		diags.Append(plan.OIDCGroupIds.ElementsAs(ctx, &desiredGroupIDs, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}
	desiredSet := make(map[string]bool, len(desiredGroupIDs))
	for _, id := range desiredGroupIDs {
		if id.IsUnknown() || id.IsNull() {
			continue
		}
		desiredSet[id.ValueString()] = true
	}
	return desiredSet, diags
}

// Convert types.Map in state to map.
func (r *TeamResource) formatExistingMappings(ctx context.Context, state TeamResourceModel) (map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	var frameworkMappings map[string]types.String
	if !state.OIDCGroupMappings.IsNull() {
		diags.Append(state.OIDCGroupMappings.ElementsAs(ctx, &frameworkMappings, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}
	actualMappings := make(map[string]string, len(frameworkMappings))
	for groupID, mappingID := range frameworkMappings {
		if mappingID.IsUnknown() || mappingID.IsNull() {
			continue
		}
		actualMappings[groupID] = mappingID.ValueString()
	}
	return actualMappings, diags
}

// Add groups that are in plan but not state.
func (r *TeamResource) addDesiredGroups(ctx context.Context, desiredSet map[string]bool, actualMappings map[string]string, teamUUID uuid.UUID) diag.Diagnostics {
	var diags diag.Diagnostics
	for groupID := range desiredSet {
		if _, ok := actualMappings[groupID]; !ok {
			groupUUID, err := uuid.Parse(groupID)
			if err != nil {
				diags.AddAttributeError(path.Root("oidc_group_ids"), "Invalid UUID", "Invalid group UUID: "+groupID)
				continue
			}

			req := dtrack.OIDCMappingRequest{
				Team:  teamUUID,
				Group: groupUUID,
			}
			_, err = r.client.OIDC.AddTeamMapping(ctx, req)
			if err != nil {
				diags.AddError("Client Error", fmt.Sprintf("Unable to add OIDC mapping for group %s: %s", groupID, err))
			}
		}
	}

	return diags
}

// Remove groups that are in state but not plan.
func (r *TeamResource) removeExtraGroups(ctx context.Context, desiredSet map[string]bool, actualMappings map[string]string) diag.Diagnostics {
	var diags diag.Diagnostics
	for groupID, mappingID := range actualMappings {
		if !desiredSet[groupID] {
			mappingUUID, err := uuid.Parse(mappingID)
			if err != nil {
				diags.AddError("Internal Error", fmt.Sprintf("Invalid mapping UUID %s from state for group %s", mappingID, groupID))
				continue
			}

			err = r.client.OIDC.RemoveTeamMapping(ctx, mappingUUID)
			if err != nil {
				diags.AddError("Client Error", fmt.Sprintf("Unable to remove OIDC mapping for group %s: %s", groupID, err))
			}
		}
	}
	return diags
}

// syncOIDCMappings adds / removes mappings based on plan and state.
func (r *TeamResource) syncOIDCMappings(ctx context.Context, plan TeamResourceModel, state TeamResourceModel, teamUUID uuid.UUID) diag.Diagnostics {
	var diags diag.Diagnostics

	desiredSet, d := r.formatDesiredGroups(ctx, plan)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	actualMappings, d := r.formatExistingMappings(ctx, state)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	d = r.addDesiredGroups(ctx, desiredSet, actualMappings, teamUUID)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	d = r.removeExtraGroups(ctx, desiredSet, actualMappings)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	return diags
}

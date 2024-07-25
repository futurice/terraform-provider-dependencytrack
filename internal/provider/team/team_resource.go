package team

import (
	"context"
	"fmt"
	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/futurice/terraform-provider-dependencytrack/internal/utils"
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
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
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

	respTeam, err := r.client.Team.Create(ctx, dtTeam)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create API key, got error: %s", err))
		return
	}

	state, diags = DTTeamToTFTeam(ctx, respTeam)
	resp.Diagnostics.Append(diags...)

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
		if apiErr, ok := err.(*dtrack.APIError); ok && apiErr.StatusCode == 404 {
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

	dtTeam, diags := TFTeamToDTTeam(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	respTeam, err := r.client.Team.Update(ctx, dtTeam)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update team, got error: %s", err))
		return
	}

	state, diags = DTTeamToTFTeam(ctx, respTeam)
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TeamResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
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
	team := TeamResourceModel{
		ID:   types.StringValue(dtTeam.UUID.String()),
		Name: types.StringValue(dtTeam.Name),
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

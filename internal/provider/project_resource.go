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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ProjectResource{}
var _ resource.ResourceWithImportState = &ProjectResource{}

func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

// ProjectResource defines the resource implementation.
type ProjectResource struct {
	client *dtrack.Client
}

// ProjectResourceModel describes the resource data model.
type ProjectResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ParentID    types.String `tfsdk:"parent_id"`
	Name        types.String `tfsdk:"name"`
	Classifier  types.String `tfsdk:"classifier"`
	Description types.String `tfsdk:"description"`
	Active      types.Bool   `tfsdk:"active"`
	// Author             types.String `tfsdk:"author"`
	// Publisher          types.String `tfsdk:"publisher"`
	// Group              types.String `tfsdk:"group"`
	// Version            types.String `tfsdk:"version"`
	// CPE                types.String `tfsdk:"cpe"`
	// PURL               types.String `tfsdk:"purl"`
	// SWIDTagID          types.String `tfsdk:"swidTagId"`
	// DirectDependencies types.String `tfsdk:"directDependencies"`
	// Metrics            ProjectMetrics    `tfsdk:"metrics"`
	// Properties         []ProjectProperty `tfsdk:"properties"`
	// Tags               []Tag             `tfsdk:"tags"`
}

func (r *ProjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Team",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the team",
				Required:            true,
			},
			"classifier": schema.StringAttribute{
				MarkdownDescription: "Specifies the type of project. Must be one of the following values: [APPLICATION, CONTAINER, PLATFORM, DEVICE, DATA, FIRMWARE, FILE, OPERATING_SYSTEM, FRAMEWORK, MACHINE_LEARNING_MODEL, LIBRARY, DEVICE_DRIVER]",
				Required:            true,
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Whether the project is active or not. Default is true.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"parent_id": schema.StringAttribute{
				MarkdownDescription: "Parent project UUID",
				Optional:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Name of the team",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Project UUID",
				Computed:            true,
			},
		},
	}
}

func (r *ProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project := dtrack.Project{
		Name:       plan.Name.ValueString(),
		Classifier: plan.Classifier.ValueString(),
		Active:     plan.Active.ValueBool(),
	}

	if !plan.ParentID.IsNull() {
		project.ParentRef = &dtrack.ParentRef{UUID: uuid.MustParse(plan.ParentID.ValueString())}
	}

	respProject, err := r.client.Project.Create(ctx, project)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create project, got error: %s", err))
		return
	}

	plan.ID = types.StringValue(respProject.UUID.String())
	plan.Name = types.StringValue(respProject.Name)
	plan.Classifier = types.StringValue(respProject.Classifier)
	plan.Active = types.BoolValue(respProject.Active)

	if respProject.Description != "" {
		plan.Description = types.StringValue(respProject.Description)
	} else {
		plan.Description = types.StringNull()
	}

	if respProject.ParentRef != nil {
		plan.ParentID = types.StringValue(respProject.ParentRef.UUID.String())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	respProject, err := r.client.Project.Get(ctx, uuid.MustParse(state.ID.ValueString()))
	if err != nil {
		if apiErr, ok := err.(*dtrack.APIError); ok && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
		return
	}

	state.ID = types.StringValue(respProject.UUID.String())
	state.Name = types.StringValue(respProject.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ProjectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project := dtrack.Project{
		UUID:       uuid.MustParse(state.ID.ValueString()),
		Name:       plan.Name.ValueString(),
		Classifier: plan.Classifier.ValueString(),
		Active:     plan.Active.ValueBool(),
	}

	if !plan.ParentID.IsNull() {
		project.ParentRef = &dtrack.ParentRef{UUID: uuid.MustParse(plan.ParentID.ValueString())}
	}

	respProject, err := r.client.Project.Update(ctx, project)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update team, got error: %s", err))
		return
	}

	state.ID = types.StringValue(respProject.UUID.String())
	state.Name = types.StringValue(respProject.Name)
	state.Classifier = types.StringValue(respProject.Classifier)
	state.Active = types.BoolValue(respProject.Active)

	if respProject.Description != "" {
		state.Description = types.StringValue(respProject.Description)
	} else {
		state.Description = types.StringNull()
	}

	// API does not return parent ID when updating, so we assume it was updated
	state.ParentID = plan.ParentID

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Project.Delete(ctx, uuid.MustParse(state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete team, got error: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

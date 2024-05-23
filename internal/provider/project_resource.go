// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/google/uuid"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ProjectResource{}
var _ resource.ResourceWithConfigure = &ProjectResource{}
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
		MarkdownDescription: "Project",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the project",
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
				MarkdownDescription: "Description of the project",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Project UUID",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
	var plan, state ProjectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	dtProject, diags := TFProjectToDTProject(ctx, plan)
	resp.Diagnostics.Append(diags...)

	respProject, err := r.client.Project.Create(ctx, dtProject)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create project, got error: %s", err))
		return
	}

	state, diags = DTProjectToTFProject(ctx, respProject)
	resp.Diagnostics.Append(diags...)

	// API does not return parent ID when updating, so we assume it was set as requested
	state.ParentID = plan.ParentID

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectResourceModel
	var diags diag.Diagnostics

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

	state, diags = DTProjectToTFProject(ctx, respProject)
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ProjectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	dtProject, diags := TFProjectToDTProject(ctx, plan)
	resp.Diagnostics.Append(diags...)

	respProject, err := r.client.Project.Update(ctx, dtProject)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update project, got error: %s", err))
		return
	}

	state, diags = DTProjectToTFProject(ctx, respProject)
	resp.Diagnostics.Append(diags...)

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
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete project, got error: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func DTProjectToTFProject(ctx context.Context, dtProject dtrack.Project) (ProjectResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	project := ProjectResourceModel{
		ID:         types.StringValue(dtProject.UUID.String()),
		Name:       types.StringValue(dtProject.Name),
		Classifier: types.StringValue(dtProject.Classifier),
		Active:     types.BoolValue(dtProject.Active),
	}

	if dtProject.ParentRef != nil {
		project.ParentID = types.StringValue(dtProject.ParentRef.UUID.String())
	} else {
		project.ParentID = types.StringNull()
	}

	if dtProject.Description != "" {
		project.Description = types.StringValue(dtProject.Description)
	} else {
		project.Description = types.StringNull()
	}

	return project, diags
}

func TFProjectToDTProject(ctx context.Context, tfProject ProjectResourceModel) (dtrack.Project, diag.Diagnostics) {
	var diags diag.Diagnostics
	project := dtrack.Project{
		Name:        tfProject.Name.ValueString(),
		Classifier:  tfProject.Classifier.ValueString(),
		Active:      tfProject.Active.ValueBool(),
		Description: tfProject.Description.ValueString(),
	}

	if tfProject.ID.ValueString() != "" {
		project.UUID = uuid.MustParse(tfProject.ID.ValueString())
	}

	if !tfProject.ParentID.IsNull() {
		project.ParentRef = &dtrack.ParentRef{UUID: uuid.MustParse(tfProject.ParentID.ValueString())}
	}

	return project, diags
}

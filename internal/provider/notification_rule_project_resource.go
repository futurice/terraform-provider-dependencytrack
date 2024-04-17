// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

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
var _ resource.Resource = &NotificationRuleProjectResource{}
var _ resource.ResourceWithImportState = &NotificationRuleProjectResource{}

func NewNotificationRuleProjectResource() resource.Resource {
	return &NotificationRuleProjectResource{}
}

// NotificationRuleProjectResource defines the resource implementation.
type NotificationRuleProjectResource struct {
	client *dtrack.Client
}

// NotificationRuleProjectResourceModel describes the resource data model.
type NotificationRuleProjectResourceModel struct {
	ProjectID types.String `tfsdk:"project_id"`
	RuleID    types.String `tfsdk:"rule_id"`
}

func (r *NotificationRuleProjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_rule_project"
}

func (r *NotificationRuleProjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Notification rule project",

		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				MarkdownDescription: "ID of the project",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rule_id": schema.StringAttribute{
				MarkdownDescription: "ID of the notification rule",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *NotificationRuleProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NotificationRuleProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NotificationRuleProjectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Notification.AddProjectToRule(ctx, uuid.MustParse(plan.RuleID.String()), uuid.MustParse(plan.ProjectID.String()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create API key, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NotificationRuleProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NotificationRuleProjectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// There is no API mehtod for a single rule, so we need to get all rules and filter
	rules, err := r.client.Notification.GetAllRules(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read team, got error: %s", err))
		return
	}

	found := false
	for _, rule := range rules {
		if rule.UUID.String() != state.RuleID.ValueString() {
			continue
		}

		for _, project := range rule.Projects {
			if project.UUID.String() == state.ProjectID.ValueString() {
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

func (r *NotificationRuleProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Internal Error", "Notification rule project relation resource is immutable")
}

func (r *NotificationRuleProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NotificationRuleProjectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Notification.DeleteProjectFromRule(ctx, uuid.MustParse(state.RuleID.ValueString()), uuid.MustParse(state.ProjectID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete notification rule project relation, got error: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *NotificationRuleProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected ID in the format 'project_id/rule_id'")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("rule_id"), parts[1])...)
}

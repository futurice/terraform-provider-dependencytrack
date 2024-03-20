// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/google/uuid"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &NotificationRuleResource{}
var _ resource.ResourceWithImportState = &NotificationRuleResource{}

func NewNotificationRuleResource() resource.Resource {
	return &NotificationRuleResource{}
}

// NotificationRuleResource defines the resource implementation.
type NotificationRuleResource struct {
	client *dtrack.Client
}

// NotificationRuleResourceModel describes the resource data model.
type NotificationRuleResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	NotificationLevel    types.String `tfsdk:"notification_level"`
	PublisherID          types.String `tfsdk:"publisher_id"`
	Scope                types.String `tfsdk:"scope"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	LogSuccessfulPublish types.Bool   `tfsdk:"log_successful_publish"`
	NotifyChildren       types.Bool   `tfsdk:"notify_children"`
	NotifyOn             types.Set    `tfsdk:"notify_on"`
	PublisherConfig      types.String `tfsdk:"publisher_config"`
}

func (r *NotificationRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_rule"
}

func (r *NotificationRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Notification rule",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the rule",
				Required:            true,
			},
			"publisher_id": schema.StringAttribute{
				MarkdownDescription: "Publisher UUID",
				Required:            true,
			},
			"scope": schema.StringAttribute{
				MarkdownDescription: "Rule scope. Possible values: [PORTFOLIO, SYSTEM]",
				Required:            true,
			},
			"notification_level": schema.StringAttribute{
				MarkdownDescription: "Notification level. Possible values: [INFORMATIONAL, WARNING, ERROR]",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Rule UUID",
				Computed:            true,
			},
			"enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"notify_children": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"notify_on": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"log_successful_publish": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"publisher_config": schema.StringAttribute{
				MarkdownDescription: "Publisher configuration in JSON format",
				Optional:            true,
			},
		},
	}
}

func (r *NotificationRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NotificationRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NotificationRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	dtRule, diags := TFRuleToDTRule(ctx, plan)
	resp.Diagnostics.Append(diags...)

	respRule, err := r.client.Notification.CreateRule(ctx, dtRule)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create notification rule, got error: %s", err))
		return
	}

	// Some attributes can not be set on creation
	if dtRule.PublisherConfig != "" {
		dtRule.UUID = respRule.UUID
		respRule, err = r.client.Notification.UpdateRule(ctx, dtRule)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update notification rule, got error: %s", err))
			return
		}
	}

	plan, diags = DTRuleToTFRule(ctx, respRule)
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NotificationRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NotificationRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	rules, err := r.client.Notification.GetAllRules(ctx)
	if err != nil {
		if apiErr, ok := err.(*dtrack.APIError); ok && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read notification rule, got error: %s", err))
		return
	}

	found := false
	for _, rule := range rules {
		if rule.UUID.String() == state.ID.ValueString() {
			found = true
			newState, diags := DTRuleToTFRule(ctx, rule)
			resp.Diagnostics.Append(diags...)
			state = newState
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NotificationRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state NotificationRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	plan.ID = state.ID

	if resp.Diagnostics.HasError() {
		return
	}

	dtRule, diags := TFRuleToDTRule(ctx, plan)
	resp.Diagnostics.Append(diags...)

	respRule, err := r.client.Notification.UpdateRule(ctx, dtRule)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update notification rule, got error: %s", err))
		return
	}

	state, diags = DTRuleToTFRule(ctx, respRule)
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NotificationRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NotificationRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Notification.DeleteRule(ctx, uuid.MustParse(state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete notification rule, got error: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *NotificationRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func DTRuleToTFRule(ctx context.Context, dtRule dtrack.NotificationRule) (NotificationRuleResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	rule := NotificationRuleResourceModel{
		ID:                   types.StringValue(dtRule.UUID.String()),
		Name:                 types.StringValue(dtRule.Name),
		PublisherID:          types.StringValue(dtRule.Publisher.UUID.String()),
		Scope:                types.StringValue(dtRule.Scope),
		NotificationLevel:    types.StringValue(dtRule.NotificationLevel),
		Enabled:              types.BoolValue(dtRule.Enabled),
		LogSuccessfulPublish: types.BoolValue(dtRule.LogSuccessfulPublish),
		NotifyChildren:       types.BoolValue(dtRule.NotifyChildren),
		PublisherConfig:      types.StringValue(dtRule.PublisherConfig),
	}

	rule.NotifyOn, diags = types.SetValueFrom(ctx, types.StringType, dtRule.NotifyOn)

	return rule, diags
}

func TFRuleToDTRule(ctx context.Context, tfRule NotificationRuleResourceModel) (dtrack.NotificationRule, diag.Diagnostics) {
	rule := dtrack.NotificationRule{
		Name:                 tfRule.Name.ValueString(),
		Publisher:            dtrack.NotificationPublisher{UUID: uuid.MustParse(tfRule.PublisherID.ValueString())},
		Scope:                tfRule.Scope.ValueString(),
		NotificationLevel:    tfRule.NotificationLevel.ValueString(),
		Enabled:              tfRule.Enabled.ValueBool(),
		LogSuccessfulPublish: tfRule.LogSuccessfulPublish.ValueBool(),
		NotifyChildren:       tfRule.NotifyChildren.ValueBool(),
		PublisherConfig:      tfRule.PublisherConfig.ValueString(),
	}

	elements := make([]types.String, 0, len(tfRule.NotifyOn.Elements()))
	diags := tfRule.NotifyOn.ElementsAs(ctx, &elements, false)
	rule.NotifyOn = make([]string, len(elements))
	for i := range elements {
		rule.NotifyOn[i] = elements[i].ValueString()
	}

	if tfRule.ID.IsUnknown() {
		rule.UUID = uuid.Nil
	} else {
		rule.UUID = uuid.MustParse(tfRule.ID.ValueString())
	}

	return rule, diags
}

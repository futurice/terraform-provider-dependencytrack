// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package notificationrule

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/futurice/terraform-provider-dependencytrack/internal/utils"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scope": schema.StringAttribute{
				MarkdownDescription: "Rule scope. Possible values: [PORTFOLIO, SYSTEM]",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"notification_level": schema.StringAttribute{
				MarkdownDescription: "Notification level. Possible values: [INFORMATIONAL, WARNING, ERROR]",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Rule UUID",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				MarkdownDescription: "Publisher configuration in JSON format. Dependency-Track v5 validates it against the publisher's rule config schema and fills in a default when it's omitted.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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

	// The client follows create with an update, since create drops most settings on both v4 and v5.
	respRule, err := r.client.Notification.CreateRule(ctx, dtRule)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create notification rule, got error: %s", err))

		// The rule exists if only the follow-up update failed. Saving it marks it tainted, so the next apply
		// replaces it instead of creating a duplicate.
		if respRule.UUID != uuid.Nil {
			created, createdDiags := DTRuleToTFRule(ctx, respRule)
			resp.Diagnostics.Append(createdDiags...)
			resp.Diagnostics.Append(resp.State.Set(ctx, &created)...)
		}
		return
	}

	newState, diags := DTRuleToTFRule(ctx, respRule)
	resp.Diagnostics.Append(diags...)
	plan = keepEquivalentPublisherConfig(newState, plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NotificationRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NotificationRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// There is no API method for a single rule, and v5 caps unpaged lists at 100.
	rules, err := dtrack.FetchAll(func(po dtrack.PageOptions) (dtrack.Page[dtrack.NotificationRule], error) {
		return r.client.Notification.GetAllRules(ctx, po)
	})
	if err != nil {
		var apiErr *dtrack.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
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
			state = keepEquivalentPublisherConfig(newState, state)
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

	newState, diags := DTRuleToTFRule(ctx, respRule)
	resp.Diagnostics.Append(diags...)
	state = keepEquivalentPublisherConfig(newState, plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NotificationRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NotificationRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ruleID, ruleIDDiags := utils.ParseAttributeUUID(state.ID.ValueString(), "id")
	resp.Diagnostics.Append(ruleIDDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Notification.DeleteRule(ctx, ruleID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete notification rule, got error: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *NotificationRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// keepEquivalentPublisherConfig keeps the prior publisher_config string when the server returns the same JSON reformatted.
func keepEquivalentPublisherConfig(rule, prior NotificationRuleResourceModel) NotificationRuleResourceModel {
	if !prior.PublisherConfig.IsNull() && !prior.PublisherConfig.IsUnknown() && !rule.PublisherConfig.IsNull() &&
		utils.JSONEqual(rule.PublisherConfig.ValueString(), prior.PublisherConfig.ValueString()) {
		rule.PublisherConfig = prior.PublisherConfig
	}
	return rule
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

	// normalize to null to allow the attribute to be optional, and compact so list and update responses match
	if len(dtRule.PublisherConfig) > 0 {
		rule.PublisherConfig = types.StringValue(utils.CompactJSON(dtRule.PublisherConfig))
	} else {
		rule.PublisherConfig = types.StringNull()
	}

	rule.NotifyOn, diags = types.SetValueFrom(ctx, types.StringType, dtRule.NotifyOn)

	return rule, diags
}

func TFRuleToDTRule(ctx context.Context, tfRule NotificationRuleResourceModel) (dtrack.NotificationRule, diag.Diagnostics) {
	var diags diag.Diagnostics

	publisherID, publisherIDDiags := utils.ParseAttributeUUID(tfRule.PublisherID.ValueString(), "publisher_id")
	diags.Append(publisherIDDiags...)

	rule := dtrack.NotificationRule{
		Name:                 tfRule.Name.ValueString(),
		Publisher:            dtrack.NotificationPublisher{UUID: publisherID},
		Scope:                tfRule.Scope.ValueString(),
		NotificationLevel:    tfRule.NotificationLevel.ValueString(),
		Enabled:              tfRule.Enabled.ValueBool(),
		LogSuccessfulPublish: tfRule.LogSuccessfulPublish.ValueBool(),
		NotifyChildren:       tfRule.NotifyChildren.ValueBool(),
		PublisherConfig:      tfRule.PublisherConfig.ValueString(),
	}

	elements := make([]types.String, 0, len(tfRule.NotifyOn.Elements()))
	notifyOnDiags := tfRule.NotifyOn.ElementsAs(ctx, &elements, false)
	diags.Append(notifyOnDiags...)
	if !notifyOnDiags.HasError() {
		rule.NotifyOn = make([]string, len(elements))
		for i := range elements {
			rule.NotifyOn[i] = elements[i].ValueString()
		}
	}

	if tfRule.ID.IsUnknown() {
		rule.UUID = uuid.Nil
	} else {
		ruleID, ruleIDDiags := utils.ParseAttributeUUID(tfRule.ID.ValueString(), "id")
		diags.Append(ruleIDDiags...)

		rule.UUID = ruleID
	}

	return rule, diags
}

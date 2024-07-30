// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package notificationpublisher

import (
	"context"
	"fmt"
	"github.com/futurice/terraform-provider-dependencytrack/internal/utils"

	dtrack "github.com/futurice/dependency-track-client-go"
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
var _ resource.Resource = &NotificationPublisherResource{}
var _ resource.ResourceWithImportState = &NotificationPublisherResource{}

func NewNotificationPublisherResource() resource.Resource {
	return &NotificationPublisherResource{}
}

// NotificationPublisherResource defines the resource implementation.
type NotificationPublisherResource struct {
	client *dtrack.Client
}

// NotificationPublisherResourceModel describes the resource data model.
type NotificationPublisherResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	PublisherClass   types.String `tfsdk:"publisher_class"`
	TemplateMimeType types.String `tfsdk:"template_mime_type"`
	Template         types.String `tfsdk:"template"`
}

func (r *NotificationPublisherResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_publisher"
}

func (r *NotificationPublisherResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Notification publisher",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the publisher",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Publisher UUID",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"template_mime_type": schema.StringAttribute{
				MarkdownDescription: "MIME type of the template",
				Required:            true,
			},
			"template": schema.StringAttribute{
				MarkdownDescription: "Template used by the publisher",
				Required:            true,
			},
			"publisher_class": schema.StringAttribute{
				MarkdownDescription: "Class of the publisher",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the publisher",
				Optional:            true,
			},
		},
	}
}

func (r *NotificationPublisherResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NotificationPublisherResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NotificationPublisherResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dtPublisher, diags := TFPublisherToDTPublisher(ctx, plan)
	resp.Diagnostics.Append(diags...)

	respPublisher, err := r.client.Notification.CreatePublisher(ctx, dtPublisher)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create notification publisher, got error: %s", err))
		return
	}

	plan, diags = DTPublisherToTFPublisher(ctx, respPublisher)
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NotificationPublisherResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NotificationPublisherResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	publishers, err := r.client.Notification.GetAllPublishers(ctx)
	if err != nil {
		if apiErr, ok := err.(*dtrack.APIError); ok && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read notification publisher, got error: %s", err))
		return
	}

	found := false
	for _, publisher := range publishers {
		if publisher.UUID.String() == state.ID.ValueString() {
			found = true
			newState, diags := DTPublisherToTFPublisher(ctx, publisher)
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

func (r *NotificationPublisherResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state NotificationPublisherResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	dtPublisher, diags := TFPublisherToDTPublisher(ctx, plan)
	resp.Diagnostics.Append(diags...)

	respPublisher, err := r.client.Notification.UpdatePublisher(ctx, dtPublisher)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update notification publisher, got error: %s", err))
		return
	}

	state, diags = DTPublisherToTFPublisher(ctx, respPublisher)
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NotificationPublisherResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NotificationPublisherResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	publisherID, publisherIDDiags := utils.ParseAttributeUUID(state.ID.ValueString(), "id")
	resp.Diagnostics.Append(publisherIDDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Notification.DeletePublisher(ctx, publisherID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete notification publisher, got error: %s", err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *NotificationPublisherResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func DTPublisherToTFPublisher(ctx context.Context, dtPublisher dtrack.NotificationPublisher) (NotificationPublisherResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	publisher := NotificationPublisherResourceModel{
		ID:               types.StringValue(dtPublisher.UUID.String()),
		Name:             types.StringValue(dtPublisher.Name),
		PublisherClass:   types.StringValue(dtPublisher.PublisherClass),
		TemplateMimeType: types.StringValue(dtPublisher.TemplateMimeType),
		Template:         types.StringValue(dtPublisher.Template),
	}

	// normalize to null to allow the attribute to be optional
	if len(dtPublisher.Description) > 0 {
		publisher.Description = types.StringValue(dtPublisher.Description)
	} else {
		publisher.Description = types.StringNull()
	}

	return publisher, diags
}

func TFPublisherToDTPublisher(ctx context.Context, tfPublisher NotificationPublisherResourceModel) (dtrack.NotificationPublisher, diag.Diagnostics) {
	var diags diag.Diagnostics

	publisher := dtrack.NotificationPublisher{
		Name:             tfPublisher.Name.ValueString(),
		Description:      tfPublisher.Description.ValueString(),
		PublisherClass:   tfPublisher.PublisherClass.ValueString(),
		TemplateMimeType: tfPublisher.TemplateMimeType.ValueString(),
		Template:         tfPublisher.Template.ValueString(),
	}

	if tfPublisher.ID.IsUnknown() {
		publisher.UUID = uuid.Nil
	} else {
		publisherID, publisherIDDiags := utils.ParseAttributeUUID(tfPublisher.ID.ValueString(), "id")
		diags.Append(publisherIDDiags...)

		publisher.UUID = publisherID
	}

	return publisher, diags
}

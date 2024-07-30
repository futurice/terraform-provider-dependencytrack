// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package notificationpublisher

import (
	"context"
	"fmt"

	dtrack "github.com/futurice/dependency-track-client-go"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &NotificationPublisherDataSource{}

func NewNotificationPublisherDataSource() datasource.DataSource {
	return &NotificationPublisherDataSource{}
}

// NotificationPublisherDataSource defines the data source implementation.
type NotificationPublisherDataSource struct {
	client *dtrack.Client
}

// NotificationPublisherDataSourceModel describes the data source data model.
type NotificationPublisherDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	PublisherClass   types.String `tfsdk:"publisher_class"`
	Template         types.String `tfsdk:"template"`
	TemplateMimeType types.String `tfsdk:"template_mime_type"`
	DefaultPublisher types.Bool   `tfsdk:"default_publisher"`
}

func (d *NotificationPublisherDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_publisher"
}

func (d *NotificationPublisherDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Notification publisher data source",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the publisher",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Publisher UUID",
				Computed:            true,
			},
			"publisher_class": schema.StringAttribute{
				MarkdownDescription: "Class of the publisher",
				Computed:            true,
			},
			"template": schema.StringAttribute{
				MarkdownDescription: "Template used by the publisher",
				Computed:            true,
			},
			"template_mime_type": schema.StringAttribute{
				MarkdownDescription: "MIME type of the template",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the publisher",
				Computed:            true,
			},
			"default_publisher": schema.BoolAttribute{
				MarkdownDescription: "Whether this is a default publisher",
				Computed:            true,
			},
		},
	}
}

func (d *NotificationPublisherDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*dtrack.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *dtrack.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *NotificationPublisherDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state NotificationPublisherDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	publishers, err := d.client.Notification.GetAllPublishers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read team, got error: %s", err))
		return
	}

	var found bool
	for _, publisher := range publishers {
		if publisher.Name == state.Name.ValueString() {
			state.ID = types.StringValue(publisher.UUID.String())
			state.Name = types.StringValue(publisher.Name)
			state.Description = types.StringValue(publisher.Description)
			state.PublisherClass = types.StringValue(publisher.PublisherClass)
			state.Template = types.StringValue(publisher.Template)
			state.TemplateMimeType = types.StringValue(publisher.TemplateMimeType)
			state.DefaultPublisher = types.BoolValue(publisher.DefaultPublisher)

			found = true
		}
	}

	if !found {
		resp.Diagnostics.AddError("Client Error", "The notification publisher could not be found")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

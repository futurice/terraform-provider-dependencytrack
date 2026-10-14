// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package extensionconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/futurice/terraform-provider-dependencytrack/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ExtensionConfigResource{}
var _ resource.ResourceWithImportState = &ExtensionConfigResource{}
var _ resource.ResourceWithModifyPlan = &ExtensionConfigResource{}
var _ resource.ResourceWithValidateConfig = &ExtensionConfigResource{}

func NewExtensionConfigResource() resource.Resource {
	return &ExtensionConfigResource{}
}

// ExtensionConfigResource defines the resource implementation.
type ExtensionConfigResource struct {
	client *dtrack.Client
}

// ExtensionConfigResourceModel describes the resource data model.
type ExtensionConfigResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ExtensionPoint types.String `tfsdk:"extension_point"`
	Extension      types.String `tfsdk:"extension"`
	Config         types.String `tfsdk:"config"`
}

func (r *ExtensionConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_extension_config"
}

func (r *ExtensionConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Config of an extension, such as the Trivy analyzer. Dependency-Track v5 only. " +
			"Every extension always has a config, so creating this resource replaces it and destroying it leaves the server config as it is.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic ID in the form of extension_point/extension",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"extension_point": schema.StringAttribute{
				MarkdownDescription: "Extension point, e.g. `vuln-analyzer` or `vuln-data-source`",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"extension": schema.StringAttribute{
				MarkdownDescription: "Extension, e.g. `trivy` or `github`",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"config": schema.StringAttribute{
				MarkdownDescription: "Config as a JSON object. It replaces the whole config, so fields left out lose their server defaults. " +
					"Dependency-Track validates it against the extension's config schema, and secret fields take the name of a `dependencytrack_secret`.",
				Required: true,
			},
		},
	}
}

func (r *ExtensionConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ExtensionConfigResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ExtensionConfigResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.Config.IsNull() || config.Config.IsUnknown() {
		return
	}

	var object map[string]interface{}
	if err := json.Unmarshal([]byte(config.Config.ValueString()), &object); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("config"), "Invalid Config", fmt.Sprintf("config must be a JSON object: %s", err))
	}
}

func (r *ExtensionConfigResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || r.client == nil {
		return
	}

	resp.Diagnostics.Append(utils.RequireV5(ctx, r.client, "dependencytrack_extension_config")...)
}

func (r *ExtensionConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ExtensionConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Extension.UpdateConfig(ctx, plan.ExtensionPoint.ValueString(), plan.Extension.ValueString(), json.RawMessage(plan.Config.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set extension config, got error: %s", err))
		return
	}

	plan.ID = types.StringValue(makeExtensionConfigID(plan.ExtensionPoint.ValueString(), plan.Extension.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ExtensionConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ExtensionConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.Extension.GetConfig(ctx, state.ExtensionPoint.ValueString(), state.Extension.ValueString())
	if err != nil {
		var apiErr *dtrack.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read extension config, got error: %s", err))
		return
	}

	// Keep the configured string when the server returns the same JSON reformatted.
	if state.Config.IsNull() || !utils.JSONEqual(string(config), state.Config.ValueString()) {
		state.Config = types.StringValue(utils.CompactJSON(string(config)))
	}
	state.ID = types.StringValue(makeExtensionConfigID(state.ExtensionPoint.ValueString(), state.Extension.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ExtensionConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ExtensionConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Extension.UpdateConfig(ctx, plan.ExtensionPoint.ValueString(), plan.Extension.ValueString(), json.RawMessage(plan.Config.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update extension config, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ExtensionConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// The config can't be deleted on the server, so destroying only forgets it.
	resp.State.RemoveResource(ctx)
}

func (r *ExtensionConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected ID in the format 'extension_point/extension', got [%s]", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("extension_point"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("extension"), parts[1])...)
}

func makeExtensionConfigID(extensionPoint, extension string) string {
	return fmt.Sprintf("%s/%s", extensionPoint, extension)
}

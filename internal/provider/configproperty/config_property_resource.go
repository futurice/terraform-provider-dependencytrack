// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package configproperty

import (
	"context"
	"fmt"
	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/futurice/terraform-provider-dependencytrack/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ConfigPropertyResource{}
var _ resource.ResourceWithImportState = &ConfigPropertyResource{}
var _ resource.ResourceWithModifyPlan = &ConfigPropertyResource{}

func NewConfigPropertyResource() resource.Resource {
	return &ConfigPropertyResource{}
}

// ConfigPropertyResource defines the resource implementation.
type ConfigPropertyResource struct {
	client *dtrack.Client
}

// ConfigPropertyResourceModel describes the resource data model.
type ConfigPropertyResourceModel struct {
	ID            types.String `tfsdk:"id"`
	GroupName     types.String `tfsdk:"group_name"`
	Name          types.String `tfsdk:"name"`
	Value         types.String `tfsdk:"value"`
	DestroyValue  types.String `tfsdk:"destroy_value"`
	OriginalValue types.String `tfsdk:"original_value"`
}

func (r *ConfigPropertyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_property"
}

func (r *ConfigPropertyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Configuration property. Dependency-Track v4 only. On v5, use `dependencytrack_extension_config` and `dependencytrack_secret`, and remove these resources from the configuration.",

		Attributes: map[string]schema.Attribute{
			"group_name": schema.StringAttribute{
				MarkdownDescription: "Name of the property group",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the property",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Value of the property",
				Required:            true,
				Sensitive:           true, // not for all variables, but for some yes
			},
			"destroy_value": schema.StringAttribute{
				MarkdownDescription: "Value of the property to set on destroy",
				Optional:            true,
				Sensitive:           true,
			},
			"original_value": schema.StringAttribute{
				MarkdownDescription: "Original value of the property to be restored on destroy (if any) unless `destroy_value` is set",
				Computed:            true,
				Sensitive:           true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic property ID in the form of group_name/name",
				Computed:            true,
			},
		},
	}
}

func (r *ConfigPropertyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ConfigPropertyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Destroy plans must keep working on v5, so the migration can remove these resources.
	if req.Plan.Raw.IsNull() || r.client == nil {
		return
	}

	v5, diags := utils.IsV5(ctx, r.client)
	resp.Diagnostics.Append(diags...)
	if !diags.HasError() && v5 {
		resp.Diagnostics.AddError("Unsupported Resource",
			"dependencytrack_config_property requires Dependency-Track v4. On v5, use dependencytrack_extension_config for analyzer and data source settings, and dependencytrack_secret for their tokens.")
	}
}

func (r *ConfigPropertyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, state ConfigPropertyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupName := plan.GroupName.ValueString()
	name := plan.Name.ValueString()
	value := plan.Value.ValueString()

	originalProperty, originalPropertyDiags := r.findConfigProperty(ctx, groupName, name)
	resp.Diagnostics.Append(originalPropertyDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	setConfigPropertyRequest := dtrack.SetConfigPropertyRequest{
		GroupName:     groupName,
		PropertyName:  name,
		PropertyValue: value,
	}

	_, err := r.client.Config.SetConfigProperty(ctx, setConfigPropertyRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set config property, got error: %s", err))
		return
	}

	state.ID = types.StringValue(makeConfigPropertyID(groupName, name))
	state.GroupName = types.StringValue(groupName)
	state.Name = types.StringValue(name)
	state.Value = types.StringValue(value)
	state.DestroyValue = plan.DestroyValue

	if originalProperty != nil && originalProperty.PropertyValue != nil {
		state.OriginalValue = types.StringValue(*originalProperty.PropertyValue)
	} else {
		state.OriginalValue = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ConfigPropertyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ConfigPropertyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// v5 has none of the v4 properties, so state carried over from v4 just goes away.
	v5, diags := utils.IsV5(ctx, r.client)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if v5 {
		resp.State.RemoveResource(ctx)
		return
	}

	groupName := state.GroupName.ValueString()
	name := state.Name.ValueString()

	property, propertyDiags := r.findConfigProperty(ctx, groupName, name)
	resp.Diagnostics.Append(propertyDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if property == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if property.PropertyValue != nil {
		state.Value = types.StringValue(*property.PropertyValue)
	} else {
		state.Value = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ConfigPropertyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ConfigPropertyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// only value can change via Update, destroy_value can change in TF state only
	groupName := state.GroupName.ValueString()
	name := state.Name.ValueString()
	value := plan.Value.ValueString()

	setConfigPropertyRequest := dtrack.SetConfigPropertyRequest{
		GroupName:     groupName,
		PropertyName:  name,
		PropertyValue: value,
	}

	_, err := r.client.Config.SetConfigProperty(ctx, setConfigPropertyRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set config property, got error: %s", err))
		return
	}

	state.Value = types.StringValue(value)
	state.DestroyValue = plan.DestroyValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ConfigPropertyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ConfigPropertyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// There is nothing to restore on v5, where the property no longer exists.
	v5, diags := utils.IsV5(ctx, r.client)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if v5 {
		resp.State.RemoveResource(ctx)
		return
	}

	groupName := state.GroupName.ValueString()
	name := state.Name.ValueString()

	var restoreValue *string
	if !state.DestroyValue.IsUnknown() && !state.DestroyValue.IsNull() {
		destroyValueTmp := state.DestroyValue.ValueString()
		restoreValue = &destroyValueTmp
	} else if !state.OriginalValue.IsUnknown() && !state.OriginalValue.IsNull() {
		originalValueTmp := state.OriginalValue.ValueString()
		restoreValue = &originalValueTmp
	} else {
		resp.Diagnostics.AddWarning("No value to restore", "Neither destroy_value not original_value is available on destroy - the property will not be modified in Dependency-Track")
	}

	if restoreValue != nil {
		setConfigPropertyRequest := dtrack.SetConfigPropertyRequest{
			GroupName:     groupName,
			PropertyName:  name,
			PropertyValue: *restoreValue,
		}

		_, err := r.client.Config.SetConfigProperty(ctx, setConfigPropertyRequest)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to reset config property to original value, got error: %s", err))
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

func (r *ConfigPropertyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Not supported", "Importing this resource is not necessary. Instead just create a resource to set the property value to what you want it to be")
}

func (r *ConfigPropertyResource) findConfigProperty(ctx context.Context, groupName, name string) (*dtrack.ConfigProperty, diag.Diagnostics) {
	var diags diag.Diagnostics

	configProperties, err := r.client.Config.GetAllConfigProperties(ctx)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to get config properties, got error: %s", err))
		return nil, diags
	}

	for _, configProperty := range configProperties {
		if configProperty.GroupName == groupName && configProperty.PropertyName == name {
			return &configProperty, diags
		}
	}

	return nil, diags
}

func makeConfigPropertyID(groupName string, name string) string {
	return fmt.Sprintf("%s/%s", groupName, name)
}

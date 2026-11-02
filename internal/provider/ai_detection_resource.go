// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pragmatic-scm/terraform-provider-community-reolink/internal/client"
)

var _ resource.Resource = (*aiDetectionResource)(nil)
var _ resource.ResourceWithConfigure = (*aiDetectionResource)(nil)
var _ resource.ResourceWithImportState = (*aiDetectionResource)(nil)

type aiDetectionResource struct{ cameras client.Registry }

type aiDetectionModel struct {
	ID            types.String  `tfsdk:"id"`
	Camera        types.String  `tfsdk:"camera"`
	Channel       types.Int64   `tfsdk:"channel"`
	Type          types.String  `tfsdk:"type"`
	Sensitivity   types.Int64   `tfsdk:"sensitivity"`
	Delay         types.Int64   `tfsdk:"delay"`
	MinimumWidth  types.Float64 `tfsdk:"minimum_width"`
	MinimumHeight types.Float64 `tfsdk:"minimum_height"`
	MaximumWidth  types.Float64 `tfsdk:"maximum_width"`
	MaximumHeight types.Float64 `tfsdk:"maximum_height"`
	Scope         types.String  `tfsdk:"scope"`
}

func NewAIDetectionResource() resource.Resource { return &aiDetectionResource{} }
func (r *aiDetectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_detection"
}
func (r *aiDetectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Manages one Reolink smart-detection class using GetAiAlarm and SetAiAlarm.", Attributes: map[string]schema.Attribute{
		"id":             schema.StringAttribute{Computed: true},
		"camera":         schema.StringAttribute{Required: true, Description: "Key in the provider cameras map."},
		"channel":        schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(0), Validators: []validator.Int64{int64validator.AtLeast(0)}},
		"type":           schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("people", "vehicle", "dog_cat", "face")}},
		"sensitivity":    schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.Between(0, 100)}},
		"delay":          schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(0), Validators: []validator.Int64{int64validator.Between(0, 8)}},
		"minimum_width":  schema.Float64Attribute{Optional: true, Computed: true},
		"minimum_height": schema.Float64Attribute{Optional: true, Computed: true},
		"maximum_width":  schema.Float64Attribute{Optional: true, Computed: true},
		"maximum_height": schema.Float64Attribute{Optional: true, Computed: true},
		"scope":          schema.StringAttribute{Optional: true, Computed: true, Description: "Camera-native AI detection mask bit string."},
	}}
}
func (r *aiDetectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	registry, ok := req.ProviderData.(client.Registry)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected client.Registry, got %T", req.ProviderData))
		return
	}
	r.cameras = registry
}
func (r *aiDetectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan aiDetectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.write(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Unable to configure AI detection", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *aiDetectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state aiDetectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.read(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Unable to read AI detection", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *aiDetectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan aiDetectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.write(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Unable to configure AI detection", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *aiDetectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Camera configuration has no delete operation. Removing the Terraform resource only stops managing it.
	resp.State.RemoveResource(ctx)
}
func (r *aiDetectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Use camera/channel/type, for example camera8/0/people.")
		return
	}
	channel, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import channel", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("camera"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("channel"), channel)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), parts[2])...)
}

func (r *aiDetectionResource) camera(key string) (*client.Client, error) {
	c, ok := r.cameras[key]
	if !ok {
		return nil, fmt.Errorf("camera %q is not defined in provider cameras", key)
	}
	return c, nil
}

func (r *aiDetectionResource) read(ctx context.Context, m *aiDetectionModel) (map[string]any, error) {
	c, err := r.camera(m.Camera.ValueString())
	if err != nil {
		return nil, err
	}
	e, err := c.Command(ctx, "GetAiAlarm", 1, map[string]any{"channel": m.Channel.ValueInt64(), "ai_type": m.Type.ValueString()})
	if err != nil {
		return nil, err
	}
	var wrapper map[string]any
	if err := json.Unmarshal(e.Value, &wrapper); err != nil {
		return nil, fmt.Errorf("decode AiAlarm: %w", err)
	}
	alarm, ok := wrapper["AiAlarm"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("GetAiAlarm omitted AiAlarm")
	}
	m.ID = types.StringValue(fmt.Sprintf("%s/%d/%s", m.Camera.ValueString(), m.Channel.ValueInt64(), m.Type.ValueString()))
	m.Sensitivity = types.Int64Value(numberInt(alarm["sensitivity"]))
	m.Delay = types.Int64Value(numberInt(alarm["stay_time"]))
	m.MinimumWidth = types.Float64Value(numberFloat(alarm["min_target_width"]))
	m.MinimumHeight = types.Float64Value(numberFloat(alarm["min_target_height"]))
	m.MaximumWidth = types.Float64Value(numberFloat(alarm["max_target_width"]))
	m.MaximumHeight = types.Float64Value(numberFloat(alarm["max_target_height"]))
	if scope, ok := alarm["scope"].(map[string]any); ok {
		if area, ok := scope["area"].(string); ok {
			m.Scope = types.StringValue(area)
		}
	}
	return alarm, nil
}

func (r *aiDetectionResource) write(ctx context.Context, m *aiDetectionModel) error {
	desired := *m
	alarm, err := r.read(ctx, m)
	if err != nil {
		return err
	}
	alarm["channel"] = desired.Channel.ValueInt64()
	alarm["ai_type"] = desired.Type.ValueString()
	alarm["sensitivity"] = desired.Sensitivity.ValueInt64()
	alarm["stay_time"] = desired.Delay.ValueInt64()
	if !desired.MinimumWidth.IsNull() && !desired.MinimumWidth.IsUnknown() {
		alarm["min_target_width"] = desired.MinimumWidth.ValueFloat64()
	}
	if !desired.MinimumHeight.IsNull() && !desired.MinimumHeight.IsUnknown() {
		alarm["min_target_height"] = desired.MinimumHeight.ValueFloat64()
	}
	if !desired.MaximumWidth.IsNull() && !desired.MaximumWidth.IsUnknown() {
		alarm["max_target_width"] = desired.MaximumWidth.ValueFloat64()
	}
	if !desired.MaximumHeight.IsNull() && !desired.MaximumHeight.IsUnknown() {
		alarm["max_target_height"] = desired.MaximumHeight.ValueFloat64()
	}
	if !desired.Scope.IsNull() && !desired.Scope.IsUnknown() {
		scope, _ := alarm["scope"].(map[string]any)
		if scope == nil {
			scope = map[string]any{}
		}
		scope["area"] = desired.Scope.ValueString()
		alarm["scope"] = scope
	}
	c, _ := r.camera(m.Camera.ValueString())
	_, err = c.Command(ctx, "SetAiAlarm", 0, map[string]any{"AiAlarm": alarm})
	if err != nil {
		return err
	}
	_, err = r.read(ctx, m)
	return err
}

func numberFloat(v any) float64 {
	if n, ok := v.(float64); ok {
		return n
	}
	return 0
}
func numberInt(v any) int64 { return int64(numberFloat(v)) }

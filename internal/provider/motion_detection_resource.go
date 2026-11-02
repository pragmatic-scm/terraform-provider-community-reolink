// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pragmatic-scm/terraform-provider-community-reolink/internal/client"
)

var _ resource.Resource = (*motionDetectionResource)(nil)
var _ resource.ResourceWithConfigure = (*motionDetectionResource)(nil)
var _ resource.ResourceWithImportState = (*motionDetectionResource)(nil)

type motionDetectionResource struct{ cameras client.Registry }
type motionDetectionModel struct {
	ID          types.String `tfsdk:"id"`
	Camera      types.String `tfsdk:"camera"`
	Channel     types.Int64  `tfsdk:"channel"`
	Sensitivity types.Int64  `tfsdk:"sensitivity"`
	Scope       types.String `tfsdk:"scope"`
}

func NewMotionDetectionResource() resource.Resource { return &motionDetectionResource{} }
func (r *motionDetectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_motion_detection"
}
func (r *motionDetectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Manages conventional Reolink motion sensitivity using GetMdAlarm and SetMdAlarm.", Attributes: map[string]schema.Attribute{
		"id":          schema.StringAttribute{Computed: true},
		"camera":      schema.StringAttribute{Required: true, Description: "Key in the provider cameras map."},
		"channel":     schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(0), Validators: []validator.Int64{int64validator.AtLeast(0)}},
		"sensitivity": schema.Int64Attribute{Required: true, Description: "Uniform all-day motion sensitivity.", Validators: []validator.Int64{int64validator.Between(1, 50)}},
		"scope":       schema.StringAttribute{Optional: true, Computed: true, Description: "Camera-native motion detection mask bit string."},
	}}
}
func (r *motionDetectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *motionDetectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan motionDetectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.write(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Unable to configure motion detection", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *motionDetectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state motionDetectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.read(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Unable to read motion detection", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *motionDetectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan motionDetectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.write(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Unable to configure motion detection", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *motionDetectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}
func (r *motionDetectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Use camera/channel, for example camera1/0.")
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
}
func (r *motionDetectionResource) camera(key string) (*client.Client, error) {
	c, ok := r.cameras[key]
	if !ok {
		return nil, fmt.Errorf("camera %q is not defined in provider cameras", key)
	}
	return c, nil
}
func (r *motionDetectionResource) read(ctx context.Context, m *motionDetectionModel) (map[string]any, error) {
	c, err := r.camera(m.Camera.ValueString())
	if err != nil {
		return nil, err
	}
	e, err := c.Command(ctx, "GetMdAlarm", 1, map[string]any{"channel": m.Channel.ValueInt64()})
	if err != nil {
		return nil, err
	}
	var wrapper map[string]any
	if err := json.Unmarshal(e.Value, &wrapper); err != nil {
		return nil, fmt.Errorf("decode MdAlarm: %w", err)
	}
	alarm, ok := wrapper["MdAlarm"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("GetMdAlarm omitted MdAlarm")
	}
	m.ID = types.StringValue(fmt.Sprintf("%s/%d", m.Camera.ValueString(), m.Channel.ValueInt64()))
	sensitivity := int64(0)
	if newSens, ok := alarm["newSens"].(map[string]any); ok {
		sensitivity = numberInt(newSens["sensDef"])
	}
	if sensitivity == 0 {
		if rows, ok := alarm["sens"].([]any); ok && len(rows) > 0 {
			if row, ok := rows[0].(map[string]any); ok {
				sensitivity = numberInt(row["sensitivity"])
			}
		}
	}
	m.Sensitivity = types.Int64Value(sensitivity)
	if scope, ok := alarm["scope"].(map[string]any); ok {
		if table, ok := scope["table"].(string); ok {
			m.Scope = types.StringValue(table)
		}
	}
	return alarm, nil
}
func (r *motionDetectionResource) write(ctx context.Context, m *motionDetectionModel) error {
	desired := *m
	alarm, err := r.read(ctx, m)
	if err != nil {
		return err
	}
	sensitivity := desired.Sensitivity.ValueInt64()
	alarm["channel"] = desired.Channel.ValueInt64()
	if rows, ok := alarm["sens"].([]any); ok {
		setSensitivity(rows, sensitivity)
	}
	if newSens, ok := alarm["newSens"].(map[string]any); ok {
		newSens["sensDef"] = sensitivity
		if rows, ok := newSens["sens"].([]any); ok {
			setSensitivity(rows, sensitivity)
		}
	}
	if !desired.Scope.IsNull() && !desired.Scope.IsUnknown() {
		scope, _ := alarm["scope"].(map[string]any)
		if scope == nil {
			scope = map[string]any{}
		}
		scope["table"] = desired.Scope.ValueString()
		alarm["scope"] = scope
	}
	c, _ := r.camera(desired.Camera.ValueString())
	if _, err := c.Command(ctx, "SetMdAlarm", 0, map[string]any{"MdAlarm": alarm}); err != nil {
		return err
	}
	_, err = r.read(ctx, m)
	return err
}
func setSensitivity(rows []any, value int64) {
	for _, item := range rows {
		if row, ok := item.(map[string]any); ok {
			row["sensitivity"] = value
		}
	}
}

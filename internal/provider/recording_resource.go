// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pragmatic-scm/terraform-provider-community-reolink/internal/client"
)

type recordingResource struct{ cameras client.Registry }
type recordingModel struct {
	ID       types.String `tfsdk:"id"`
	Camera   types.String `tfsdk:"camera"`
	Enabled  types.Bool   `tfsdk:"enabled"`
	Schedule types.String `tfsdk:"schedule"`
}
type recordingSettings struct {
	value   map[string]any
	version int
}

func NewRecordingResource() resource.Resource { return &recordingResource{} }
func (r *recordingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_recording"
}
func (r *recordingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Manages the Reolink recording enable switch and native alarm schedule.", Attributes: map[string]schema.Attribute{
		"id":       schema.StringAttribute{Computed: true},
		"camera":   schema.StringAttribute{Required: true, Description: "Key in the provider cameras map."},
		"enabled":  schema.BoolAttribute{Required: true, Description: "Whether camera recording is enabled."},
		"schedule": schema.StringAttribute{Required: true, Description: "Native JSON schedule object, preserving legacy and V20 alarm tables exactly."},
	}}
}
func (r *recordingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *recordingResource) camera(key string) (*client.Client, error) {
	c, ok := r.cameras[key]
	if !ok {
		return nil, fmt.Errorf("camera %q is not defined in provider cameras", key)
	}
	return c, nil
}
func (r *recordingResource) read(ctx context.Context, model *recordingModel) (*recordingSettings, error) {
	c, err := r.camera(model.Camera.ValueString())
	if err != nil {
		return nil, err
	}
	envelope, v20Err := c.Command(ctx, "GetRecV20", 1, map[string]any{"channel": 0})
	version := 20
	if v20Err != nil {
		version = 0
		envelope, err = c.Command(ctx, "GetRec", 1, map[string]any{"channel": 0})
		if err != nil {
			return nil, fmt.Errorf("GetRecV20 failed: %v; GetRec failed: %w", v20Err, err)
		}
	}
	var wrapper map[string]any
	if err := json.Unmarshal(envelope.Value, &wrapper); err != nil {
		return nil, fmt.Errorf("decode Rec: %w", err)
	}
	settings, ok := wrapper["Rec"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s omitted Rec", envelope.Cmd)
	}
	schedule, ok := settings["schedule"]
	if !ok {
		return nil, fmt.Errorf("%s omitted Rec.schedule", envelope.Cmd)
	}
	encoded, err := json.Marshal(schedule)
	if err != nil {
		return nil, fmt.Errorf("encode Rec schedule: %w", err)
	}
	model.ID = types.StringValue(model.Camera.ValueString())
	model.Schedule = types.StringValue(string(encoded))
	if version == 20 {
		model.Enabled = types.BoolValue(numberInt(settings["enable"]) != 0)
	} else {
		scheduleMap, ok := schedule.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("legacy GetRec omitted Rec.schedule object")
		}
		model.Enabled = types.BoolValue(numberInt(scheduleMap["enable"]) != 0)
	}
	return &recordingSettings{value: settings, version: version}, nil
}
func (r *recordingResource) write(ctx context.Context, model *recordingModel) error {
	desired := *model
	settings, err := r.read(ctx, model)
	if err != nil {
		return err
	}
	var schedule any
	if err := json.Unmarshal([]byte(desired.Schedule.ValueString()), &schedule); err != nil {
		return fmt.Errorf("decode desired Rec schedule: %w", err)
	}
	settings.value["schedule"] = schedule
	enable := 0
	if desired.Enabled.ValueBool() {
		enable = 1
	}
	command := "SetRecV20"
	if settings.version == 20 {
		settings.value["enable"] = enable
	} else {
		command = "SetRec"
		scheduleMap, ok := settings.value["schedule"].(map[string]any)
		if !ok {
			return fmt.Errorf("legacy Rec schedule is not an object")
		}
		scheduleMap["enable"] = enable
	}
	c, _ := r.camera(desired.Camera.ValueString())
	if _, err := c.Command(ctx, command, 0, map[string]any{"Rec": settings.value}); err != nil {
		return err
	}
	_, err = r.read(ctx, model)
	return err
}
func (r *recordingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan recordingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.write(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Unable to configure recording", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *recordingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state recordingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.read(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Unable to read recording", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *recordingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan recordingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.write(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Unable to configure recording", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *recordingResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}
func (r *recordingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Use the camera key, for example camera8.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("camera"), req.ID)...)
}

var _ resource.ResourceWithConfigure = (*recordingResource)(nil)
var _ resource.ResourceWithImportState = (*recordingResource)(nil)

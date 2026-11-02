// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pragmatic-scm/terraform-provider-community-reolink/internal/client"
)

var _ datasource.DataSource = (*cameraInfoDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*cameraInfoDataSource)(nil)

type cameraInfoDataSource struct{ cameras client.Registry }

type cameraInfoModel struct {
	Camera          types.String `tfsdk:"camera"`
	Name            types.String `tfsdk:"name"`
	Model           types.String `tfsdk:"model"`
	ExactType       types.String `tfsdk:"exact_type"`
	HardwareVersion types.String `tfsdk:"hardware_version"`
	FirmwareVersion types.String `tfsdk:"firmware_version"`
	ConfigVersion   types.String `tfsdk:"config_version"`
	BuildDay        types.String `tfsdk:"build_day"`
	ChannelCount    types.Int64  `tfsdk:"channel_count"`
}

func NewCameraInfoDataSource() datasource.DataSource { return &cameraInfoDataSource{} }

func (d *cameraInfoDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_camera_info"
}

func (d *cameraInfoDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads model, hardware, and firmware metadata from a Reolink camera using GetDevInfo.",
		Attributes: map[string]schema.Attribute{
			"camera":           schema.StringAttribute{Required: true, Description: "Key in the provider cameras map."},
			"name":             schema.StringAttribute{Computed: true, Description: "Camera-reported device name."},
			"model":            schema.StringAttribute{Computed: true, Description: "Camera model."},
			"exact_type":       schema.StringAttribute{Computed: true, Description: "Camera-reported exact device type."},
			"hardware_version": schema.StringAttribute{Computed: true, Description: "Hardware version."},
			"firmware_version": schema.StringAttribute{Computed: true, Description: "Firmware version."},
			"config_version":   schema.StringAttribute{Computed: true, Description: "Configuration version."},
			"build_day":        schema.StringAttribute{Computed: true, Description: "Firmware build date reported by the camera."},
			"channel_count":    schema.Int64Attribute{Computed: true, Description: "Number of camera channels."},
		},
	}
}

func (d *cameraInfoDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	registry, ok := req.ProviderData.(client.Registry)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected client.Registry, got %T", req.ProviderData))
		return
	}
	d.cameras = registry
}

func (d *cameraInfoDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state cameraInfoModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c, ok := d.cameras[state.Camera.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Camera not configured", fmt.Sprintf("Camera %q is not defined in provider cameras", state.Camera.ValueString()))
		return
	}
	envelope, err := c.Command(ctx, "GetDevInfo", 0, map[string]any{})
	if err != nil {
		resp.Diagnostics.AddError("Unable to read camera information", err.Error())
		return
	}
	var wrapper map[string]any
	if err := json.Unmarshal(envelope.Value, &wrapper); err != nil {
		resp.Diagnostics.AddError("Unable to decode camera information", err.Error())
		return
	}
	info, ok := wrapper["DevInfo"].(map[string]any)
	if !ok {
		resp.Diagnostics.AddError("Unable to read camera information", "GetDevInfo omitted DevInfo")
		return
	}
	state.Name = types.StringValue(stringValue(info["name"]))
	state.Model = types.StringValue(stringValue(info["model"]))
	state.ExactType = types.StringValue(stringValue(info["exactType"]))
	state.HardwareVersion = types.StringValue(stringValue(info["hardVer"]))
	state.FirmwareVersion = types.StringValue(stringValue(info["firmVer"]))
	state.ConfigVersion = types.StringValue(stringValue(info["cfgVer"]))
	state.BuildDay = types.StringValue(stringValue(info["buildDay"]))
	state.ChannelCount = types.Int64Value(numberInt(info["channelNum"]))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

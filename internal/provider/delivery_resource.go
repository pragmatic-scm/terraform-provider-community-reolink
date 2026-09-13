// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/eremington/terraform-provider-reolink/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*deliveryResource)(nil)
var _ resource.ResourceWithConfigure = (*deliveryResource)(nil)
var _ resource.ResourceWithImportState = (*deliveryResource)(nil)
var _ resource.ResourceWithUpgradeState = (*deliveryResource)(nil)

type deliveryResource struct {
	cameras client.Registry
	kind    string
}

type deliveryModel struct {
	ID            types.String `tfsdk:"id"`
	Camera        types.String `tfsdk:"camera"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	Schedule      types.String `tfsdk:"schedule"`
	Configuration types.String `tfsdk:"configuration"`
	Revision      types.String `tfsdk:"configuration_revision"`
}

type deliveryModelV1 struct {
	ID            types.String `tfsdk:"id"`
	Camera        types.String `tfsdk:"camera"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	Configuration types.String `tfsdk:"configuration"`
	Revision      types.String `tfsdk:"configuration_revision"`
}

type deliveryModelV0 struct {
	ID            types.String `tfsdk:"id"`
	Camera        types.String `tfsdk:"camera"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	Configuration types.String `tfsdk:"configuration"`
}

type deliverySettings struct {
	value   map[string]any
	version int
}

func NewEmailResource() resource.Resource { return &deliveryResource{kind: "Email"} }
func NewFTPResource() resource.Resource   { return &deliveryResource{kind: "Ftp"} }

func (r *deliveryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + strings.ToLower(r.kind)
}

func (r *deliveryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     2,
		Description: fmt.Sprintf("Manages the Reolink %s enable switch while preserving existing settings.", r.kind),
		Attributes: map[string]schema.Attribute{
			"id":                     schema.StringAttribute{Computed: true},
			"camera":                 schema.StringAttribute{Required: true, Description: "Key in the provider cameras map."},
			"enabled":                schema.BoolAttribute{Required: true},
			"schedule":               schema.StringAttribute{Optional: true, Computed: true, Description: "Native JSON schedule object, preserving legacy and V20 alarm tables exactly."},
			"configuration":          schema.StringAttribute{Required: true, Sensitive: true, WriteOnly: true, Description: "Write-only JSON object of desired delivery fields loaded from 1Password."},
			"configuration_revision": schema.StringAttribute{Required: true, Description: "Non-secret revision marker used to trigger updates when the write-only configuration changes."},
		},
	}
}

func (r *deliveryResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	priorSchema := schema.Schema{
		Description: fmt.Sprintf("Manages the Reolink %s enable switch while preserving existing settings.", r.kind),
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Computed: true},
			"camera":        schema.StringAttribute{Required: true, Description: "Key in the provider cameras map."},
			"enabled":       schema.BoolAttribute{Required: true},
			"configuration": schema.StringAttribute{Required: true, Sensitive: true, Description: "JSON object of desired delivery fields loaded from 1Password."},
		},
	}
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior deliveryModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}
				upgraded := deliveryModel{
					ID: prior.ID, Camera: prior.Camera, Enabled: prior.Enabled,
					Schedule: types.StringNull(), Configuration: types.StringNull(), Revision: types.StringNull(),
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
		1: {
			PriorSchema: func() *schema.Schema {
				prior := schema.Schema{Version: 1, Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{Computed: true}, "camera": schema.StringAttribute{Required: true},
					"enabled":                schema.BoolAttribute{Required: true},
					"configuration":          schema.StringAttribute{Required: true, Sensitive: true, WriteOnly: true},
					"configuration_revision": schema.StringAttribute{Required: true},
				}}
				return &prior
			}(),
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior deliveryModelV1
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}
				upgraded := deliveryModel{ID: prior.ID, Camera: prior.Camera, Enabled: prior.Enabled, Schedule: types.StringNull(), Configuration: types.StringNull(), Revision: prior.Revision}
				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
	}
}

func (r *deliveryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *deliveryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deliveryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var config deliveryModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Configuration = config.Configuration
	if err := r.write(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Unable to configure "+r.kind, err.Error())
		return
	}
	plan.Configuration = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *deliveryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deliveryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.read(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Unable to read "+r.kind, err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *deliveryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan deliveryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var config deliveryModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Configuration = config.Configuration
	if err := r.write(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Unable to configure "+r.kind, err.Error())
		return
	}
	plan.Configuration = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *deliveryResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

func (r *deliveryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Use the camera key, for example camera8.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("camera"), req.ID)...)
}

func (r *deliveryResource) camera(key string) (*client.Client, error) {
	c, ok := r.cameras[key]
	if !ok {
		return nil, fmt.Errorf("camera %q is not defined in provider cameras", key)
	}
	return c, nil
}

func (r *deliveryResource) read(ctx context.Context, model *deliveryModel) (*deliverySettings, error) {
	c, err := r.camera(model.Camera.ValueString())
	if err != nil {
		return nil, err
	}
	getV20 := "Get" + r.kind + "V20"
	getLegacy := "Get" + r.kind
	envelope, v20Err := c.Command(ctx, getV20, 1, map[string]any{"channel": 0})
	version := 20
	if v20Err != nil {
		version = 0
		envelope, err = c.Command(ctx, getLegacy, 1, map[string]any{"channel": 0})
		if err != nil {
			return nil, fmt.Errorf("%s failed: %v; %s failed: %w", getV20, v20Err, getLegacy, err)
		}
	}
	var wrapper map[string]any
	if err := json.Unmarshal(envelope.Value, &wrapper); err != nil {
		return nil, fmt.Errorf("decode %s: %w", r.kind, err)
	}
	settings, ok := wrapper[r.kind].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s omitted %s", envelope.Cmd, r.kind)
	}
	model.ID = types.StringValue(model.Camera.ValueString())
	if schedule, ok := settings["schedule"]; ok {
		encoded, marshalErr := json.Marshal(schedule)
		if marshalErr != nil {
			return nil, fmt.Errorf("encode %s schedule: %w", r.kind, marshalErr)
		}
		model.Schedule = types.StringValue(string(encoded))
	} else {
		model.Schedule = types.StringNull()
	}
	if version == 20 {
		model.Enabled = types.BoolValue(numberInt(settings["enable"]) != 0)
	} else {
		schedule, ok := settings["schedule"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("legacy %s omitted %s.schedule", getLegacy, r.kind)
		}
		model.Enabled = types.BoolValue(numberInt(schedule["enable"]) != 0)
	}
	return &deliverySettings{value: settings, version: version}, nil
}

func (r *deliveryResource) write(ctx context.Context, model *deliveryModel) error {
	desired := *model
	settings, err := r.read(ctx, model)
	if err != nil {
		return err
	}
	enable := 0
	if desired.Enabled.ValueBool() {
		enable = 1
	}
	var configured map[string]any
	if err := json.Unmarshal([]byte(desired.Configuration.ValueString()), &configured); err != nil {
		return fmt.Errorf("decode desired %s configuration: %w", r.kind, err)
	}
	for key, value := range configured {
		settings.value[key] = value
	}
	if !desired.Schedule.IsNull() && !desired.Schedule.IsUnknown() {
		var schedule any
		if err := json.Unmarshal([]byte(desired.Schedule.ValueString()), &schedule); err != nil {
			return fmt.Errorf("decode desired %s schedule: %w", r.kind, err)
		}
		settings.value["schedule"] = schedule
	}
	if r.kind == "Email" {
		delete(settings.value, "supportTextType")
		delete(settings.value, "supportVideo")
	}
	command := "Set" + r.kind + "V20"
	if settings.version == 20 {
		settings.value["enable"] = enable
	} else {
		command = "Set" + r.kind
		if r.kind == "Ftp" {
			delete(settings.value, "onlyFtps")
			delete(settings.value, "autoDir")
			delete(settings.value, "picInterval")
		}
		schedule, ok := settings.value["schedule"].(map[string]any)
		if !ok {
			return fmt.Errorf("legacy %s settings omitted schedule", r.kind)
		}
		schedule["enable"] = enable
		settings.value["schedule"] = schedule
	}
	c, _ := r.camera(model.Camera.ValueString())
	if _, err := c.Command(ctx, command, 0, map[string]any{r.kind: settings.value}); err != nil {
		return err
	}
	_, err = r.read(ctx, model)
	return err
}

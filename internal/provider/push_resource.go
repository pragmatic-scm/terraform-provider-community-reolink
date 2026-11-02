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

var _ resource.Resource = (*pushResource)(nil)
var _ resource.ResourceWithConfigure = (*pushResource)(nil)
var _ resource.ResourceWithImportState = (*pushResource)(nil)

type pushResource struct{ cameras client.Registry }

type pushModel struct {
	ID      types.String `tfsdk:"id"`
	Camera  types.String `tfsdk:"camera"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

type pushSettings struct {
	value   map[string]any
	version int
}

func NewPushResource() resource.Resource { return &pushResource{} }

func (r *pushResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_push"
}

func (r *pushResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Reolink Push notifications using GetPush and SetPush.",
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Computed: true},
			"camera":  schema.StringAttribute{Required: true, Description: "Key in the provider cameras map."},
			"enabled": schema.BoolAttribute{Required: true},
		},
	}
}

func (r *pushResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *pushResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan pushModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.write(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Unable to configure Push notifications", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pushResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state pushModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.read(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Unable to read Push notifications", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pushResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan pushModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.write(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Unable to configure Push notifications", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pushResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Camera configuration has no delete operation. Removing the resource only stops managing it.
	resp.State.RemoveResource(ctx)
}

func (r *pushResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Use the camera key, for example camera8.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("camera"), req.ID)...)
}

func (r *pushResource) camera(key string) (*client.Client, error) {
	c, ok := r.cameras[key]
	if !ok {
		return nil, fmt.Errorf("camera %q is not defined in provider cameras", key)
	}
	return c, nil
}

func (r *pushResource) read(ctx context.Context, model *pushModel) (*pushSettings, error) {
	c, err := r.camera(model.Camera.ValueString())
	if err != nil {
		return nil, err
	}
	var envelope *client.Envelope
	var version int
	var v20Err error
	if envelope, v20Err = c.Command(ctx, "GetPushV20", 1, map[string]any{"channel": 0}); v20Err == nil {
		version = 20
	} else {
		envelope, err = c.Command(ctx, "GetPush", 1, map[string]any{"channel": 0})
		if err != nil {
			return nil, fmt.Errorf("GetPushV20 failed: %v; GetPush failed: %w", v20Err, err)
		}
	}
	var wrapper map[string]any
	if err := json.Unmarshal(envelope.Value, &wrapper); err != nil {
		return nil, fmt.Errorf("decode Push: %w", err)
	}
	push, ok := wrapper["Push"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("GetPush omitted Push")
	}
	model.ID = types.StringValue(model.Camera.ValueString())
	if version == 20 {
		model.Enabled = types.BoolValue(numberInt(push["enable"]) != 0)
	} else {
		schedule, ok := push["schedule"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("legacy GetPush omitted Push.schedule")
		}
		model.Enabled = types.BoolValue(numberInt(schedule["enable"]) != 0)
	}
	return &pushSettings{value: push, version: version}, nil
}

func (r *pushResource) write(ctx context.Context, model *pushModel) error {
	desired := model.Enabled.ValueBool()
	settings, err := r.read(ctx, model)
	if err != nil {
		return err
	}
	enable := 0
	if desired {
		enable = 1
	}
	command := "SetPushV20"
	if settings.version == 20 {
		settings.value["enable"] = enable
	} else {
		command = "SetPush"
		schedule, _ := settings.value["schedule"].(map[string]any)
		schedule["enable"] = enable
		settings.value["schedule"] = schedule
	}
	c, _ := r.camera(model.Camera.ValueString())
	if _, err := c.Command(ctx, command, 0, map[string]any{"Push": settings.value}); err != nil {
		return err
	}
	_, err = r.read(ctx, model)
	return err
}

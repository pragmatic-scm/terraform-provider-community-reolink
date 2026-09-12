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

type deliveryResource struct {
	cameras client.Registry
	kind    string
}

type deliveryModel struct {
	ID      types.String `tfsdk:"id"`
	Camera  types.String `tfsdk:"camera"`
	Enabled types.Bool   `tfsdk:"enabled"`
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
		Description: fmt.Sprintf("Manages the Reolink %s enable switch while preserving existing settings.", r.kind),
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Computed: true},
			"camera":  schema.StringAttribute{Required: true, Description: "Key in the provider cameras map."},
			"enabled": schema.BoolAttribute{Required: true},
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
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.write(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Unable to configure "+r.kind, err.Error())
		return
	}
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
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.write(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Unable to configure "+r.kind, err.Error())
		return
	}
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
	desired := model.Enabled.ValueBool()
	settings, err := r.read(ctx, model)
	if err != nil {
		return err
	}
	enable := 0
	if desired {
		enable = 1
	}
	command := "Set" + r.kind + "V20"
	payload := map[string]any{r.kind: map[string]any{"enable": enable}}
	if settings.version == 20 {
		// The minimal switch payload avoids echoing masked credentials returned
		// by GetEmailV20/GetFtpV20 back into the camera configuration.
	} else {
		command = "Set" + r.kind
		payload = map[string]any{r.kind: map[string]any{"schedule": map[string]any{"enable": enable}}}
	}
	c, _ := r.camera(model.Camera.ValueString())
	if _, err := c.Command(ctx, command, 0, payload); err != nil {
		return err
	}
	_, err = r.read(ctx, model)
	return err
}

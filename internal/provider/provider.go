// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/eremington/terraform-provider-reolink/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = (*reolinkProvider)(nil)

type reolinkProvider struct{ version string }
type providerModel struct {
	Cameras types.Map `tfsdk:"cameras"`
}
type cameraModel struct {
	URL      types.String `tfsdk:"url"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Insecure types.Bool   `tfsdk:"insecure"`
}

func New() provider.Provider { return &reolinkProvider{version: "0.1.0"} }
func (p *reolinkProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "reolink"
	resp.Version = p.version
}
func (p *reolinkProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"cameras": schema.MapNestedAttribute{Required: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"url":      schema.StringAttribute{Required: true},
			"username": schema.StringAttribute{Required: true},
			"password": schema.StringAttribute{Required: true, Sensitive: true},
			"insecure": schema.BoolAttribute{Optional: true},
		}}},
	}}
}
func (p *reolinkProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	models := map[string]cameraModel{}
	resp.Diagnostics.Append(config.Cameras.ElementsAs(ctx, &models, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	registry := client.Registry{}
	for key, model := range models {
		c, err := client.New(client.Config{URL: model.URL.ValueString(), Username: model.Username.ValueString(), Password: model.Password.ValueString(), Insecure: model.Insecure.ValueBool()})
		if err != nil {
			resp.Diagnostics.AddError("Invalid camera configuration", fmt.Sprintf("Camera %q: %s", key, err))
			continue
		}
		registry[key] = c
	}
	resp.ResourceData = registry
}
func (p *reolinkProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewAIDetectionResource, NewMotionDetectionResource}
}
func (p *reolinkProvider) DataSources(context.Context) []func() datasource.DataSource { return nil }

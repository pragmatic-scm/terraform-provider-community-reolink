// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestProviderSchemaIsValid(t *testing.T) {
	resp := &frameworkprovider.SchemaResponse{}
	New("test")().Schema(context.Background(), frameworkprovider.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("provider schema diagnostics: %v", resp.Diagnostics)
	}
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("invalid provider schema: %v", diags)
	}
}

func TestProviderMetadataUsesBuildVersion(t *testing.T) {
	resp := &frameworkprovider.MetadataResponse{}
	New("1.2.3")().Metadata(context.Background(), frameworkprovider.MetadataRequest{}, resp)
	if resp.TypeName != "reolink" {
		t.Fatalf("provider type name = %q, want reolink", resp.TypeName)
	}
	if resp.Version != "1.2.3" {
		t.Fatalf("provider version = %q, want 1.2.3", resp.Version)
	}
}

func TestAIDetectionSchemaIsValid(t *testing.T) {
	resp := &resource.SchemaResponse{}
	NewAIDetectionResource().Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("resource schema diagnostics: %v", resp.Diagnostics)
	}
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("invalid AI detection schema: %v", diags)
	}
}

func TestMotionDetectionSchemaIsValid(t *testing.T) {
	resp := &resource.SchemaResponse{}
	NewMotionDetectionResource().Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("resource schema diagnostics: %v", resp.Diagnostics)
	}
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("invalid motion detection schema: %v", diags)
	}
}

func TestPushSchemaIsValid(t *testing.T) {
	resp := &resource.SchemaResponse{}
	NewPushResource().Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("resource schema diagnostics: %v", resp.Diagnostics)
	}
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("invalid Push schema: %v", diags)
	}
}

func TestCameraInfoSchemaIsValid(t *testing.T) {
	resp := &datasource.SchemaResponse{}
	NewCameraInfoDataSource().Schema(context.Background(), datasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("data source schema diagnostics: %v", resp.Diagnostics)
	}
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("invalid camera info schema: %v", diags)
	}
}

func TestEmailSchemaIsValid(t *testing.T) {
	resp := &resource.SchemaResponse{}
	NewEmailResource().Schema(context.Background(), resource.SchemaRequest{}, resp)
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("invalid Email schema: %v", diags)
	}
}

func TestFTPSchemaIsValid(t *testing.T) {
	resp := &resource.SchemaResponse{}
	NewFTPResource().Schema(context.Background(), resource.SchemaRequest{}, resp)
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("invalid FTP schema: %v", diags)
	}
}

func TestRecordingSchemaIsValid(t *testing.T) {
	resp := &resource.SchemaResponse{}
	NewRecordingResource().Schema(context.Background(), resource.SchemaRequest{}, resp)
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("invalid recording schema: %v", diags)
	}
}

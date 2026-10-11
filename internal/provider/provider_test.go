package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestProvider_SchemaAndResources(t *testing.T) {
	ctx := context.Background()
	p := New("1.0.0")()

	var metaResp provider.MetadataResponse
	p.Metadata(ctx, provider.MetadataRequest{}, &metaResp)
	if metaResp.TypeName != "polaris" {
		t.Fatalf("expected type name 'polaris', got '%s'", metaResp.TypeName)
	}

	var schemaResp provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("provider schema has diagnostics: %v", schemaResp.Diagnostics)
	}

	resources := p.Resources(ctx)
	if len(resources) != 7 {
		t.Fatalf("expected 7 resources, got %d", len(resources))
	}

	for _, rf := range resources {
		r := rf()
		var rMetaResp resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "polaris"}, &rMetaResp)
		var rSchemaResp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &rSchemaResp)
		if rSchemaResp.Diagnostics.HasError() {
			t.Fatalf("resource %s schema has errors: %v", rMetaResp.TypeName, rSchemaResp.Diagnostics)
		}
	}

	dataSources := p.DataSources(ctx)
	if len(dataSources) != 4 {
		t.Fatalf("expected 4 data sources, got %d", len(dataSources))
	}

	for _, dsf := range dataSources {
		d := dsf()
		var dMetaResp datasource.MetadataResponse
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "polaris"}, &dMetaResp)
		var dSchemaResp datasource.SchemaResponse
		d.Schema(ctx, datasource.SchemaRequest{}, &dSchemaResp)
		if dSchemaResp.Diagnostics.HasError() {
			t.Fatalf("data source %s schema has errors: %v", dMetaResp.TypeName, dSchemaResp.Diagnostics)
		}
	}
}

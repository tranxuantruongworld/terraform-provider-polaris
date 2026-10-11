package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tranxuantruongworld/terraform-provider-polaris/internal/client"
)

var (
	_ datasource.DataSource              = &CatalogDataSource{}
	_ datasource.DataSourceWithConfigure = &CatalogDataSource{}
)

func NewCatalogDataSource() datasource.DataSource {
	return &CatalogDataSource{}
}

type CatalogDataSource struct {
	client *client.Client
}

type CatalogDataSourceModel struct {
	Name                types.String `tfsdk:"name"`
	Type                types.String `tfsdk:"type"`
	DefaultBaseLocation types.String `tfsdk:"default_base_location"`
	Properties          types.Map    `tfsdk:"properties"`
	CreateTimestamp     types.Int64  `tfsdk:"create_timestamp"`
	LastUpdateTimestamp types.Int64  `tfsdk:"last_update_timestamp"`
	EntityVersion       types.Int64  `tfsdk:"entity_version"`
}

func (d *CatalogDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_catalog"
}

func (d *CatalogDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads metadata of an existing Apache Polaris catalog.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the catalog to query.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The catalog type (`INTERNAL` or `EXTERNAL`).",
				Computed:    true,
			},
			"default_base_location": schema.StringAttribute{
				Description: "Default base location configured for the catalog.",
				Computed:    true,
			},
			"properties": schema.MapAttribute{
				Description: "Catalog metadata key-value properties.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"create_timestamp": schema.Int64Attribute{
				Description: "Creation epoch timestamp in milliseconds.",
				Computed:    true,
			},
			"last_update_timestamp": schema.Int64Attribute{
				Description: "Last update epoch timestamp in milliseconds.",
				Computed:    true,
			},
			"entity_version": schema.Int64Attribute{
				Description: "Current entity version counter.",
				Computed:    true,
			},
		},
	}
}

func (d *CatalogDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to provider maintainers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *CatalogDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CatalogDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cat, err := d.client.GetCatalog(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Polaris Catalog", fmt.Sprintf("Could not read catalog %s: %s", data.Name.ValueString(), err.Error()))
		return
	}

	data.Name = types.StringValue(cat.Name)
	data.Type = types.StringValue(cat.Type)
	data.CreateTimestamp = types.Int64Value(cat.CreateTimestamp)
	data.LastUpdateTimestamp = types.Int64Value(cat.LastUpdateTimestamp)
	data.EntityVersion = types.Int64Value(cat.EntityVersion)

	if loc, ok := cat.Properties["default-base-location"]; ok {
		data.DefaultBaseLocation = types.StringValue(loc)
	}

	if len(cat.Properties) > 0 {
		propsMap, _ := types.MapValueFrom(ctx, types.StringType, cat.Properties)
		data.Properties = propsMap
	} else {
		data.Properties = types.MapNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}


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
	_ datasource.DataSource              = &CatalogRoleDataSource{}
	_ datasource.DataSourceWithConfigure = &CatalogRoleDataSource{}
)

func NewCatalogRoleDataSource() datasource.DataSource {
	return &CatalogRoleDataSource{}
}

type CatalogRoleDataSource struct {
	client *client.Client
}

type CatalogRoleDataSourceModel struct {
	CatalogName         types.String `tfsdk:"catalog_name"`
	Name                types.String `tfsdk:"name"`
	Properties          types.Map    `tfsdk:"properties"`
	CreateTimestamp     types.Int64  `tfsdk:"create_timestamp"`
	LastUpdateTimestamp types.Int64  `tfsdk:"last_update_timestamp"`
	EntityVersion       types.Int64  `tfsdk:"entity_version"`
}

func (d *CatalogRoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_catalog_role"
}

func (d *CatalogRoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads metadata of an existing Apache Polaris catalog role.",
		Attributes: map[string]schema.Attribute{
			"catalog_name": schema.StringAttribute{
				Description: "The name of the catalog containing the role.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the catalog role to query.",
				Required:    true,
			},
			"properties": schema.MapAttribute{
				Description: "Custom properties associated with this catalog role.",
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

func (d *CatalogRoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CatalogRoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CatalogRoleDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := d.client.GetCatalogRole(ctx, data.CatalogName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Catalog Role", fmt.Sprintf("Could not read catalog role %s in catalog %s: %s", data.Name.ValueString(), data.CatalogName.ValueString(), err.Error()))
		return
	}

	data.CatalogName = types.StringValue(data.CatalogName.ValueString())
	data.Name = types.StringValue(role.Name)
	data.CreateTimestamp = types.Int64Value(role.CreateTimestamp)
	data.LastUpdateTimestamp = types.Int64Value(role.LastUpdateTimestamp)
	data.EntityVersion = types.Int64Value(role.EntityVersion)

	if len(role.Properties) > 0 {
		propsMap, _ := types.MapValueFrom(ctx, types.StringType, role.Properties)
		data.Properties = propsMap
	} else {
		data.Properties = types.MapNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}


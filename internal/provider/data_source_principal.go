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
	_ datasource.DataSource              = &PrincipalDataSource{}
	_ datasource.DataSourceWithConfigure = &PrincipalDataSource{}
)

func NewPrincipalDataSource() datasource.DataSource {
	return &PrincipalDataSource{}
}

type PrincipalDataSource struct {
	client *client.Client
}

type PrincipalDataSourceModel struct {
	Name                types.String `tfsdk:"name"`
	ClientID            types.String `tfsdk:"client_id"`
	Properties          types.Map    `tfsdk:"properties"`
	CreateTimestamp     types.Int64  `tfsdk:"create_timestamp"`
	LastUpdateTimestamp types.Int64  `tfsdk:"last_update_timestamp"`
	EntityVersion       types.Int64  `tfsdk:"entity_version"`
}

func (d *PrincipalDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_principal"
}

func (d *PrincipalDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PrincipalDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads metadata of an existing Apache Polaris principal.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the principal to query.",
				Required:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "OAuth2 client ID associated with the principal.",
				Computed:    true,
			},
			"properties": schema.MapAttribute{
				Description: "Custom properties associated with this principal.",
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

func (d *PrincipalDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PrincipalDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	principal, err := d.client.GetPrincipal(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Polaris Principal", fmt.Sprintf("Could not read principal %s: %s", data.Name.ValueString(), err.Error()))
		return
	}

	data.Name = types.StringValue(principal.Name)
	if principal.ClientID != "" {
		data.ClientID = types.StringValue(principal.ClientID)
	}
	data.CreateTimestamp = types.Int64Value(principal.CreateTimestamp)
	data.LastUpdateTimestamp = types.Int64Value(principal.LastUpdateTimestamp)
	data.EntityVersion = types.Int64Value(principal.EntityVersion)

	if len(principal.Properties) > 0 {
		propsMap, _ := types.MapValueFrom(ctx, types.StringType, principal.Properties)
		data.Properties = propsMap
	} else {
		data.Properties = types.MapNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

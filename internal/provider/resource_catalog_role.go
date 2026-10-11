package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tranxuantruongworld/terraform-provider-polaris/internal/client"
)

var (
	_ resource.Resource                = &CatalogRoleResource{}
	_ resource.ResourceWithConfigure   = &CatalogRoleResource{}
	_ resource.ResourceWithImportState = &CatalogRoleResource{}
)

func NewCatalogRoleResource() resource.Resource {
	return &CatalogRoleResource{}
}

type CatalogRoleResource struct {
	client *client.Client
}

type CatalogRoleResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	CatalogName         types.String `tfsdk:"catalog_name"`
	Name                types.String `tfsdk:"name"`
	Properties          types.Map    `tfsdk:"properties"`
	CreateTimestamp     types.Int64  `tfsdk:"create_timestamp"`
	LastUpdateTimestamp types.Int64  `tfsdk:"last_update_timestamp"`
	EntityVersion       types.Int64  `tfsdk:"entity_version"`
}

func (r *CatalogRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_catalog_role"
}

func (r *CatalogRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Apache Polaris catalog role scoped to a catalog.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite identifier of the catalog role in format `catalog_name/role_name`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"catalog_name": schema.StringAttribute{
				Description: "The catalog in which the role is created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the catalog role.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"properties": schema.MapAttribute{
				Description: "Custom properties associated with this catalog role.",
				Optional:    true,
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
				Description: "Current entity version counter for optimistic concurrency.",
				Computed:    true,
			},
		},
	}
}

func (r *CatalogRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to provider maintainers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *CatalogRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CatalogRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	props := make(map[string]string)
	if !data.Properties.IsNull() && !data.Properties.IsUnknown() {
		resp.Diagnostics.Append(data.Properties.ElementsAs(ctx, &props, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	createReq := client.CreateCatalogRoleRequest{
		CatalogRole: client.CatalogRole{
			Name:       data.Name.ValueString(),
			Properties: props,
		},
	}

	created, err := r.client.CreateCatalogRole(ctx, data.CatalogName.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Catalog Role", fmt.Sprintf("Could not create catalog role %s in catalog %s: %s", data.Name.ValueString(), data.CatalogName.ValueString(), err.Error()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.CatalogName.ValueString(), created.Name))
	data.Name = types.StringValue(created.Name)
	data.CreateTimestamp = types.Int64Value(created.CreateTimestamp)
	data.LastUpdateTimestamp = types.Int64Value(created.LastUpdateTimestamp)
	data.EntityVersion = types.Int64Value(created.EntityVersion)

	if len(created.Properties) > 0 {
		propsMap, _ := types.MapValueFrom(ctx, types.StringType, created.Properties)
		data.Properties = propsMap
	} else {
		data.Properties = types.MapNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CatalogRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CatalogRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.GetCatalogRole(ctx, data.CatalogName.ValueString(), data.Name.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Catalog Role", fmt.Sprintf("Could not read catalog role %s in catalog %s: %s", data.Name.ValueString(), data.CatalogName.ValueString(), err.Error()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.CatalogName.ValueString(), role.Name))
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

func (r *CatalogRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CatalogRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	props := make(map[string]string)
	if !data.Properties.IsNull() && !data.Properties.IsUnknown() {
		resp.Diagnostics.Append(data.Properties.ElementsAs(ctx, &props, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	updateReq := client.UpdateCatalogRoleRequest{
		CurrentEntityVersion: data.EntityVersion.ValueInt64(),
		Properties:           props,
	}

	updated, err := r.client.UpdateCatalogRole(ctx, data.CatalogName.ValueString(), data.Name.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Catalog Role", fmt.Sprintf("Could not update catalog role %s in catalog %s: %s", data.Name.ValueString(), data.CatalogName.ValueString(), err.Error()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.CatalogName.ValueString(), updated.Name))
	data.Name = types.StringValue(updated.Name)
	data.CreateTimestamp = types.Int64Value(updated.CreateTimestamp)
	data.LastUpdateTimestamp = types.Int64Value(updated.LastUpdateTimestamp)
	data.EntityVersion = types.Int64Value(updated.EntityVersion)

	if len(updated.Properties) > 0 {
		propsMap, _ := types.MapValueFrom(ctx, types.StringType, updated.Properties)
		data.Properties = propsMap
	} else {
		data.Properties = types.MapNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CatalogRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CatalogRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCatalogRole(ctx, data.CatalogName.ValueString(), data.Name.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Catalog Role", fmt.Sprintf("Could not delete catalog role %s in catalog %s: %s", data.Name.ValueString(), data.CatalogName.ValueString(), err.Error()))
		return
	}
}

func (r *CatalogRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected ID in format 'catalog_name/role_name', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("catalog_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}


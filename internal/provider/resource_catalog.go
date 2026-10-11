package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tranxuantruongworld/terraform-provider-polaris/internal/client"
)

var (
	_ resource.Resource                = &CatalogResource{}
	_ resource.ResourceWithConfigure   = &CatalogResource{}
	_ resource.ResourceWithImportState = &CatalogResource{}
)

func NewCatalogResource() resource.Resource {
	return &CatalogResource{}
}

type CatalogResource struct {
	client *client.Client
}

type StorageConfigModel struct {
	StorageType        types.String `tfsdk:"storage_type"`
	AllowedLocations   types.List   `tfsdk:"allowed_locations"`
	StorageName        types.String `tfsdk:"storage_name"`
	RoleArn            types.String `tfsdk:"role_arn"`
	ExternalId         types.String `tfsdk:"external_id"`
	UserArn            types.String `tfsdk:"user_arn"`
	Region             types.String `tfsdk:"region"`
	Endpoint           types.String `tfsdk:"endpoint"`
	StsEndpoint        types.String `tfsdk:"sts_endpoint"`
	StsUnavailable     types.Bool   `tfsdk:"sts_unavailable"`
	PathStyleAccess    types.Bool   `tfsdk:"path_style_access"`
	KmsUnavailable     types.Bool   `tfsdk:"kms_unavailable"`
	GcsServiceAccount  types.String `tfsdk:"gcs_service_account"`
	TenantId           types.String `tfsdk:"tenant_id"`
	MultiTenantAppName types.String `tfsdk:"multi_tenant_app_name"`
	ConsentUrl         types.String `tfsdk:"consent_url"`
	Hierarchical       types.Bool   `tfsdk:"hierarchical"`
}

type CatalogResourceModel struct {
	Name                types.String        `tfsdk:"name"`
	Type                types.String        `tfsdk:"type"`
	DefaultBaseLocation types.String        `tfsdk:"default_base_location"`
	Properties          types.Map           `tfsdk:"properties"`
	StorageConfig       *StorageConfigModel `tfsdk:"storage_config"`
	CreateTimestamp     types.Int64         `tfsdk:"create_timestamp"`
	LastUpdateTimestamp types.Int64         `tfsdk:"last_update_timestamp"`
	EntityVersion       types.Int64         `tfsdk:"entity_version"`
}

func (r *CatalogResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_catalog"
}

func (r *CatalogResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Apache Polaris catalog with storage configuration (S3, GCS, Azure, File).",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The unique name of the catalog.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Catalog type. Supported values: `INTERNAL` (default) or `EXTERNAL`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("INTERNAL"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"default_base_location": schema.StringAttribute{
				Description: "Default base storage location for tables in this catalog (e.g. `s3://bucket/path`, `gs://bucket/path`, `abfss://container@account.blob.core.windows.net/path`).",
				Required:    true,
			},
			"properties": schema.MapAttribute{
				Description: "Arbitrary key-value metadata properties for the catalog.",
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
		Blocks: map[string]schema.Block{
			"storage_config": schema.SingleNestedBlock{
				Description: "Storage configuration for cloud credential vending and location access validation.",
				Attributes: map[string]schema.Attribute{
					"storage_type": schema.StringAttribute{
						Description: "Storage provider type: `S3`, `GCS`, `AZURE`, or `FILE`.",
						Required:    true,
					},
					"allowed_locations": schema.ListAttribute{
						Description: "List of storage locations allowed for tables created in this catalog.",
						Optional:    true,
						Computed:    true,
						ElementType: types.StringType,
					},
					"storage_name": schema.StringAttribute{
						Description: "Optional name referencing a server-side storage configuration.",
						Optional:    true,
					},
					// AWS S3
					"role_arn": schema.StringAttribute{
						Description: "AWS IAM Role ARN assumed by Polaris to vend credentials for S3.",
						Optional:    true,
					},
					"external_id": schema.StringAttribute{
						Description: "Optional AWS external ID used to assume the IAM role.",
						Optional:    true,
					},
					"user_arn": schema.StringAttribute{
						Description: "AWS IAM User ARN used to assume the role.",
						Optional:    true,
					},
					"region": schema.StringAttribute{
						Description: "AWS Region where the S3 bucket is hosted.",
						Optional:    true,
					},
					"endpoint": schema.StringAttribute{
						Description: "Custom S3 endpoint URL (for S3-compatible object stores such as MinIO or Ceph).",
						Optional:    true,
					},
					"sts_endpoint": schema.StringAttribute{
						Description: "Custom AWS STS endpoint URL.",
						Optional:    true,
					},
					"sts_unavailable": schema.BoolAttribute{
						Description: "If true, avoids using STS endpoints when vending storage credentials (for non-STS S3 compatible stores).",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"path_style_access": schema.BoolAttribute{
						Description: "Whether to use path-style addressing for S3 buckets.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"kms_unavailable": schema.BoolAttribute{
						Description: "If true, instructs Polaris to avoid adding KMS key policies.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					// GCS
					"gcs_service_account": schema.StringAttribute{
						Description: "Google Cloud Storage service account email used for authentication.",
						Optional:    true,
					},
					// Azure
					"tenant_id": schema.StringAttribute{
						Description: "Azure Tenant ID where the storage account belongs.",
						Optional:    true,
					},
					"multi_tenant_app_name": schema.StringAttribute{
						Description: "Azure client application name.",
						Optional:    true,
					},
					"consent_url": schema.StringAttribute{
						Description: "URL for Azure permissions request consent.",
						Optional:    true,
					},
					"hierarchical": schema.BoolAttribute{
						Description: "If true, scopes SAS tokens for hierarchical namespace (ADLS Gen2).",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
				},
			},
		},
	}
}

func (r *CatalogResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CatalogResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CatalogResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Prepare properties
	props := make(map[string]string)
	if !data.Properties.IsNull() && !data.Properties.IsUnknown() {
		resp.Diagnostics.Append(data.Properties.ElementsAs(ctx, &props, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	props["default-base-location"] = data.DefaultBaseLocation.ValueString()

	// Prepare storage config
	storageConfig := mapStorageConfigModelToClient(ctx, data.StorageConfig, data.DefaultBaseLocation.ValueString())

	catalogReq := client.CreateCatalogRequest{
		Catalog: client.Catalog{
			Name:              data.Name.ValueString(),
			Type:              data.Type.ValueString(),
			Properties:        props,
			StorageConfigInfo: storageConfig,
		},
	}

	created, err := r.client.CreateCatalog(ctx, catalogReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Polaris Catalog", fmt.Sprintf("Could not create catalog %s: %s", data.Name.ValueString(), err.Error()))
		return
	}

	r.mapClientToState(ctx, created, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CatalogResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CatalogResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cat, err := r.client.GetCatalog(ctx, data.Name.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Polaris Catalog", fmt.Sprintf("Could not read catalog %s: %s", data.Name.ValueString(), err.Error()))
		return
	}

	r.mapClientToState(ctx, cat, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CatalogResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CatalogResourceModel

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
	props["default-base-location"] = data.DefaultBaseLocation.ValueString()

	storageConfig := mapStorageConfigModelToClient(ctx, data.StorageConfig, data.DefaultBaseLocation.ValueString())

	updateReq := client.UpdateCatalogRequest{
		CurrentEntityVersion: data.EntityVersion.ValueInt64(),
		Properties:           props,
		StorageConfigInfo:    storageConfig,
	}

	updated, err := r.client.UpdateCatalog(ctx, data.Name.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Polaris Catalog", fmt.Sprintf("Could not update catalog %s: %s", data.Name.ValueString(), err.Error()))
		return
	}

	r.mapClientToState(ctx, updated, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CatalogResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CatalogResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCatalog(ctx, data.Name.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Polaris Catalog", fmt.Sprintf("Could not delete catalog %s: %s", data.Name.ValueString(), err.Error()))
		return
	}
}

func (r *CatalogResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (r *CatalogResource) mapClientToState(ctx context.Context, cat *client.Catalog, data *CatalogResourceModel) {
	data.Name = types.StringValue(cat.Name)
	data.Type = types.StringValue(cat.Type)
	data.CreateTimestamp = types.Int64Value(cat.CreateTimestamp)
	data.LastUpdateTimestamp = types.Int64Value(cat.LastUpdateTimestamp)
	data.EntityVersion = types.Int64Value(cat.EntityVersion)

	if loc, ok := cat.Properties["default-base-location"]; ok {
		data.DefaultBaseLocation = types.StringValue(loc)
	}

	// Properties
	if len(cat.Properties) > 0 {
		propsMap, _ := types.MapValueFrom(ctx, types.StringType, cat.Properties)
		data.Properties = propsMap
	} else {
		data.Properties = types.MapNull(types.StringType)
	}

	// StorageConfig
	if cat.StorageConfigInfo != nil {
		if data.StorageConfig == nil {
			data.StorageConfig = &StorageConfigModel{}
		}
		sc := cat.StorageConfigInfo
		data.StorageConfig.StorageType = types.StringValue(sc.StorageType)
		if sc.StorageName != "" {
			data.StorageConfig.StorageName = types.StringValue(sc.StorageName)
		}

		if len(sc.AllowedLocations) > 0 {
			locs, _ := types.ListValueFrom(ctx, types.StringType, sc.AllowedLocations)
			data.StorageConfig.AllowedLocations = locs
		} else {
			data.StorageConfig.AllowedLocations = types.ListNull(types.StringType)
		}

		if sc.RoleArn != "" {
			data.StorageConfig.RoleArn = types.StringValue(sc.RoleArn)
		}
		if sc.ExternalId != "" {
			data.StorageConfig.ExternalId = types.StringValue(sc.ExternalId)
		}
		if sc.UserArn != "" {
			data.StorageConfig.UserArn = types.StringValue(sc.UserArn)
		}
		if sc.Region != "" {
			data.StorageConfig.Region = types.StringValue(sc.Region)
		}
		if sc.Endpoint != "" {
			data.StorageConfig.Endpoint = types.StringValue(sc.Endpoint)
		}
		if sc.StsEndpoint != "" {
			data.StorageConfig.StsEndpoint = types.StringValue(sc.StsEndpoint)
		}
		if sc.StsUnavailable != nil {
			data.StorageConfig.StsUnavailable = types.BoolValue(*sc.StsUnavailable)
		}
		if sc.PathStyleAccess != nil {
			data.StorageConfig.PathStyleAccess = types.BoolValue(*sc.PathStyleAccess)
		}
		if sc.KmsUnavailable != nil {
			data.StorageConfig.KmsUnavailable = types.BoolValue(*sc.KmsUnavailable)
		}
		if sc.GcsServiceAccount != "" {
			data.StorageConfig.GcsServiceAccount = types.StringValue(sc.GcsServiceAccount)
		}
		if sc.TenantId != "" {
			data.StorageConfig.TenantId = types.StringValue(sc.TenantId)
		}
		if sc.MultiTenantAppName != "" {
			data.StorageConfig.MultiTenantAppName = types.StringValue(sc.MultiTenantAppName)
		}
		if sc.ConsentUrl != "" {
			data.StorageConfig.ConsentUrl = types.StringValue(sc.ConsentUrl)
		}
		if sc.Hierarchical != nil {
			data.StorageConfig.Hierarchical = types.BoolValue(*sc.Hierarchical)
		}
	}
}

func mapStorageConfigModelToClient(ctx context.Context, m *StorageConfigModel, defaultLocation string) *client.StorageConfigInfo {
	if m == nil {
		return nil
	}

	info := &client.StorageConfigInfo{
		StorageType: m.StorageType.ValueString(),
	}

	if !m.StorageName.IsNull() {
		info.StorageName = m.StorageName.ValueString()
	}

	if !m.AllowedLocations.IsNull() && !m.AllowedLocations.IsUnknown() {
		var locs []string
		_ = m.AllowedLocations.ElementsAs(ctx, &locs, false)
		info.AllowedLocations = locs
	} else if defaultLocation != "" {
		info.AllowedLocations = []string{defaultLocation}
	}

	if !m.RoleArn.IsNull() {
		info.RoleArn = m.RoleArn.ValueString()
	}
	if !m.ExternalId.IsNull() {
		info.ExternalId = m.ExternalId.ValueString()
	}
	if !m.UserArn.IsNull() {
		info.UserArn = m.UserArn.ValueString()
	}
	if !m.Region.IsNull() {
		info.Region = m.Region.ValueString()
	}
	if !m.Endpoint.IsNull() {
		info.Endpoint = m.Endpoint.ValueString()
	}
	if !m.StsEndpoint.IsNull() {
		info.StsEndpoint = m.StsEndpoint.ValueString()
	}
	if !m.StsUnavailable.IsNull() {
		v := m.StsUnavailable.ValueBool()
		info.StsUnavailable = &v
	}
	if !m.PathStyleAccess.IsNull() {
		v := m.PathStyleAccess.ValueBool()
		info.PathStyleAccess = &v
	}
	if !m.KmsUnavailable.IsNull() {
		v := m.KmsUnavailable.ValueBool()
		info.KmsUnavailable = &v
	}

	if !m.GcsServiceAccount.IsNull() {
		info.GcsServiceAccount = m.GcsServiceAccount.ValueString()
	}

	if !m.TenantId.IsNull() {
		info.TenantId = m.TenantId.ValueString()
	}
	if !m.MultiTenantAppName.IsNull() {
		info.MultiTenantAppName = m.MultiTenantAppName.ValueString()
	}
	if !m.ConsentUrl.IsNull() {
		info.ConsentUrl = m.ConsentUrl.ValueString()
	}
	if !m.Hierarchical.IsNull() {
		v := m.Hierarchical.ValueBool()
		info.Hierarchical = &v
	}

	return info
}

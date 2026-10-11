package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tranxuantruongworld/terraform-provider-polaris/internal/client"
)

var (
	_ resource.Resource                = &PrincipalRoleResource{}
	_ resource.ResourceWithConfigure   = &PrincipalRoleResource{}
	_ resource.ResourceWithImportState = &PrincipalRoleResource{}
)

func NewPrincipalRoleResource() resource.Resource {
	return &PrincipalRoleResource{}
}

type PrincipalRoleResource struct {
	client *client.Client
}

type PrincipalRoleResourceModel struct {
	Name                types.String `tfsdk:"name"`
	Federated           types.Bool   `tfsdk:"federated"`
	Properties          types.Map    `tfsdk:"properties"`
	CreateTimestamp     types.Int64  `tfsdk:"create_timestamp"`
	LastUpdateTimestamp types.Int64  `tfsdk:"last_update_timestamp"`
	EntityVersion       types.Int64  `tfsdk:"entity_version"`
}

func (r *PrincipalRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_principal_role"
}

func (r *PrincipalRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Apache Polaris principal role for RBAC assignment to principals.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the principal role.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"federated": schema.BoolAttribute{
				Description: "Whether the role is managed by an external identity provider.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"properties": schema.MapAttribute{
				Description: "Custom properties associated with this principal role.",
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

func (r *PrincipalRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PrincipalRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PrincipalRoleResourceModel

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

	createReq := client.CreatePrincipalRoleRequest{
		PrincipalRole: client.PrincipalRole{
			Name:       data.Name.ValueString(),
			Federated:  data.Federated.ValueBool(),
			Properties: props,
		},
	}

	created, err := r.client.CreatePrincipalRole(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Principal Role", fmt.Sprintf("Could not create principal role %s: %s", data.Name.ValueString(), err.Error()))
		return
	}

	data.Name = types.StringValue(created.Name)
	data.Federated = types.BoolValue(created.Federated)
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

func (r *PrincipalRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PrincipalRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.GetPrincipalRole(ctx, data.Name.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Principal Role", fmt.Sprintf("Could not read principal role %s: %s", data.Name.ValueString(), err.Error()))
		return
	}

	data.Name = types.StringValue(role.Name)
	data.Federated = types.BoolValue(role.Federated)
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

func (r *PrincipalRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PrincipalRoleResourceModel

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

	updateReq := client.UpdatePrincipalRoleRequest{
		CurrentEntityVersion: data.EntityVersion.ValueInt64(),
		Properties:           props,
	}

	updated, err := r.client.UpdatePrincipalRole(ctx, data.Name.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Principal Role", fmt.Sprintf("Could not update principal role %s: %s", data.Name.ValueString(), err.Error()))
		return
	}

	data.Name = types.StringValue(updated.Name)
	data.Federated = types.BoolValue(updated.Federated)
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

func (r *PrincipalRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PrincipalRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePrincipalRole(ctx, data.Name.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Principal Role", fmt.Sprintf("Could not delete principal role %s: %s", data.Name.ValueString(), err.Error()))
		return
	}
}

func (r *PrincipalRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}


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
	_ resource.Resource                = &CatalogRoleAssignmentResource{}
	_ resource.ResourceWithConfigure   = &CatalogRoleAssignmentResource{}
	_ resource.ResourceWithImportState = &CatalogRoleAssignmentResource{}
)

func NewCatalogRoleAssignmentResource() resource.Resource {
	return &CatalogRoleAssignmentResource{}
}

type CatalogRoleAssignmentResource struct {
	client *client.Client
}

type CatalogRoleAssignmentResourceModel struct {
	ID                types.String `tfsdk:"id"`
	PrincipalRoleName types.String `tfsdk:"principal_role_name"`
	CatalogName       types.String `tfsdk:"catalog_name"`
	CatalogRoleName   types.String `tfsdk:"catalog_role_name"`
}

func (r *CatalogRoleAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_catalog_role_assignment"
}

func (r *CatalogRoleAssignmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns a Polaris catalog role to a principal role, granting catalog-level permissions to principals holding that principal role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite identifier of the assignment in format `principal_role_name/catalog_name/catalog_role_name`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"principal_role_name": schema.StringAttribute{
				Description: "The name of the principal role receiving the catalog role.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"catalog_name": schema.StringAttribute{
				Description: "The catalog containing the role.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"catalog_role_name": schema.StringAttribute{
				Description: "The name of the catalog role to assign.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *CatalogRoleAssignmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CatalogRoleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CatalogRoleAssignmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.AssignCatalogRoleToPrincipalRole(
		ctx,
		data.PrincipalRoleName.ValueString(),
		data.CatalogName.ValueString(),
		data.CatalogRoleName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Assigning Catalog Role",
			fmt.Sprintf("Could not assign catalog role %s (catalog %s) to principal role %s: %s",
				data.CatalogRoleName.ValueString(), data.CatalogName.ValueString(), data.PrincipalRoleName.ValueString(), err.Error()),
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s",
		data.PrincipalRoleName.ValueString(), data.CatalogName.ValueString(), data.CatalogRoleName.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CatalogRoleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CatalogRoleAssignmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, err := r.client.ListCatalogRolesForPrincipalRole(
		ctx,
		data.PrincipalRoleName.ValueString(),
		data.CatalogName.ValueString(),
	)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Catalog Role Assignment",
			fmt.Sprintf("Could not list catalog roles for principal role %s: %s", data.PrincipalRoleName.ValueString(), err.Error()),
		)
		return
	}

	found := false
	for _, role := range roles {
		if role.Name == data.CatalogRoleName.ValueString() {
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CatalogRoleAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes RequireReplace
}

func (r *CatalogRoleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CatalogRoleAssignmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.UnassignCatalogRoleFromPrincipalRole(
		ctx,
		data.PrincipalRoleName.ValueString(),
		data.CatalogName.ValueString(),
		data.CatalogRoleName.ValueString(),
	)
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Error Unassigning Catalog Role",
			fmt.Sprintf("Could not unassign catalog role %s from principal role %s: %s",
				data.CatalogRoleName.ValueString(), data.PrincipalRoleName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *CatalogRoleAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected ID in format 'principal_role_name/catalog_name/catalog_role_name', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("principal_role_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("catalog_name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("catalog_role_name"), parts[2])...)
}


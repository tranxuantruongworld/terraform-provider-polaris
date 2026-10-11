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
	_ resource.Resource                = &PrincipalRoleAssignmentResource{}
	_ resource.ResourceWithConfigure   = &PrincipalRoleAssignmentResource{}
	_ resource.ResourceWithImportState = &PrincipalRoleAssignmentResource{}
)

func NewPrincipalRoleAssignmentResource() resource.Resource {
	return &PrincipalRoleAssignmentResource{}
}

type PrincipalRoleAssignmentResource struct {
	client *client.Client
}

type PrincipalRoleAssignmentResourceModel struct {
	ID                types.String `tfsdk:"id"`
	PrincipalName     types.String `tfsdk:"principal_name"`
	PrincipalRoleName types.String `tfsdk:"principal_role_name"`
}

func (r *PrincipalRoleAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_principal_role_assignment"
}

func (r *PrincipalRoleAssignmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns a Polaris principal role to a principal identity.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite identifier of the assignment in format `principal_name/principal_role_name`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"principal_name": schema.StringAttribute{
				Description: "The name of the principal receiving the role.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"principal_role_name": schema.StringAttribute{
				Description: "The name of the principal role to assign.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *PrincipalRoleAssignmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PrincipalRoleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PrincipalRoleAssignmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.AssignPrincipalRole(ctx, data.PrincipalName.ValueString(), data.PrincipalRoleName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Assigning Principal Role",
			fmt.Sprintf("Could not assign principal role %s to principal %s: %s", data.PrincipalRoleName.ValueString(), data.PrincipalName.ValueString(), err.Error()),
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.PrincipalName.ValueString(), data.PrincipalRoleName.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrincipalRoleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PrincipalRoleAssignmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, err := r.client.ListPrincipalRolesForPrincipal(ctx, data.PrincipalName.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Principal Role Assignment",
			fmt.Sprintf("Could not list roles for principal %s: %s", data.PrincipalName.ValueString(), err.Error()),
		)
		return
	}

	found := false
	for _, role := range roles {
		if role.Name == data.PrincipalRoleName.ValueString() {
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

func (r *PrincipalRoleAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// RequiresReplace on all attributes
}

func (r *PrincipalRoleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PrincipalRoleAssignmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.UnassignPrincipalRole(ctx, data.PrincipalName.ValueString(), data.PrincipalRoleName.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Error Unassigning Principal Role",
			fmt.Sprintf("Could not unassign principal role %s from principal %s: %s", data.PrincipalRoleName.ValueString(), data.PrincipalName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *PrincipalRoleAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected ID in format 'principal_name/principal_role_name', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("principal_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("principal_role_name"), parts[1])...)
}


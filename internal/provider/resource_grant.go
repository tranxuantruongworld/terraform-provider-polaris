package provider

import (
	"context"
	"fmt"
	"reflect"
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
	_ resource.Resource                = &GrantResource{}
	_ resource.ResourceWithConfigure   = &GrantResource{}
	_ resource.ResourceWithImportState = &GrantResource{}
)

func NewGrantResource() resource.Resource {
	return &GrantResource{}
}

type GrantResource struct {
	client *client.Client
}

type GrantResourceModel struct {
	ID              types.String `tfsdk:"id"`
	CatalogName     types.String `tfsdk:"catalog_name"`
	CatalogRoleName types.String `tfsdk:"catalog_role_name"`
	Type            types.String `tfsdk:"type"`
	Privilege       types.String `tfsdk:"privilege"`
	Namespace       types.List   `tfsdk:"namespace"`
	TableName       types.String `tfsdk:"table_name"`
	ViewName        types.String `tfsdk:"view_name"`
}

func (r *GrantResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_grant"
}

func (r *GrantResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a securable privilege grant (catalog, namespace, table, or view) to an Apache Polaris catalog role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the grant.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"catalog_name": schema.StringAttribute{
				Description: "The catalog containing the role and securable resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"catalog_role_name": schema.StringAttribute{
				Description: "The catalog role receiving the grant.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The securable type: `catalog`, `namespace`, `table`, or `view`.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"privilege": schema.StringAttribute{
				Description: "The privilege to grant (e.g. `CATALOG_MANAGE_ACCESS`, `TABLE_READ_DATA`, `TABLE_WRITE_DATA`, `NAMESPACE_CREATE`).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"namespace": schema.ListAttribute{
				Description: "Namespace hierarchy for namespace, table, or view grants (e.g. `[\"db1\"]` or `[\"db1\", \"schema2\"]`).",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					// replacement on changes
				},
			},
			"table_name": schema.StringAttribute{
				Description: "Table name for table grants.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"view_name": schema.StringAttribute{
				Description: "View name for view grants.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *GrantResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GrantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GrantResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	grant := r.modelToGrant(ctx, &data)

	err := r.client.AddGrant(ctx, data.CatalogName.ValueString(), data.CatalogRoleName.ValueString(), grant)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Polaris Grant",
			fmt.Sprintf("Could not add grant %s (%s) to catalog role %s: %s",
				data.Privilege.ValueString(), data.Type.ValueString(), data.CatalogRoleName.ValueString(), err.Error()),
		)
		return
	}

	data.ID = types.StringValue(r.computeGrantID(&data))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GrantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GrantResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	grants, err := r.client.ListGrants(ctx, data.CatalogName.ValueString(), data.CatalogRoleName.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Polaris Grant",
			fmt.Sprintf("Could not list grants for catalog role %s: %s", data.CatalogRoleName.ValueString(), err.Error()),
		)
		return
	}

	target := r.modelToGrant(ctx, &data)
	found := false
	for _, g := range grants {
		if r.grantsEqual(g, target) {
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

func (r *GrantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes RequireReplace
}

func (r *GrantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GrantResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	grant := r.modelToGrant(ctx, &data)
	err := r.client.RevokeGrant(ctx, data.CatalogName.ValueString(), data.CatalogRoleName.ValueString(), grant)
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Error Revoking Polaris Grant",
			fmt.Sprintf("Could not revoke grant %s from catalog role %s: %s",
				data.Privilege.ValueString(), data.CatalogRoleName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *GrantResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) < 4 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected ID with at least 4 segments: 'catalog_name/catalog_role_name/type/privilege', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("catalog_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("catalog_role_name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("privilege"), parts[3])...)
}

func (r *GrantResource) modelToGrant(ctx context.Context, data *GrantResourceModel) client.GrantResource {
	grant := client.GrantResource{
		Type:      strings.ToLower(data.Type.ValueString()),
		Privilege: data.Privilege.ValueString(),
	}

	if !data.Namespace.IsNull() && !data.Namespace.IsUnknown() {
		var ns []string
		_ = data.Namespace.ElementsAs(ctx, &ns, false)
		grant.Namespace = ns
	}
	if !data.TableName.IsNull() && !data.TableName.IsUnknown() {
		grant.TableName = data.TableName.ValueString()
	}
	if !data.ViewName.IsNull() && !data.ViewName.IsUnknown() {
		grant.ViewName = data.ViewName.ValueString()
	}

	return grant
}

func (r *GrantResource) grantsEqual(a, b client.GrantResource) bool {
	if !strings.EqualFold(a.Type, b.Type) || a.Privilege != b.Privilege {
		return false
	}
	if !reflect.DeepEqual(a.Namespace, b.Namespace) {
		return false
	}
	if a.TableName != b.TableName || a.ViewName != b.ViewName {
		return false
	}
	return true
}

func (r *GrantResource) computeGrantID(data *GrantResourceModel) string {
	parts := []string{
		data.CatalogName.ValueString(),
		data.CatalogRoleName.ValueString(),
		strings.ToLower(data.Type.ValueString()),
		data.Privilege.ValueString(),
	}
	if !data.TableName.IsNull() {
		parts = append(parts, data.TableName.ValueString())
	}
	if !data.ViewName.IsNull() {
		parts = append(parts, data.ViewName.ValueString())
	}
	return strings.Join(parts, "/")
}

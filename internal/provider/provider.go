package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tranxuantruongworld/terraform-provider-polaris/internal/client"
)

// Ensure PolarisProvider satisfies provider.Provider interface.
var _ provider.Provider = &PolarisProvider{}

// PolarisProvider defines the provider implementation.
type PolarisProvider struct {
	version string
}

// PolarisProviderModel describes the provider data model.
type PolarisProviderModel struct {
	ServerURL    types.String `tfsdk:"server_url"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	Realm        types.String `tfsdk:"realm"`
	Token        types.String `tfsdk:"token"`
}

// New returns a new provider implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PolarisProvider{
			version: version,
		}
	}
}

// Metadata returns provider metadata.
func (p *PolarisProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "polaris"
	resp.Version = p.version
}

// Schema defines the provider-level schema.
func (p *PolarisProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for Apache Polaris (incubating), an open-source Iceberg REST catalog with multi-tenant RBAC and credential vending.",
		Attributes: map[string]schema.Attribute{
			"server_url": schema.StringAttribute{
				Description: "The base URL of the Apache Polaris service (e.g. `http://localhost:8181`). May also be set via the `POLARIS_SERVER_URL` environment variable.",
				Optional:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "OAuth2 client ID for Polaris authentication. May also be set via the `POLARIS_CLIENT_ID` environment variable.",
				Optional:    true,
			},
			"client_secret": schema.StringAttribute{
				Description: "OAuth2 client secret for Polaris authentication. May also be set via the `POLARIS_CLIENT_SECRET` environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"realm": schema.StringAttribute{
				Description: "The Polaris multi-tenant realm context (header `Polaris-Realm`). Defaults to `default-realm`. May also be set via the `POLARIS_REALM` environment variable.",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "A static Bearer token for Polaris authentication, bypassing OAuth2 client credentials grant. May also be set via the `POLARIS_TOKEN` environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure prepares the Polaris API client for resources and data sources.
func (p *PolarisProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config PolarisProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverURL := os.Getenv("POLARIS_SERVER_URL")
	if !config.ServerURL.IsNull() {
		serverURL = config.ServerURL.ValueString()
	}

	clientID := os.Getenv("POLARIS_CLIENT_ID")
	if !config.ClientID.IsNull() {
		clientID = config.ClientID.ValueString()
	}

	clientSecret := os.Getenv("POLARIS_CLIENT_SECRET")
	if !config.ClientSecret.IsNull() {
		clientSecret = config.ClientSecret.ValueString()
	}

	realm := os.Getenv("POLARIS_REALM")
	if !config.Realm.IsNull() {
		realm = config.Realm.ValueString()
	}

	token := os.Getenv("POLARIS_TOKEN")
	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}

	if serverURL == "" {
		resp.Diagnostics.AddError(
			"Missing Polaris Server URL",
			"The provider cannot configure the Polaris client because server_url was not provided. "+
				"Set the server_url attribute in the provider block or configure the POLARIS_SERVER_URL environment variable.",
		)
		return
	}

	c, err := client.NewClient(client.Config{
		BaseURL:      serverURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Realm:        realm,
		Token:        token,
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Polaris client", err.Error())
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

// Resources defines the provider resources.
func (p *PolarisProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCatalogResource,
		NewPrincipalResource,
		NewPrincipalRoleResource,
		NewPrincipalRoleAssignmentResource,
		NewCatalogRoleResource,
		NewCatalogRoleAssignmentResource,
		NewGrantResource,
	}
}

// DataSources defines the provider data sources.
func (p *PolarisProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewCatalogDataSource,
		NewPrincipalDataSource,
		NewPrincipalRoleDataSource,
		NewCatalogRoleDataSource,
	}
}

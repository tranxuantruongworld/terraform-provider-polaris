package client

// StorageConfigInfo defines cloud storage configuration for Polaris catalogs.
type StorageConfigInfo struct {
	StorageType      string   `json:"storageType"` // "S3", "GCS", "AZURE", "FILE"
	AllowedLocations []string `json:"allowedLocations,omitempty"`
	StorageName      string   `json:"storageName,omitempty"`

	// S3 specific fields
	RoleArn         string   `json:"roleArn,omitempty"`
	ExternalId      string   `json:"externalId,omitempty"`
	UserArn         string   `json:"userArn,omitempty"`
	Region          string   `json:"region,omitempty"`
	Endpoint        string   `json:"endpoint,omitempty"`
	StsEndpoint     string   `json:"stsEndpoint,omitempty"`
	StsUnavailable  *bool    `json:"stsUnavailable,omitempty"`
	PathStyleAccess *bool    `json:"pathStyleAccess,omitempty"`
	KmsUnavailable  *bool    `json:"kmsUnavailable,omitempty"`

	// GCS specific fields
	GcsServiceAccount string `json:"gcsServiceAccount,omitempty"`

	// Azure specific fields
	TenantId           string `json:"tenantId,omitempty"`
	MultiTenantAppName string `json:"multiTenantAppName,omitempty"`
	ConsentUrl         string `json:"consentUrl,omitempty"`
	Hierarchical       *bool  `json:"hierarchical,omitempty"`
}

// Catalog represents an Apache Polaris catalog.
type Catalog struct {
	Name                string             `json:"name"`
	Type                string             `json:"type"` // "INTERNAL" or "EXTERNAL"
	Properties          map[string]string  `json:"properties"`
	StorageConfigInfo   *StorageConfigInfo `json:"storageConfigInfo,omitempty"`
	CreateTimestamp     int64              `json:"createTimestamp,omitempty"`
	LastUpdateTimestamp int64              `json:"lastUpdateTimestamp,omitempty"`
	EntityVersion       int64              `json:"entityVersion,omitempty"`
}

// CreateCatalogRequest wraps catalog creation payload.
type CreateCatalogRequest struct {
	Catalog Catalog `json:"catalog"`
}

// UpdateCatalogRequest represents catalog update payload.
type UpdateCatalogRequest struct {
	CurrentEntityVersion int64              `json:"currentEntityVersion"`
	Properties           map[string]string  `json:"properties,omitempty"`
	StorageConfigInfo    *StorageConfigInfo `json:"storageConfigInfo,omitempty"`
}

// CatalogListResponse represents list of catalogs response.
type CatalogListResponse struct {
	Catalogs []Catalog `json:"catalogs"`
}

// Principal represents a security identity in Polaris.
type Principal struct {
	Name                string            `json:"name"`
	ClientID            string            `json:"clientId,omitempty"`
	Properties          map[string]string `json:"properties,omitempty"`
	CreateTimestamp     int64             `json:"createTimestamp,omitempty"`
	LastUpdateTimestamp int64             `json:"lastUpdateTimestamp,omitempty"`
	EntityVersion       int64             `json:"entityVersion,omitempty"`
}

// CreatePrincipalRequest wraps principal creation payload.
type CreatePrincipalRequest struct {
	Principal                  Principal `json:"principal"`
	CredentialRotationRequired bool      `json:"credentialRotationRequired,omitempty"`
}

// PrincipalCredentials represents credentials returned upon principal creation or rotation.
type PrincipalCredentials struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

// PrincipalWithCredentials represents response containing generated credentials.
type PrincipalWithCredentials struct {
	Principal   Principal            `json:"principal"`
	Credentials PrincipalCredentials `json:"credentials"`
}

// UpdatePrincipalRequest represents principal update payload.
type UpdatePrincipalRequest struct {
	CurrentEntityVersion int64             `json:"currentEntityVersion"`
	Properties           map[string]string `json:"properties"`
}

// PrincipalListResponse represents response from listing principals.
type PrincipalListResponse struct {
	Principals []Principal `json:"principals"`
}

// PrincipalRole represents a role that can be granted to principals.
type PrincipalRole struct {
	Name                string            `json:"name"`
	Federated           bool              `json:"federated,omitempty"`
	Properties          map[string]string `json:"properties,omitempty"`
	CreateTimestamp     int64             `json:"createTimestamp,omitempty"`
	LastUpdateTimestamp int64             `json:"lastUpdateTimestamp,omitempty"`
	EntityVersion       int64             `json:"entityVersion,omitempty"`
}

// CreatePrincipalRoleRequest wraps principal role creation payload.
type CreatePrincipalRoleRequest struct {
	PrincipalRole PrincipalRole `json:"principalRole"`
}

// GrantPrincipalRoleRequest wraps assigning principal role payload.
type GrantPrincipalRoleRequest struct {
	PrincipalRole PrincipalRole `json:"principalRole"`
}

// UpdatePrincipalRoleRequest represents principal role update payload.
type UpdatePrincipalRoleRequest struct {
	CurrentEntityVersion int64             `json:"currentEntityVersion"`
	Properties           map[string]string `json:"properties"`
}

// PrincipalRoleListResponse represents list of principal roles.
type PrincipalRoleListResponse struct {
	Roles []PrincipalRole `json:"roles"`
}

// CatalogRole represents a role scoped to a catalog.
type CatalogRole struct {
	Name                string            `json:"name"`
	Properties          map[string]string `json:"properties,omitempty"`
	CreateTimestamp     int64             `json:"createTimestamp,omitempty"`
	LastUpdateTimestamp int64             `json:"lastUpdateTimestamp,omitempty"`
	EntityVersion       int64             `json:"entityVersion,omitempty"`
}

// CreateCatalogRoleRequest wraps catalog role creation payload.
type CreateCatalogRoleRequest struct {
	CatalogRole CatalogRole `json:"catalogRole"`
}

// GrantCatalogRoleRequest wraps assigning catalog role payload.
type GrantCatalogRoleRequest struct {
	CatalogRole CatalogRole `json:"catalogRole"`
}

// UpdateCatalogRoleRequest represents catalog role update payload.
type UpdateCatalogRoleRequest struct {
	CurrentEntityVersion int64             `json:"currentEntityVersion"`
	Properties           map[string]string `json:"properties"`
}

// CatalogRoleListResponse represents list of catalog roles.
type CatalogRoleListResponse struct {
	Roles []CatalogRole `json:"roles"`
}

// GrantResource represents a grant on a catalog role.
type GrantResource struct {
	Type      string   `json:"type"`                // "catalog", "namespace", "table", "view"
	Privilege string   `json:"privilege"`           // e.g. "CATALOG_MANAGE_ACCESS", "TABLE_READ_DATA"
	Namespace []string `json:"namespace,omitempty"` // For namespace, table, view
	TableName string   `json:"tableName,omitempty"` // For table
	ViewName  string   `json:"viewName,omitempty"`  // For view
}

// AddGrantRequest represents payload to grant privileges.
type AddGrantRequest struct {
	Grant GrantResource `json:"grant"`
}

// RevokeGrantRequest represents payload to revoke privileges.
type RevokeGrantRequest struct {
	Grant GrantResource `json:"grant"`
}

// GrantResourcesResponse represents list of grants on a catalog role.
type GrantResourcesResponse struct {
	Grants []GrantResource `json:"grants"`
}

// OAuthTokenResponse represents response from /api/catalog/v1/oauth/tokens.
type OAuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

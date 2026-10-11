package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	// ErrNotFound is returned when Polaris API returns HTTP 404.
	ErrNotFound = errors.New("resource not found")
)

// IsNotFound checks if the error indicates a 404 Not Found.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrNotFound) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found") ||
		strings.Contains(err.Error(), "404")
}

// Config holds Polaris client configuration.
type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	Realm        string
	Token        string
	HTTPClient   *http.Client
}

// Client represents a client for Apache Polaris REST API.
type Client struct {
	baseURL      string
	clientID     string
	clientSecret string
	realm        string
	httpClient   *http.Client

	tokenMu     sync.RWMutex
	token       string
	tokenExpiry time.Time
}

// NewClient creates a new Apache Polaris API client.
func NewClient(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("baseURL is required")
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	realm := cfg.Realm
	if realm == "" {
		realm = "default-realm"
	}

	c := &Client{
		baseURL:      strings.TrimRight(cfg.BaseURL, "/"),
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		realm:        realm,
		httpClient:   httpClient,
		token:        cfg.Token,
	}

	return c, nil
}

func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	c.tokenMu.RLock()
	if c.token != "" && (c.clientID == "" || time.Now().Before(c.tokenExpiry)) {
		token := c.token
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	// If we don't have client credentials, return the current token
	if c.clientID == "" || c.clientSecret == "" {
		return c.token, nil
	}

	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Double check inside write lock
	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return c.token, nil
	}

	// Fetch token from /api/catalog/v1/oauth/tokens
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("scope", "PRINCIPAL_ROLE:ALL")

	tokenURL := fmt.Sprintf("%s/api/catalog/v1/oauth/tokens", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create oauth token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	if c.realm != "" {
		req.Header.Set("Polaris-Realm", c.realm)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch oauth token: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("oauth token request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResp OAuthTokenResponse
	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse oauth token response: %w", err)
	}

	c.token = tokenResp.AccessToken
	if tokenResp.ExpiresIn > 60 {
		c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)
	} else {
		c.tokenExpiry = time.Now().Add(30 * time.Minute)
	}

	return c.token, nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, in any, out any) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}

	var bodyReader io.Reader
	if in != nil {
		payload, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	reqURL := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	if c.realm != "" {
		req.Header.Set("Polaris-Realm", c.realm)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("api request %s %s returned status %d: %s", method, path, resp.StatusCode, string(bodyBytes))
	}

	if out != nil && len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, out); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// --- Catalogs ---

// CreateCatalog creates a new catalog.
func (c *Client) CreateCatalog(ctx context.Context, req CreateCatalogRequest) (*Catalog, error) {
	var resp Catalog
	err := c.doRequest(ctx, http.MethodPost, "/api/management/v1/catalogs", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetCatalog retrieves a catalog by name.
func (c *Client) GetCatalog(ctx context.Context, name string) (*Catalog, error) {
	var resp Catalog
	path := fmt.Sprintf("/api/management/v1/catalogs/%s", url.PathEscape(name))
	err := c.doRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateCatalog updates an existing catalog.
func (c *Client) UpdateCatalog(ctx context.Context, name string, req UpdateCatalogRequest) (*Catalog, error) {
	var resp Catalog
	path := fmt.Sprintf("/api/management/v1/catalogs/%s", url.PathEscape(name))
	err := c.doRequest(ctx, http.MethodPut, path, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteCatalog deletes an existing catalog.
func (c *Client) DeleteCatalog(ctx context.Context, name string) error {
	path := fmt.Sprintf("/api/management/v1/catalogs/%s", url.PathEscape(name))
	return c.doRequest(ctx, http.MethodDelete, path, nil, nil)
}

// ListCatalogs lists all catalogs.
func (c *Client) ListCatalogs(ctx context.Context) ([]Catalog, error) {
	var resp CatalogListResponse
	err := c.doRequest(ctx, http.MethodGet, "/api/management/v1/catalogs", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Catalogs, nil
}

// --- Principals ---

// CreatePrincipal creates a new principal and returns its credentials.
func (c *Client) CreatePrincipal(ctx context.Context, req CreatePrincipalRequest) (*PrincipalWithCredentials, error) {
	var resp PrincipalWithCredentials
	err := c.doRequest(ctx, http.MethodPost, "/api/management/v1/principals", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetPrincipal retrieves a principal by name.
func (c *Client) GetPrincipal(ctx context.Context, name string) (*Principal, error) {
	var resp Principal
	path := fmt.Sprintf("/api/management/v1/principals/%s", url.PathEscape(name))
	err := c.doRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdatePrincipal updates an existing principal.
func (c *Client) UpdatePrincipal(ctx context.Context, name string, req UpdatePrincipalRequest) (*Principal, error) {
	var resp Principal
	path := fmt.Sprintf("/api/management/v1/principals/%s", url.PathEscape(name))
	err := c.doRequest(ctx, http.MethodPut, path, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeletePrincipal deletes an existing principal.
func (c *Client) DeletePrincipal(ctx context.Context, name string) error {
	path := fmt.Sprintf("/api/management/v1/principals/%s", url.PathEscape(name))
	return c.doRequest(ctx, http.MethodDelete, path, nil, nil)
}

// ListPrincipals lists all principals.
func (c *Client) ListPrincipals(ctx context.Context) ([]Principal, error) {
	var resp PrincipalListResponse
	err := c.doRequest(ctx, http.MethodGet, "/api/management/v1/principals", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Principals, nil
}

// RotatePrincipalCredentials rotates principal credentials.
func (c *Client) RotatePrincipalCredentials(ctx context.Context, name string) (*PrincipalWithCredentials, error) {
	var resp PrincipalWithCredentials
	path := fmt.Sprintf("/api/management/v1/principals/%s/rotate", url.PathEscape(name))
	err := c.doRequest(ctx, http.MethodPost, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// --- Principal Roles ---

// CreatePrincipalRole creates a new principal role.
func (c *Client) CreatePrincipalRole(ctx context.Context, req CreatePrincipalRoleRequest) (*PrincipalRole, error) {
	var resp PrincipalRole
	err := c.doRequest(ctx, http.MethodPost, "/api/management/v1/principal-roles", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetPrincipalRole retrieves a principal role by name.
func (c *Client) GetPrincipalRole(ctx context.Context, name string) (*PrincipalRole, error) {
	var resp PrincipalRole
	path := fmt.Sprintf("/api/management/v1/principal-roles/%s", url.PathEscape(name))
	err := c.doRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdatePrincipalRole updates a principal role.
func (c *Client) UpdatePrincipalRole(ctx context.Context, name string, req UpdatePrincipalRoleRequest) (*PrincipalRole, error) {
	var resp PrincipalRole
	path := fmt.Sprintf("/api/management/v1/principal-roles/%s", url.PathEscape(name))
	err := c.doRequest(ctx, http.MethodPut, path, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeletePrincipalRole deletes a principal role.
func (c *Client) DeletePrincipalRole(ctx context.Context, name string) error {
	path := fmt.Sprintf("/api/management/v1/principal-roles/%s", url.PathEscape(name))
	return c.doRequest(ctx, http.MethodDelete, path, nil, nil)
}

// ListPrincipalRoles lists all principal roles.
func (c *Client) ListPrincipalRoles(ctx context.Context) ([]PrincipalRole, error) {
	var resp PrincipalRoleListResponse
	err := c.doRequest(ctx, http.MethodGet, "/api/management/v1/principal-roles", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Roles, nil
}

// AssignPrincipalRole assigns a principal role to a principal.
func (c *Client) AssignPrincipalRole(ctx context.Context, principalName string, roleName string) error {
	req := GrantPrincipalRoleRequest{
		PrincipalRole: PrincipalRole{
			Name: roleName,
		},
	}
	path := fmt.Sprintf("/api/management/v1/principals/%s/principal-roles", url.PathEscape(principalName))
	return c.doRequest(ctx, http.MethodPut, path, req, nil)
}

// UnassignPrincipalRole unassigns a principal role from a principal.
func (c *Client) UnassignPrincipalRole(ctx context.Context, principalName string, roleName string) error {
	path := fmt.Sprintf("/api/management/v1/principals/%s/principal-roles/%s", url.PathEscape(principalName), url.PathEscape(roleName))
	return c.doRequest(ctx, http.MethodDelete, path, nil, nil)
}

// ListPrincipalRolesForPrincipal lists principal roles assigned to a principal.
func (c *Client) ListPrincipalRolesForPrincipal(ctx context.Context, principalName string) ([]PrincipalRole, error) {
	var resp PrincipalRoleListResponse
	path := fmt.Sprintf("/api/management/v1/principals/%s/principal-roles", url.PathEscape(principalName))
	err := c.doRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Roles, nil
}

// --- Catalog Roles ---

// CreateCatalogRole creates a role within a catalog.
func (c *Client) CreateCatalogRole(ctx context.Context, catalogName string, req CreateCatalogRoleRequest) (*CatalogRole, error) {
	var resp CatalogRole
	path := fmt.Sprintf("/api/management/v1/catalogs/%s/catalog-roles", url.PathEscape(catalogName))
	err := c.doRequest(ctx, http.MethodPost, path, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetCatalogRole retrieves a catalog role.
func (c *Client) GetCatalogRole(ctx context.Context, catalogName string, roleName string) (*CatalogRole, error) {
	var resp CatalogRole
	path := fmt.Sprintf("/api/management/v1/catalogs/%s/catalog-roles/%s", url.PathEscape(catalogName), url.PathEscape(roleName))
	err := c.doRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateCatalogRole updates a catalog role.
func (c *Client) UpdateCatalogRole(ctx context.Context, catalogName string, roleName string, req UpdateCatalogRoleRequest) (*CatalogRole, error) {
	var resp CatalogRole
	path := fmt.Sprintf("/api/management/v1/catalogs/%s/catalog-roles/%s", url.PathEscape(catalogName), url.PathEscape(roleName))
	err := c.doRequest(ctx, http.MethodPut, path, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteCatalogRole deletes a catalog role.
func (c *Client) DeleteCatalogRole(ctx context.Context, catalogName string, roleName string) error {
	path := fmt.Sprintf("/api/management/v1/catalogs/%s/catalog-roles/%s", url.PathEscape(catalogName), url.PathEscape(roleName))
	return c.doRequest(ctx, http.MethodDelete, path, nil, nil)
}

// ListCatalogRoles lists catalog roles for a catalog.
func (c *Client) ListCatalogRoles(ctx context.Context, catalogName string) ([]CatalogRole, error) {
	var resp CatalogRoleListResponse
	path := fmt.Sprintf("/api/management/v1/catalogs/%s/catalog-roles", url.PathEscape(catalogName))
	err := c.doRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Roles, nil
}

// AssignCatalogRoleToPrincipalRole assigns a catalog role to a principal role.
func (c *Client) AssignCatalogRoleToPrincipalRole(ctx context.Context, principalRoleName string, catalogName string, catalogRoleName string) error {
	req := GrantCatalogRoleRequest{
		CatalogRole: CatalogRole{
			Name: catalogRoleName,
		},
	}
	path := fmt.Sprintf("/api/management/v1/principal-roles/%s/catalog-roles/%s", url.PathEscape(principalRoleName), url.PathEscape(catalogName))
	return c.doRequest(ctx, http.MethodPut, path, req, nil)
}

// UnassignCatalogRoleFromPrincipalRole removes a catalog role from a principal role.
func (c *Client) UnassignCatalogRoleFromPrincipalRole(ctx context.Context, principalRoleName string, catalogName string, catalogRoleName string) error {
	path := fmt.Sprintf("/api/management/v1/principal-roles/%s/catalog-roles/%s/%s", url.PathEscape(principalRoleName), url.PathEscape(catalogName), url.PathEscape(catalogRoleName))
	return c.doRequest(ctx, http.MethodDelete, path, nil, nil)
}

// ListCatalogRolesForPrincipalRole lists catalog roles assigned to a principal role for a catalog.
func (c *Client) ListCatalogRolesForPrincipalRole(ctx context.Context, principalRoleName string, catalogName string) ([]CatalogRole, error) {
	var resp CatalogRoleListResponse
	path := fmt.Sprintf("/api/management/v1/principal-roles/%s/catalog-roles/%s", url.PathEscape(principalRoleName), url.PathEscape(catalogName))
	err := c.doRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Roles, nil
}

// --- Grants ---

// AddGrant grants a privilege to a catalog role.
func (c *Client) AddGrant(ctx context.Context, catalogName string, catalogRoleName string, grant GrantResource) error {
	req := AddGrantRequest{
		Grant: grant,
	}
	path := fmt.Sprintf("/api/management/v1/catalogs/%s/catalog-roles/%s/grants", url.PathEscape(catalogName), url.PathEscape(catalogRoleName))
	return c.doRequest(ctx, http.MethodPut, path, req, nil)
}

// RevokeGrant revokes a privilege from a catalog role.
func (c *Client) RevokeGrant(ctx context.Context, catalogName string, catalogRoleName string, grant GrantResource) error {
	req := RevokeGrantRequest{
		Grant: grant,
	}
	path := fmt.Sprintf("/api/management/v1/catalogs/%s/catalog-roles/%s/grants", url.PathEscape(catalogName), url.PathEscape(catalogRoleName))
	return c.doRequest(ctx, http.MethodPost, path, req, nil)
}

// ListGrants lists all grants on a catalog role.
func (c *Client) ListGrants(ctx context.Context, catalogName string, catalogRoleName string) ([]GrantResource, error) {
	var resp GrantResourcesResponse
	path := fmt.Sprintf("/api/management/v1/catalogs/%s/catalog-roles/%s/grants", url.PathEscape(catalogName), url.PathEscape(catalogRoleName))
	err := c.doRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Grants, nil
}


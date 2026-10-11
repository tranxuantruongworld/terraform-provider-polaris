package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_OAuthAndCatalogCRUD(t *testing.T) {
	mux := http.NewServeMux()

	// OAuth2 token mock
	mux.HandleFunc("/api/catalog/v1/oauth/tokens", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Polaris-Realm") != "test-realm" {
			t.Errorf("expected realm test-realm, got %s", r.Header.Get("Polaris-Realm"))
		}
		_ = r.ParseForm()
		if r.FormValue("client_id") != "test_id" || r.FormValue("client_secret") != "test_secret" {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		resp := OAuthTokenResponse{
			AccessToken: "mock-token-xyz",
			TokenType:   "bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	catalogs := make(map[string]Catalog)

	// Catalogs handler
	mux.HandleFunc("/api/management/v1/catalogs", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer mock-token-xyz" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Polaris-Realm") != "test-realm" {
			http.Error(w, "missing realm", http.StatusBadRequest)
			return
		}

		if r.Method == http.MethodPost {
			var req CreateCatalogRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			catalogs[req.Catalog.Name] = req.Catalog
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(req.Catalog)
			return
		} else if r.Method == http.MethodGet {
			list := make([]Catalog, 0, len(catalogs))
			for _, cat := range catalogs {
				list = append(list, cat)
			}
			_ = json.NewEncoder(w).Encode(CatalogListResponse{Catalogs: list})
			return
		}
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/api/management/v1/catalogs/", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[len("/api/management/v1/catalogs/"):]
		if r.Header.Get("Authorization") != "Bearer mock-token-xyz" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		switch r.Method {
		case http.MethodGet:
			cat, exists := catalogs[name]
			if !exists {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(cat)
		case http.MethodPut:
			var req UpdateCatalogRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			cat, exists := catalogs[name]
			if !exists {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			if req.Properties != nil {
				cat.Properties = req.Properties
			}
			cat.EntityVersion = req.CurrentEntityVersion + 1
			catalogs[name] = cat
			_ = json.NewEncoder(w).Encode(cat)
		case http.MethodDelete:
			delete(catalogs, name)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c, err := NewClient(Config{
		BaseURL:      server.URL,
		ClientID:     "test_id",
		ClientSecret: "test_secret",
		Realm:        "test-realm",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()

	// 1. Create Catalog
	created, err := c.CreateCatalog(ctx, CreateCatalogRequest{
		Catalog: Catalog{
			Name: "test_catalog",
			Type: "INTERNAL",
			Properties: map[string]string{
				"default-base-location": "s3://my-bucket/lakehouse",
			},
			StorageConfigInfo: &StorageConfigInfo{
				StorageType:      "S3",
				RoleArn:          "arn:aws:iam::123456789012:role/polaris",
				AllowedLocations: []string{"s3://my-bucket/lakehouse"},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateCatalog failed: %v", err)
	}
	if created.Name != "test_catalog" {
		t.Fatalf("expected catalog name 'test_catalog', got '%s'", created.Name)
	}

	// 2. Get Catalog
	got, err := c.GetCatalog(ctx, "test_catalog")
	if err != nil {
		t.Fatalf("GetCatalog failed: %v", err)
	}
	if got.Properties["default-base-location"] != "s3://my-bucket/lakehouse" {
		t.Fatalf("unexpected properties: %v", got.Properties)
	}

	// 3. Update Catalog
	updated, err := c.UpdateCatalog(ctx, "test_catalog", UpdateCatalogRequest{
		CurrentEntityVersion: got.EntityVersion,
		Properties: map[string]string{
			"default-base-location": "s3://my-bucket/lakehouse-v2",
		},
	})
	if err != nil {
		t.Fatalf("UpdateCatalog failed: %v", err)
	}
	if updated.Properties["default-base-location"] != "s3://my-bucket/lakehouse-v2" {
		t.Fatalf("update was not reflected: %v", updated.Properties)
	}

	// 4. List Catalogs
	list, err := c.ListCatalogs(ctx)
	if err != nil {
		t.Fatalf("ListCatalogs failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 catalog, got %d", len(list))
	}

	// 5. Delete Catalog
	if err := c.DeleteCatalog(ctx, "test_catalog"); err != nil {
		t.Fatalf("DeleteCatalog failed: %v", err)
	}

	// 6. Get Catalog should return 404
	_, err = c.GetCatalog(ctx, "test_catalog")
	if !IsNotFound(err) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestClient_PrincipalAndRolesAndGrants(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/management/v1/principals", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			resp := PrincipalWithCredentials{
				Principal: Principal{
					Name:          "analyst",
					ClientID:      "analyst_id",
					EntityVersion: 1,
				},
				Credentials: PrincipalCredentials{
					ClientID:     "analyst_id",
					ClientSecret: "secret_value",
				},
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})

	mux.HandleFunc("/api/management/v1/principal-roles", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var req CreatePrincipalRoleRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(req.PrincipalRole)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})

	mux.HandleFunc("/api/management/v1/principals/analyst/principal-roles", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})

	mux.HandleFunc("/api/management/v1/catalogs/test_cat/catalog-roles", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var req CreateCatalogRoleRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(req.CatalogRole)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})

	mux.HandleFunc("/api/management/v1/principal-roles/data_team/catalog-roles/test_cat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})

	mux.HandleFunc("/api/management/v1/catalogs/test_cat/catalog-roles/read_only/grants", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusCreated)
			return
		} else if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			return
		} else if r.Method == http.MethodGet {
			resp := GrantResourcesResponse{
				Grants: []GrantResource{
					{
						Type:      "catalog",
						Privilege: "CATALOG_MANAGE_ACCESS",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c, err := NewClient(Config{
		BaseURL: server.URL,
		Token:   "static-token",
		Realm:   "default-realm",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()

	// Principal
	pResp, err := c.CreatePrincipal(ctx, CreatePrincipalRequest{
		Principal: Principal{Name: "analyst"},
	})
	if err != nil {
		t.Fatalf("CreatePrincipal failed: %v", err)
	}
	if pResp.Credentials.ClientSecret != "secret_value" {
		t.Fatalf("unexpected secret: %s", pResp.Credentials.ClientSecret)
	}

	// Principal role
	pr, err := c.CreatePrincipalRole(ctx, CreatePrincipalRoleRequest{
		PrincipalRole: PrincipalRole{Name: "data_team"},
	})
	if err != nil {
		t.Fatalf("CreatePrincipalRole failed: %v", err)
	}
	if pr.Name != "data_team" {
		t.Fatalf("unexpected role name: %s", pr.Name)
	}

	// Assign principal role
	if err := c.AssignPrincipalRole(ctx, "analyst", "data_team"); err != nil {
		t.Fatalf("AssignPrincipalRole failed: %v", err)
	}

	// Catalog role
	cr, err := c.CreateCatalogRole(ctx, "test_cat", CreateCatalogRoleRequest{
		CatalogRole: CatalogRole{Name: "read_only"},
	})
	if err != nil {
		t.Fatalf("CreateCatalogRole failed: %v", err)
	}
	if cr.Name != "read_only" {
		t.Fatalf("unexpected catalog role: %s", cr.Name)
	}

	// Assign catalog role to principal role
	if err := c.AssignCatalogRoleToPrincipalRole(ctx, "data_team", "test_cat", "read_only"); err != nil {
		t.Fatalf("AssignCatalogRoleToPrincipalRole failed: %v", err)
	}

	// Grant
	grant := GrantResource{
		Type:      "catalog",
		Privilege: "CATALOG_MANAGE_ACCESS",
	}
	if err := c.AddGrant(ctx, "test_cat", "read_only", grant); err != nil {
		t.Fatalf("AddGrant failed: %v", err)
	}

	grants, err := c.ListGrants(ctx, "test_cat", "read_only")
	if err != nil {
		t.Fatalf("ListGrants failed: %v", err)
	}
	if len(grants) != 1 || grants[0].Privilege != "CATALOG_MANAGE_ACCESS" {
		t.Fatalf("unexpected grants: %v", grants)
	}

	if err := c.RevokeGrant(ctx, "test_cat", "read_only", grant); err != nil {
		t.Fatalf("RevokeGrant failed: %v", err)
	}
}

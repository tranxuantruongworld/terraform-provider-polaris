terraform {
  required_providers {
    polaris = {
      source  = "tranxuantruongworld/polaris"
      version = "~> 1.0.0"
    }
  }
}

provider "polaris" {
  server_url    = "http://localhost:8181"
  client_id     = "polaris_root"
  client_secret = "polaris_secret"
  realm         = "default-realm"
}

# 1. Manage Iceberg Catalog with S3 Storage Credential Vending
resource "polaris_catalog" "analytics" {
  name                  = "lakehouse_analytics"
  default_base_location = "s3://lakehouse-data-bucket/analytics"

  storage_config {
    storage_type      = "S3"
    allowed_locations = ["s3://lakehouse-data-bucket/analytics"]
    role_arn          = "arn:aws:iam::123456789012:role/polaris-lakehouse-vending"
    region            = "us-east-1"
  }

  properties = {
    team        = "data-analytics"
    environment = "production"
  }
}

# 2. Manage Principals (User / Service Identities)
resource "polaris_principal" "analyst" {
  name = "alice_analyst"

  properties = {
    email      = "alice@company.com"
    department = "bi"
  }
}

# 3. Manage Principal Roles
resource "polaris_principal_role" "bi_engineers" {
  name = "bi_engineers"

  properties = {
    description = "Business Intelligence engineering team"
  }
}

# 4. Assign Principal Role to Principal
resource "polaris_principal_role_assignment" "alice_bi" {
  principal_name      = polaris_principal.analyst.name
  principal_role_name = polaris_principal_role.bi_engineers.name
}

# 5. Manage Catalog Roles scoped to Catalog
resource "polaris_catalog_role" "analytics_reader" {
  catalog_name = polaris_catalog.analytics.name
  name         = "analytics_reader"

  properties = {
    description = "Read-only access to analytics datasets"
  }
}

# 6. Map Catalog Role to Principal Role
resource "polaris_catalog_role_assignment" "bi_analytics_reader" {
  principal_role_name = polaris_principal_role.bi_engineers.name
  catalog_name        = polaris_catalog.analytics.name
  catalog_role_name   = polaris_catalog_role.analytics_reader.name
}

# 7. Grant Privileges on Table / Namespace
resource "polaris_grant" "read_sales_table" {
  catalog_name      = polaris_catalog.analytics.name
  catalog_role_name = polaris_catalog_role.analytics_reader.name
  type              = "table"
  namespace         = ["ecommerce"]
  table_name        = "sales_orders"
  privilege         = "TABLE_READ_DATA"
}

# 8. Query Data Sources
data "polaris_catalog" "current" {
  name       = polaris_catalog.analytics.name
  depends_on = [polaris_catalog.analytics]
}

output "principal_client_id" {
  description = "OAuth2 Client ID for Alice"
  value       = polaris_principal.analyst.client_id
}

output "catalog_default_location" {
  description = "Resolved base location"
  value       = data.polaris_catalog.current.default_base_location
}


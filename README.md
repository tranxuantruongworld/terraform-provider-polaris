# Terraform Provider for Apache Polaris

[![Tests](https://github.com/tranxuantruongworld/terraform-provider-polaris/actions/workflows/test.yml/badge.svg)](https://github.com/tranxuantruongworld/terraform-provider-polaris/actions/workflows/test.yml)
[![Release](https://github.com/tranxuantruongworld/terraform-provider-polaris/actions/workflows/release.yml/badge.svg)](https://github.com/tranxuantruongworld/terraform-provider-polaris/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/tranxuantruongworld/terraform-provider-polaris)](https://goreportcard.com/report/github.com/tranxuantruongworld/terraform-provider-polaris)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A production-ready [Terraform](https://www.terraform.io/) provider for managing [Apache Polaris (incubating)](https://polaris.apache.org/) — the open-source Iceberg REST catalog with central security, multi-tenant RBAC, and cloud credential vending.

---

## Key Features

- **Iceberg Catalogs (`polaris_catalog`)**: Manage internal and external catalogs with storage configurations for **AWS S3**, **Google Cloud Storage (GCS)**, **Azure ADLS Gen2**, and local testing.
- **Principals & Credentials (`polaris_principal`)**: Create identities with automatically provisioned OAuth2 `client_id` and sensitive `client_secret`.
- **Principal Roles (`polaris_principal_role`)**: Manage organization roles and federated roles.
- **Principal Role Assignments (`polaris_principal_role_assignment`)**: Bind principal roles to specific users and service principals.
- **Catalog Roles (`polaris_catalog_role`)**: Manage catalog-scoped roles.
- **Catalog Role Assignments (`polaris_catalog_role_assignment`)**: Map catalog roles to principal roles for cross-catalog RBAC.
- **Fine-Grained Privileges (`polaris_grant`)**: Enforce securable privilege grants on catalogs, namespaces, tables, and views (e.g., `TABLE_READ_DATA`, `TABLE_WRITE_DATA`, `CATALOG_MANAGE_ACCESS`).
- **Data Sources**: Read existing catalogs, principals, principal roles, and catalog roles.

---

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22 (for local development and compiling)
- Apache Polaris Server >= 0.1.0

---

## Authentication & Configuration

The provider supports both OAuth2 client credentials authentication and direct static Bearer token injection. You can also specify the multi-tenant `realm` context.

### Provider Configuration

```hcl
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
  client_secret = "secret"
  realm         = "default-realm" # Optional, defaults to "default-realm"
}
```

### Environment Variables

You can alternatively configure the provider using environment variables:

| Environment Variable | Description |
|---|---|
| `POLARIS_SERVER_URL` | Base URL of the Polaris service (e.g. `http://localhost:8181`) |
| `POLARIS_CLIENT_ID` | OAuth2 Client ID |
| `POLARIS_CLIENT_SECRET` | OAuth2 Client Secret |
| `POLARIS_REALM` | Polaris Realm name (defaults to `default-realm`) |
| `POLARIS_TOKEN` | Direct static Bearer token (bypasses OAuth2 endpoint) |

---

## Example Usage

```hcl
# 1. Create an Iceberg Catalog with AWS S3 storage vending
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

# 2. Create Principal identity
resource "polaris_principal" "analyst" {
  name = "alice_analyst"

  properties = {
    email      = "alice@company.com"
    department = "bi"
  }
}

# 3. Create Principal Role
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

# 5. Create Catalog Role scoped to the catalog
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

# 7. Grant Privileges
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

output "analyst_client_id" {
  description = "OAuth2 Client ID for Alice"
  value       = polaris_principal.analyst.client_id
}
```

---

## Development & Testing

### Building Locally

```bash
make build
```

This compiles the provider binary to `bin/terraform-provider-polaris`.

### Running Tests

```bash
make test
```

### Generating Registry Documentation

Documentation is generated using `tfplugindocs`:

```bash
make generate
```

### Local Terraform Testing with dev_overrides

Use `examples/dev.tfrc` to test the provider binary without installing from the registry:

```bash
TF_CLI_CONFIG_FILE=dev.tfrc terraform validate
```

---

## License

This project is licensed under the [Apache License 2.0](LICENSE).

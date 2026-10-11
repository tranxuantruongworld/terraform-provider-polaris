resource "polaris_grant" "admin_all_access" {
  catalog_name      = polaris_catalog.analytics.name
  catalog_role_name = polaris_catalog_role.catalog_admin.name
  type              = "catalog"
  privilege         = "CATALOG_MANAGE_ACCESS"
}

resource "polaris_grant" "read_table_data" {
  catalog_name      = polaris_catalog.analytics.name
  catalog_role_name = polaris_catalog_role.catalog_admin.name
  type              = "table"
  namespace         = ["warehouse"]
  table_name        = "events"
  privilege         = "TABLE_READ_DATA"
}

resource "polaris_catalog_role_assignment" "engineers_catalog_admin" {
  principal_role_name = polaris_principal_role.data_engineers.name
  catalog_name        = polaris_catalog.analytics.name
  catalog_role_name   = polaris_catalog_role.catalog_admin.name
}


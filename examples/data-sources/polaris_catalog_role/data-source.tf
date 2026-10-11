data "polaris_catalog_role" "admin" {
  catalog_name = "analytics_catalog"
  name         = "catalog_admin"
}

output "catalog_role_version" {
  value = data.polaris_catalog_role.admin.entity_version
}

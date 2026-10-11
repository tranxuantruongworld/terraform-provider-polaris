resource "polaris_catalog_role" "catalog_admin" {
  catalog_name = polaris_catalog.analytics.name
  name         = "catalog_admin"

  properties = {
    description = "Full administrator for analytics catalog"
  }
}

data "polaris_catalog" "existing" {
  name = "analytics_catalog"
}

output "catalog_location" {
  value = data.polaris_catalog.existing.default_base_location
}


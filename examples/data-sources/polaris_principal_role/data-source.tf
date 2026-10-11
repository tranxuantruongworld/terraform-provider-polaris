data "polaris_principal_role" "analysts" {
  name = "data_engineers_role"
}

output "role_version" {
  value = data.polaris_principal_role.analysts.entity_version
}

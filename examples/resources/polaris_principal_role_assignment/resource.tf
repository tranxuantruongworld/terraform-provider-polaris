resource "polaris_principal_role_assignment" "john_engineers" {
  principal_name      = polaris_principal.data_engineer.name
  principal_role_name = polaris_principal_role.data_engineers.name
}


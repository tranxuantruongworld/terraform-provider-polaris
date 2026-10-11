resource "polaris_principal_role" "data_engineers" {
  name = "data_engineers_role"

  properties = {
    description = "Role for data engineering members"
  }
}

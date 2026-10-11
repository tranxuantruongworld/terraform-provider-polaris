resource "polaris_principal" "data_engineer" {
  name = "john_engineer"

  properties = {
    email      = "john.engineer@company.com"
    department = "data-platform"
  }
}


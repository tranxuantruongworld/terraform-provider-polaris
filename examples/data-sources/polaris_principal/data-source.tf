data "polaris_principal" "analyst" {
  name = "john_engineer"
}

output "principal_client_id" {
  value = data.polaris_principal.analyst.client_id
}

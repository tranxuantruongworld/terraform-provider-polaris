terraform {
  required_providers {
    polaris = {
      source  = "tranxuantruongworld/polaris"
      version = "~> 1.0.0"
    }
  }
}

provider "polaris" {
  server_url    = "http://localhost:8181"
  client_id     = "polaris_root"
  client_secret = "polaris_secret"
  realm         = "default-realm"
}

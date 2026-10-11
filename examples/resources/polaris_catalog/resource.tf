resource "polaris_catalog" "analytics" {
  name                  = "analytics_catalog"
  default_base_location = "s3://lakehouse-bucket/analytics"

  storage_config {
    storage_type      = "S3"
    allowed_locations = ["s3://lakehouse-bucket/analytics"]
    role_arn          = "arn:aws:iam::123456789012:role/polaris-lakehouse-role"
    region            = "us-east-1"
  }

  properties = {
    team        = "analytics"
    environment = "production"
  }
}


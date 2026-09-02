# expected load:
#   monthly_requests: 5_000_000
#   request_duration_ms: 120
#   confidence: high
#   source: observed
resource "aws_lambda_function" "api" {
  function_name = "api"
}

# expected load:
#   storage_gb: 250
resource "aws_s3_bucket" "assets" {
  bucket = "assets"
}

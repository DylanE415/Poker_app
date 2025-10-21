/*
AWS provider configuration using root-level variables.
- region: required
- profile: optional (null uses default credential chain)
- assume_role: optional, created dynamically only when assume_role_arn is provided
- skip_credentials_validation: pass-through boolean
*/

provider "aws" {
  region = var.aws_region
  # only set profile when explicitly provided to avoid overriding default chain with null
  profile = var.aws_profile != null ? var.aws_profile : null

  # Dynamically add assume_role block only if an ARN is provided
  dynamic "assume_role" {
    for_each = var.assume_role_arn != null ? [var.assume_role_arn] : []
    content {
      role_arn     = assume_role.value
      session_name = var.assume_role_session_name != "" ? var.assume_role_session_name : "tf-session"
    }
  }

  skip_credentials_validation = var.skip_credentials_validation
}

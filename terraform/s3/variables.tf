// Variables for the static assets S3 bucket module

variable "bucket_name" {
  description = "The name of the S3 bucket. Must be globally unique."
  type        = string
}

variable "index_document" {
  description = "Index document for S3 static website hosting."
  type        = string
  default     = "index.html"
}

variable "error_document" {
  description = "Error document for S3 static website hosting."
  type        = string
  default     = "error.html"
}

variable "force_destroy" {
  description = "Whether to force destroy the bucket (delete all objects) when the bucket resource is destroyed. Useful for dev/testing."
  type        = bool
  default     = false
}

variable "tags" {
  description = "A map of tags to add to all resources."
  type        = map(string)
  default     = {}
}

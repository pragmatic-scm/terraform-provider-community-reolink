# SPDX-License-Identifier: MPL-2.0

variable "reolink_email_configuration" {
  type      = string
  sensitive = true
}

variable "reolink_email_configuration_revision" {
  type = string
}

resource "reolink_email" "front" {
  camera                 = "front"
  enabled                = false
  configuration          = var.reolink_email_configuration
  configuration_revision = var.reolink_email_configuration_revision
}

# SPDX-License-Identifier: MPL-2.0

variable "reolink_ftp_configuration" {
  type      = string
  sensitive = true
}

variable "reolink_ftp_configuration_revision" {
  type = string
}

resource "reolink_ftp" "front" {
  camera                 = "front"
  enabled                = false
  configuration          = var.reolink_ftp_configuration
  configuration_revision = var.reolink_ftp_configuration_revision
}

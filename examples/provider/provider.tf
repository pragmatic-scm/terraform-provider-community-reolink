# SPDX-License-Identifier: MPL-2.0

terraform {
  required_version = ">= 1.11.0"

  required_providers {
    reolink = {
      source  = "pragmatic-scm/community-reolink"
      version = "~> 0.1"
    }
  }
}

variable "reolink_password" {
  type      = string
  sensitive = true
}

provider "reolink" {
  cameras = {
    front = {
      url      = "https://camera.example.com"
      username = "admin"
      password = var.reolink_password
      insecure = false
    }
  }
}

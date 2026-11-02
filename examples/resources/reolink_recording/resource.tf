# SPDX-License-Identifier: MPL-2.0

variable "reolink_recording_schedule" {
  description = "Camera-native weekly recording schedule encoded as JSON."
  type        = string
}

resource "reolink_recording" "front" {
  camera   = "front"
  enabled  = true
  schedule = var.reolink_recording_schedule
}

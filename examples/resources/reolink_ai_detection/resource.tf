# SPDX-License-Identifier: MPL-2.0

resource "reolink_ai_detection" "people" {
  camera      = "front"
  channel     = 0
  type        = "people"
  sensitivity = 60
  delay       = 0
}

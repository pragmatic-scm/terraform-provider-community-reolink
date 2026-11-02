# SPDX-License-Identifier: MPL-2.0

resource "reolink_motion_detection" "front" {
  camera      = "front"
  channel     = 0
  sensitivity = 25
}

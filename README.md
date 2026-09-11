<!-- SPDX-License-Identifier: MPL-2.0 -->

# Reolink Terraform provider

This provider manages settings on standalone Reolink cameras through the
camera HTTP API. It is intentionally separate from the ZoneMinder provider.

The provider accepts a map of camera connections so resources can use the same
stable camera keys as the ZoneMinder camera loader without requiring one
provider alias per camera. Passwords are sensitive provider configuration and
are not resource attributes.

```hcl
terraform {
  required_providers {
    reolink = {
      source = "eremington/reolink"
    }
  }
}

provider "reolink" {
  cameras = var.reolink_cameras
}

resource "reolink_ai_detection" "inside_garage_people" {
  camera      = "camera8"
  channel     = 0
  type        = "people"
  sensitivity = 60
  delay       = 1
}
```

`reolink_ai_detection` manages the camera-native `AiAlarm` object. Besides
sensitivity and alarm delay, it can manage normalized minimum/maximum target
dimensions and the camera's native detection-mask bit string. It reads the
complete object before every update and preserves fields that Terraform does
not manage, which is required across Reolink firmware variants.

Import IDs use `camera/channel/type`:

```sh
terraform import reolink_ai_detection.inside_garage_people camera8/0/people
```

Removing a resource from configuration only stops managing that setting;
there is no meaningful camera-side delete operation.

`reolink_motion_detection` manages the older pixel-motion settings used by
RLC-410 cameras. Its `sensitivity` is the camera-native 1–50 value and is
applied uniformly to the camera's existing time periods; their schedule
boundaries and other firmware-specific fields are preserved.

Run tests with:

```sh
go test ./...
```

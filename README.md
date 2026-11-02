<!-- SPDX-License-Identifier: MPL-2.0 -->

# terraform-provider-community-reolink

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
      source = "pragmatic-scm/community-reolink"
    }
  }
}

provider "reolink" {
  cameras = var.reolink_cameras
}

resource "reolink_ai_detection" "camera8_people" {
  camera      = "camera8"
  channel     = 0
  type        = "people"
  sensitivity = 60
  delay       = 0
}
```

`reolink_ai_detection` manages the camera-native `AiAlarm` object. Besides
sensitivity and alarm delay, it can manage normalized minimum/maximum target
dimensions and the camera's native detection-mask bit string. It reads the
complete object before every update and preserves fields that Terraform does
not manage, which is required across Reolink firmware variants.

Import IDs use `camera/channel/type`:

```sh
terraform import reolink_ai_detection.camera8_people camera8/0/people
```

Removing a resource from configuration only stops managing that setting;
there is no meaningful camera-side delete operation.

`reolink_push` manages the global Push enable switch while preserving the
camera's firmware-specific schedule and other fields:

```hcl
resource "reolink_push" "camera8" {
  camera  = "camera8"
  enabled = false
}
```

Import IDs are camera keys, for example:

```sh
terraform import reolink_push.camera8 camera8
```

`reolink_camera_info` reads the camera's model, exact type, hardware version,
firmware version, configuration version, build date, and channel count. These
values are discovered from `GetDevInfo`; they are not duplicated in provider
configuration. API resources still probe supported command variants because
Reolink capability reporting varies across firmware releases.

`reolink_email` and `reolink_ftp` manage their respective enable switch and a
sensitive, write-only JSON `configuration` object. A non-secret
`configuration_revision` marker is stored in state and triggers an update when
the associated 1Password item changes. Before writing, the provider reads the
complete camera object and overlays the declared configuration. This preserves
schedules, capability fields, and other firmware-specific values that are not
declared by Terraform. Both resources automatically select the legacy or V20
camera API. Import IDs are camera keys.

The configuration JSON is available to the provider during apply but is not
stored in new Terraform state or saved plans. Imported resources need both a
configuration and revision value before they can be planned or applied. Older
state and plan files created before this feature may still contain credentials.
Apply configurations containing write-only values directly; a separately saved
plan cannot replay values that Terraform deliberately did not retain.

`reolink_motion_detection` manages the older pixel-motion settings used by
RLC-410 cameras. Its `sensitivity` is the camera-native 1–50 value and is
applied uniformly to the camera's existing time periods; their schedule
boundaries and other firmware-specific fields are preserved.

`reolink_recording` manages camera recording enablement and its native weekly
alarm-action schedule. The `schedule` attributes on `reolink_recording`,
`reolink_email`, and `reolink_ftp` contain JSON because legacy and V20 firmware
use different schedule structures. The provider reads and writes those native
objects without normalization. Email and FTP credentials remain separate
sensitive, write-only configuration.

Run tests with:

```sh
go test ./...
```

Security issues should be reported privately to
terraform-provider-community-reolink@pragmatic-scm.com.

## Validated camera firmware

The following model, hardware, and firmware combinations have passed the
provider's live read-only API validation and a clean Terraform refresh. The
diagnostic logs in to the camera and reads device information, capabilities,
Email/EmailV20, and FTP/FTPV20. The Terraform refresh also reads the configured
Push and detection resources through the provider. An entry here records a
version known to work; it is not a statement that older or newer firmware is
unsupported.

| Model | Hardware | Firmware |
| --- | --- | --- |
| Reolink Duo PoE | `IPC_528B174MP` | `v3.0.0.1388_24021900` |
| RLC-410-5MP | `IPC_515B16M5M` | `v3.0.0.136_20121100` |
| RLC-510A | `IPC_523128M5MP_V2` | `v3.1.0.801_22010700` |
| RLC-510A | `IPC_523128M5MP_V2` | `v3.1.0.2368_23062500` |
| RLC-510A | `IPC_MS4NA45MP` | `v3.2.0.5180_2507241368` |

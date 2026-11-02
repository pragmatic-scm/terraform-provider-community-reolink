<!-- SPDX-License-Identifier: MPL-2.0 -->

# terraform-provider-community-reolink

[![License: MPL-2.0](https://img.shields.io/badge/License-MPL--2.0-blue.svg)](LICENSE)

This provider manages settings on standalone Reolink cameras through the
camera HTTP API. A single provider configuration accepts a map of camera
connections, allowing resources to refer to cameras by stable keys without
requiring one provider alias per camera. Passwords are sensitive provider
configuration and are not resource attributes.

```hcl
terraform {
  required_providers {
    reolink = {
      source = "pragmatic-scm/community-reolink"
    }
  }
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

resource "reolink_ai_detection" "front_people" {
  camera      = "front"
  channel     = 0
  type        = "people"
  sensitivity = 60
  delay       = 0
}
```

Each entry in `cameras` requires `url`, `username`, and `password`. Set the
optional `insecure` attribute to `true` only when the camera uses a TLS
certificate that cannot be verified. Supply passwords through sensitive input
variables or another secret-management mechanism; do not commit them to the
configuration.

## Resources and data sources

The provider includes these resources:

| Name | Purpose |
| --- | --- |
| `reolink_ai_detection` | Smart-detection sensitivity, delay, target dimensions, and mask for one detection class |
| `reolink_motion_detection` | Conventional pixel-motion sensitivity and mask |
| `reolink_push` | Push-notification enablement |
| `reolink_email` | Email enablement, schedule, and configuration |
| `reolink_ftp` | FTP enablement, schedule, and configuration |
| `reolink_recording` | Recording enablement and native weekly schedule |

The `reolink_camera_info` data source reads model, hardware, firmware,
configuration, build-date, and channel-count information from a camera.

`reolink_ai_detection` manages the camera-native `AiAlarm` object. Besides
sensitivity and alarm delay, it can manage normalized minimum/maximum target
dimensions and the camera's native detection-mask bit string. It reads the
complete object before every update and preserves fields that Terraform does
not manage, which is required across Reolink firmware variants.

Import IDs use `camera/channel/type`:

```sh
terraform import reolink_ai_detection.front_people front/0/people
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
the externally supplied configuration changes. The marker can be a secret
version, checksum, timestamp, or another value managed by the calling
configuration. Before writing, the provider reads the complete camera object
and overlays the declared configuration. This preserves
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

## Support and contributing

Use the repository's structured
[issue forms](https://github.com/pragmatic-scm/terraform-provider-community-reolink/issues/new/choose)
for bug reports and enhancement requests. Include the provider, Terraform,
camera model, hardware, and firmware versions when reporting compatibility
problems. See [CONTRIBUTING.md](CONTRIBUTING.md) for development and pull
request guidance.

Security issues and exposed credentials must not be posted publicly. Follow
the private reporting instructions in [SECURITY.md](SECURITY.md).

## License

This project is licensed under the Mozilla Public License 2.0. See
[LICENSE](LICENSE) for the complete terms.

## Validated camera firmware

This provider began as a hobby project because I wanted to manage the cameras
in my own installation with Terraform. The validated models and firmware
versions below reflect the hardware I currently have available for testing,
not an intentional limit on what the provider may support. Compatibility
reports are welcome, and I am open to donated test hardware or safely
coordinated temporary access to other models and firmware. Never post camera
credentials or access details in a public issue; open a sanitized issue first
so any access can be coordinated privately. This remains a hobby project that
I maintain on a best-effort basis, and I cannot guarantee response or
turnaround times. Hardware or financial sponsorship may let me dedicate more
time to testing and development, but it does not by itself create a support
commitment or service level agreement.

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

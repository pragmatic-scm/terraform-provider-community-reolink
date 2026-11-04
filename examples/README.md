<!-- SPDX-License-Identifier: MPL-2.0 -->

# Examples

These examples show how to configure the provider for standalone Reolink
cameras and manage supported camera settings. They are small building blocks:
copy the provider example and the resource or data-source examples you want to
use into one Terraform working directory, then adapt them for your cameras.

## Available examples

| Example | Description | Import example |
| --- | --- | --- |
| [Provider configuration](provider/provider.tf) | Configures the provider with a `front` camera connection. | — |
| [`reolink_camera_info` data source](data-sources/reolink_camera_info/data-source.tf) | Reads model, hardware, firmware, build, and channel information from a camera. | — |
| [`reolink_ai_detection` resource](resources/reolink_ai_detection/resource.tf) | Manages AI detection settings for one detection class and camera channel. | [Import](resources/reolink_ai_detection/import.sh) |
| [`reolink_motion_detection` resource](resources/reolink_motion_detection/resource.tf) | Manages conventional pixel-motion sensitivity for a camera channel. | [Import](resources/reolink_motion_detection/import.sh) |
| [`reolink_push` resource](resources/reolink_push/resource.tf) | Manages the camera's push-notification enable switch. | [Import](resources/reolink_push/import.sh) |
| [`reolink_email` resource](resources/reolink_email/resource.tf) | Manages camera email settings using sensitive configuration JSON. | [Import](resources/reolink_email/import.sh) |
| [`reolink_ftp` resource](resources/reolink_ftp/resource.tf) | Manages camera FTP settings using sensitive configuration JSON. | [Import](resources/reolink_ftp/import.sh) |
| [`reolink_recording` resource](resources/reolink_recording/resource.tf) | Manages camera recording enablement and its native weekly schedule. | [Import](resources/reolink_recording/import.sh) |

## Set up a working directory

Terraform loads `.tf` files from one directory, not recursively from the
`examples` tree. From the repository root, create a working directory and copy
in the provider configuration plus the examples you want to try. For example:

```sh
mkdir camera-example
cp examples/provider/provider.tf camera-example/
cp examples/data-sources/reolink_camera_info/data-source.tf camera-example/
cp examples/resources/reolink_push/resource.tf camera-example/
cd camera-example
```

Edit `provider.tf` to replace the example URL and username with values for your
camera. Keep the password out of version control. The provider example declares
it as the sensitive variable `reolink_password`, which you can supply through
an environment variable:

```sh
export TF_VAR_reolink_password='your-camera-password'
```

Review every copied resource before applying it. The resource examples contain
illustrative values and can change settings on the selected camera. Some
examples declare additional variables; provide those with `TF_VAR_...`
environment variables or a `.tfvars` file that is excluded from version
control.

## Initialize, preview, and apply

Run these commands from the working directory:

```sh
terraform init
terraform validate
terraform plan
terraform apply
```

`terraform plan` previews the proposed camera-setting changes. Review the plan
before running `terraform apply`, which sends the approved changes to the
camera. To save and apply the exact reviewed plan, use:

```sh
terraform plan -out=tfplan
terraform apply tfplan
```

Do not use a saved plan for the email or FTP examples: their sensitive,
write-only configuration values are intentionally not retained in saved plans.
For those resources, review with `terraform plan` and then run `terraform
apply` directly.

## Import existing settings

The `import.sh` files show the import ID expected by each resource. Copy the
matching command into your working directory or run the equivalent `terraform
import` command after configuring the resource. Import reads the current camera
setting into state; it does not replace the need to review the configuration
and the next plan.

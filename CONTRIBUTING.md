<!-- SPDX-License-Identifier: MPL-2.0 -->

# Contributing

Bug reports, camera compatibility reports, documentation fixes, and focused
provider enhancements are welcome. Use the repository's structured issue forms
before starting a substantial change so the intended Terraform behavior and
firmware constraints can be discussed.

Do not submit camera passwords, session tokens, private addresses, unredacted
API responses, state files, saved plans, or configurations containing secrets.
Report suspected vulnerabilities and exposed credentials through the private
channel described in [SECURITY.md](SECURITY.md).

## Bug and compatibility reports

Include the provider and Terraform versions, the camera model, hardware
version, firmware version, a minimal sanitized configuration, and reproducible
steps. Reolink API behavior varies across models and firmware generations, so
those exact versions are important. Reduce logs to the commands and fields
needed to demonstrate the problem and redact sensitive values before posting.

## Development

Use a supported Go toolchain and run the test suite before submitting a pull
request:

```sh
go test ./...
```

Generate Registry documentation after changing provider schemas or
descriptions:

```sh
go generate ./...
```

Review the generated documentation and commit it with the schema change. Keep
changes focused, add tests for new behavior and regressions, and preserve
camera fields that the provider does not explicitly manage.

Live camera testing must use non-production credentials where possible. Never
commit logs or fixtures captured from a real camera unless they have been
carefully reduced to synthetic, non-sensitive data.

## Pull requests

A pull request should explain the user-visible behavior, affected camera and
firmware variants, tests performed, and any documentation changes. By
contributing, you agree that your contribution is licensed under the Mozilla
Public License 2.0.

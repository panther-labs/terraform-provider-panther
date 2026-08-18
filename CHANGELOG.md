# Changelog

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html),
following [HashiCorp's provider versioning guidance](https://developer.hashicorp.com/terraform/plugin/best-practices/versioning):
MINOR for new backwards-compatible resources and features, PATCH for
backwards-compatible bug fixes, MAJOR for breaking changes.

## 0.3.0 (August 18, 2026)

FEATURES:

* **New Resource:** `panther_role` ([#114](https://github.com/panther-labs/terraform-provider-panther/pull/114))

NOTES:

* Adopted Semantic Versioning going forward. Adding a new resource is a
  backwards-compatible feature, so this release increments the MINOR version.
* Various dependency updates ([#110](https://github.com/panther-labs/terraform-provider-panther/pull/110), [#111](https://github.com/panther-labs/terraform-provider-panther/pull/111), [#112](https://github.com/panther-labs/terraform-provider-panther/pull/112), [#113](https://github.com/panther-labs/terraform-provider-panther/pull/113)).

## 0.2.13 (July 7, 2026)

FEATURES:

* **New Resource:** `panther_plf_source` ([#108](https://github.com/panther-labs/terraform-provider-panther/pull/108))

NOTES:

* Various dependency and CI updates, including a `go-git/go-billy` bump for VULN-134.

## 0.2.12 (May 25, 2026)

FEATURES:

* **New Resource:** `panther_aws_cloud_account` ([#98](https://github.com/panther-labs/terraform-provider-panther/pull/98))
* **New Resource:** `panther_log_source_alarm` ([#93](https://github.com/panther-labs/terraform-provider-panther/pull/93))

ENHANCEMENTS:

* Send a custom `User-Agent` on outbound HTTP requests ([#100](https://github.com/panther-labs/terraform-provider-panther/pull/100))

NOTES:

* Bumped `go.opentelemetry.io/otel` and `otel/sdk` to v1.43.0 for VULN-124 and VULN-125 ([#101](https://github.com/panther-labs/terraform-provider-panther/pull/101))

## 0.2.11 (April 27, 2026)

ENHANCEMENTS:

* Migrate the S3 log source from the GraphQL API to the REST API ([#92](https://github.com/panther-labs/terraform-provider-panther/pull/92))
* Improve the release pipeline: GitHub-native changelog, pinned `goreleaser-action` ([#95](https://github.com/panther-labs/terraform-provider-panther/pull/95))

## 0.2.10 (March 31, 2026)

FEATURES:

* **New Resource:** `panther_gcs_source` ([#90](https://github.com/panther-labs/terraform-provider-panther/pull/90))
* **New Resource:** `panther_pubsub_source` ([#86](https://github.com/panther-labs/terraform-provider-panther/pull/86))

ENHANCEMENTS:

* Replace the REST client interface with typed helpers and structured errors ([#89](https://github.com/panther-labs/terraform-provider-panther/pull/89))

## 0.2.9 (March 23, 2026)

ENHANCEMENTS:

* Add `retainEnvelopeFields` flag to the S3 source resource ([#83](https://github.com/panther-labs/terraform-provider-panther/pull/83))
* Document the release process in the README ([#76](https://github.com/panther-labs/terraform-provider-panther/pull/76))

NOTES:

* Various dependency updates.

## 0.2.8 (January 14, 2026)

ENHANCEMENTS:

* Add XML stream type support ([#58](https://github.com/panther-labs/terraform-provider-panther/pull/58))
* Add support for a root XML element ([#64](https://github.com/panther-labs/terraform-provider-panther/pull/64))
* Add `logStreamTypeOptions` to the S3 source resource ([#65](https://github.com/panther-labs/terraform-provider-panther/pull/65))
* Upgrade to Go 1.25 ([#75](https://github.com/panther-labs/terraform-provider-panther/pull/75))

NOTES:

* Various dependency updates.

## 0.2.7 (June 27, 2025)

ENHANCEMENTS:

* Update HTTP source documentation ([#57](https://github.com/panther-labs/terraform-provider-panther/pull/57))
* Clarify the S3 `log_stream_type` description ([#56](https://github.com/panther-labs/terraform-provider-panther/pull/56))

NOTES:

* Various dependency updates.

## 0.2.6 (May 30, 2025)

NOTES:

* Dependency and CI updates only.

## 0.2.5 (May 13, 2025)

ENHANCEMENTS:

* Add `logStreamTypeOptions` field to the HTTP source resource ([#46](https://github.com/panther-labs/terraform-provider-panther/pull/46))
* Enable vulnerability reporting ([#50](https://github.com/panther-labs/terraform-provider-panther/pull/50))

## 0.2.4 (February 10, 2025)

ENHANCEMENTS:

* Update files to generate documentation ([#44](https://github.com/panther-labs/terraform-provider-panther/pull/44))

## 0.2.3 (January 31, 2025)

ENHANCEMENTS:

* Update the Panther URL ([#42](https://github.com/panther-labs/terraform-provider-panther/pull/42))
* Switch the Panther provider registry asset owner ([#43](https://github.com/panther-labs/terraform-provider-panther/pull/43))

## 0.2.2 (January 23, 2025)

NOTES:

* Add CODEOWNERS ([#40](https://github.com/panther-labs/terraform-provider-panther/pull/40)) and dependency updates.

## 0.2.1 (January 22, 2025)

BUG FIXES:

* Fix the GoReleaser configuration ([#39](https://github.com/panther-labs/terraform-provider-panther/pull/39))

## 0.2.0 (January 21, 2025)

FEATURES:

* **New Resource:** `panther_httpsource` ([#33](https://github.com/panther-labs/terraform-provider-panther/pull/33))

ENHANCEMENTS:

* Add a full example for the S3 log source ([#24](https://github.com/panther-labs/terraform-provider-panther/pull/24))
* Harden CI workflows with `harden-runner` and full Action SHAs ([#15](https://github.com/panther-labs/terraform-provider-panther/pull/15), [#16](https://github.com/panther-labs/terraform-provider-panther/pull/16))

## 0.1.4 (May 1, 2024)

ENHANCEMENTS:

* Allow `Auto` as a value for `log_stream_type` ([#14](https://github.com/panther-labs/terraform-provider-panther/pull/14))

## 0.1.3 (August 15, 2023)

BUG FIXES:

* No longer use a pointer for `kms_key` ([#12](https://github.com/panther-labs/terraform-provider-panther/pull/12))

## 0.1.2 (August 10, 2023)

ENHANCEMENTS:

* Enforce a length limit and validation on the S3 source name ([#10](https://github.com/panther-labs/terraform-provider-panther/pull/10), [#11](https://github.com/panther-labs/terraform-provider-panther/pull/11))

## 0.1.1 (August 8, 2023)

NOTES:

* Update the example with correct field names ([#9](https://github.com/panther-labs/terraform-provider-panther/pull/9))

## 0.1.0 (July 31, 2023)

FEATURES:

* Initial release.
* **New Resource:** `panther_s3_source`

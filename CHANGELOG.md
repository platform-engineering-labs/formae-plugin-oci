# Changelog

All notable changes to the formae OCI plugin are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Install with `sudo formae plugin install oci` on the host that runs the
formae agent.

## [0.1.4]

### Added

- Public Hub onboarding: added the files needed to add the plugin to the Public
  Hub.

## [0.1.3]

### Added

- New `dev-environment/` and `platform/` examples in the
  [plugin repository](https://github.com/platform-engineering-labs/formae-plugin-oci/tree/main/examples).

### Changed

- Per-field config mutability: the `profile` and `configFilePath` fields are now
  mutable; changing them updates the target in place without recreating
  resources. The `region` field remains immutable; changing it triggers a full
  target replace as before.
- Readable labels for discovered resources: discovered OCI resources now surface
  their human-readable name as the label instead of the raw OCID. Most resources
  use their `displayName`; Compartments, Policies, OKE Clusters, Node Pools, and
  Object Storage Buckets use their `name` field.

### Fixed

- VirtualNodePool resources could fail to apply or update because the
  provisioner wasn't parsing nested configuration fields correctly. Nested
  fields are now read reliably.

## [0.1.2]

### Fixed

- Resources in a terminal lifecycle state (e.g. deleting, terminated) are now
  correctly detected as deleted during synchronization.
- Spurious diffs on `definedTags` and `freeformTags` fields during updates and
  synchronization. These provider-populated fields are now correctly recognized
  as defaults.

## [0.1.1]

### Fixed

- Fixed conformance tests with updated CI pipeline and integration test coverage
  for core OCI resources.

## [0.1.0]

### Added

- Initial release of the OCI plugin as a standalone package built on the formae
  Plugin SDK.

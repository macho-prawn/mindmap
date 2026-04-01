# Changelog

All notable changes to `netmap` are documented in this file.

## 1.0.0

### Added

- Initial `netmap` CLI for mapping source GCP dedicated interconnects to destination VLAN attachments, Cloud Routers, interfaces, and BGP peers from a local YAML inventory.
- Selector expansion across `org`, `workload`, and `environment`, including exact-match and fanout modes.
- Canonical output model shared across CSV, TSV, JSON, tree, and Mermaid renderers.
- Embedded version reporting via `netmap version`, returning the current CLI version without requiring config or cloud discovery.
- Source-side fields for interconnect state, MACsec enablement, and active MACsec key name.
- Destination-side fields for project, region, VPC, VLAN attachment, Cloud Router, router ASN, interface, BGP peer, peer ASN, and peering status.
- Mermaid DAG rendering with shared-node collapse for repeated environment, source project, interconnect, destination region, and destination VPC shapes.
- CLI progress reporting on `stderr` as a timed two-column table with a Braille spinner for active work and final output/total-time summary rows.
- GitHub Actions workflows for PR-time Go test validation and merged-PR release publishing for macOS and Windows amd64 binaries:
  - `NetMap Test` in `.github/workflows/netmap-test.yml`
  - `NetMap Release` in `.github/workflows/netmap-release.yml`
- GitHub Actions job display names for workflow visibility:
  - `Code Test`
  - `Build and Release`

### Changed

- Renamed the CLI and module surface from `mindmap` to `netmap`.
- Standardized output naming, field names, and canonical column ordering across all formats.
- Made Mermaid output the default when `-f` is omitted and documented compatibility with `https://mermaid.live`.
- Standardized release semantics so the repository `VERSION` file remains `1.0.0`, while GitHub release tags are published as `v1.0.0`.
- Updated release publishing to use `CHANGELOG.md` as the GitHub release notes source.

### Notes

- Source dedicated interconnects are modeled as global resources.
- Unmapped interconnects remain visible in output so missing destination mappings are explicit.
- Release archives contain only `README.md` and the platform-specific binary.

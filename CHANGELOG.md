# Changelog

All notable changes to `netmap` are documented in this file.

## 1.0.0

### Added

- Initial `netmap` CLI for mapping source GCP dedicated interconnects to destination VLAN attachments, Cloud Routers, interfaces, and BGP peers from a local YAML inventory.
- Selector expansion across `org`, `workload`, and `environment`, including exact-match and fanout modes.
- Canonical output model shared across CSV, TSV, JSON, tree, and Mermaid renderers.
- Source-side fields for interconnect state, MACsec enablement, and active MACsec key name.
- Destination-side fields for project, region, VPC, VLAN attachment, Cloud Router, router ASN, interface, BGP peer, peer ASN, and peering status.
- Mermaid DAG rendering with shared-node collapse for repeated environment, source project, interconnect, destination region, and destination VPC shapes.
- CLI progress reporting on `stderr` as a timed two-column table with a Braille spinner for active work and final output/total-time summary rows.
- GitHub Actions workflows for PR-time Go test validation and merged-PR release publishing for macOS and Windows amd64 binaries.

### Changed

- Renamed the CLI and module surface from `mindmap` to `netmap`.
- Standardized output naming, field names, and canonical column ordering across all formats.
- Made Mermaid output the default when `-f` is omitted and documented compatibility with `https://mermaid.live`.

### Notes

- Source dedicated interconnects are modeled as global resources.
- Unmapped interconnects remain visible in output so missing destination mappings are explicit.
- Release archives contain only `README.md` and the platform-specific binary.

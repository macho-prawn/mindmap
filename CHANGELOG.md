# Changelog

## v2.0.0

### Changed

- `-t vpn` now performs real discovery instead of returning an unimplemented error.
- VPN runs resolve `-o/-w/-e` selections as source projects, reject `-p`, and use those source projects to discover HA and Classic VPN gateways, tunnels, Cloud Routers, interfaces, and BGP peers.
- HA VPN tunnels now follow peer gateway references into destination projects to map peer-side VPN gateways, tunnels, and Cloud Routers, while Classic VPN tunnels remain in the output as source-side unmapped rows when peer GCP discovery is not available.
- CSV, TSV, JSON, tree, Mermaid, and HTML output are now shared across interconnect and VPN reports, with additive VPN fields in the canonical flat output.
- Mermaid rendering now uses a VPN-specific node strategy that collapses repeated `src_project -> src_region` and `dst_project -> dst_region` pairs while keeping project and region as separate nodes.
- VPN output filenames now use the `netmap-vpn-...` prefix, with `netmap-vpn-<src>-to-<dst>-<timestamp>.<ext>` for single-source/single-destination output and `netmap-vpn-<org>-all-<timestamp>.<ext>` for aggregate output.
- `netmap version` now prints `v2.0.0`, and release documentation now references `v2.0.0` with `VERSION=2.0.0` as the raw source of truth.

## v1.2.0

### Changed

- All CLI parameter checks and config-file semantic validation now complete before `netmap` initializes the Compute provider, checks ADC validity, or talks to the Compute API.
- The CLI usage/help text now comes from the user-editable embedded file [internal/app/usage.txt](/home/macho_prawn/gh-repo/netmap/internal/app/usage.txt), so help copy can be updated without editing the Go source that renders it.
- `netmap version` now prints `v1.2.0`.
- README and release documentation now reference `v1.2.0`, while the repository `VERSION` file remains the raw `1.2.0` source of truth used to derive release tag/title `v1.2.0`.

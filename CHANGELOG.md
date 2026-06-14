# Changelog

## v2.0.0

### Changed

- `-t vpn` now performs real discovery instead of returning an unimplemented error.
- VPN runs resolve `-o/-w/-e` selections as source projects, reject `-p`, and use those source projects to discover HA and Classic VPN gateways, tunnels, Cloud Routers, interfaces, and BGP peers.
- HA VPN tunnels now follow peer gateway references into destination projects to map peer-side VPN gateways, tunnels, and Cloud Routers, while Classic VPN tunnels remain in the output as source-side unmapped rows when peer GCP discovery is not available.
- CSV, TSV, JSON, tree, Mermaid, and HTML output are now shared across interconnect and VPN reports, with additive VPN fields in the canonical flat output.
- VPN outputs now include source and destination router-interface `routes`, derived from peer-advertised groups and IP ranges and shown across CSV, TSV, JSON, tree, Mermaid, and HTML reports.
- Mermaid rendering now uses a VPN-specific node strategy that collapses repeated `src_project -> src_region` and `dst_project -> dst_region` pairs while keeping project and region as separate nodes.
- VPN output filenames now use the `netmap-vpn-...` prefix, with `netmap-vpn-<src>-to-<dst>-<timestamp>.<ext>` for single-source/single-destination output and `netmap-vpn-<org>-all-<timestamp>.<ext>` for aggregate output.
- Generated output files can now be directed into a selected directory with `-od`, while the current working directory remains the default output location.
- `netmap version` now prints `v2.0.0`, and release documentation now references `v2.0.0` with `VERSION=2.0.0` as the raw source of truth.

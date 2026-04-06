# netmap

`netmap` is a opinionated CLI built in G0 1.26 that resolves GCP projects from a local YAML inventory and maps either source dedicated interconnects or source VPN resources to their connected destination resources, Cloud Routers, interfaces, and BGP peers.

## Requirements

- Go `1.26.x` to build the binary
- Google Application Default Credentials available in the environment
- IAM permissions that allow listing interconnects, interconnect attachments, VPN gateways, VPN tunnels, routers, and router status

## Config

The CLI expects a YAML file with this structure:

```yaml
org:
  - name: dbc
    workload:
      - name: native
        env:
          - name: dev
            project_id: project
```

`project_id` is the destination project.

Selector expansion is:

```text
-o only: iterate all workloads and all environments under that org
-o and -w without -e: iterate all environments for that workload
-o and -e without -w: iterate all workloads containing that environment
-o, -w, and -e: resolve one exact workload/environment tuple
```

## Build

```bash
/usr/local/go/bin/go build ./cmd/netmap
```

## Version

```bash
./netmap version
./netmap
./netmap -h
```

```text
./netmap version prints the embedded CLI version, currently v2.0.0.
./netmap and ./netmap -h print the help menu without requiring ADC credentials.
All CLI parameter checks and config-file semantic validation complete before netmap initializes the Compute provider, checks ADC validity, or talks to the Compute API.
```

## Run Without Building

```bash
/usr/local/go/bin/go run ./cmd/netmap \
  -t interconnect \
  -o dbc \
  -w native \
  -e dev \
  -p src-project
```

If the default Go build cache location is not writable, run with an explicit cache path:

```bash
GOCACHE=/tmp/go-build-cache /usr/local/go/bin/go run ./cmd/netmap \
  -t interconnect \
  -o dbc \
  -w native \
  -e dev \
  -p src-project
```

## Test

```bash
/usr/local/go/bin/go test ./...
```

## Release

- `VERSION` is the release source of truth and currently contains `2.0.0`
- The release workflow prepends `v` when creating and checking release tags, so `VERSION=2.0.0` produces release tag `v2.0.0`
- The GitHub release title is also set to `v2.0.0`
- PR validation runs in workflow `NetMap Test` from `.github/workflows/netmap-test.yml`
- Release publishing runs in workflow `NetMap Release` from `.github/workflows/netmap-release.yml`
- The GitHub Actions job labels are:
  - `Code Test`
  - `Build and Release`
- Pull requests targeting `main` run `go test ./...` when changes land under `cmd/` or `internal/`
- Pull request checks are the only test gate; the release workflow does not re-run tests after merge
- The release workflow runs only after the pull request is merged into `main`
- The release workflow validates `VERSION` and fails in the build job if that version already exists
- Release artifacts are:
  - `netmap_<version>_darwin_amd64.tar.gz`
  - `netmap_<version>_windows_amd64.zip`
- Each archive contains only `README.md` and the platform binary
- `CHANGELOG.md` contains the shipped release history for versions such as `v2.0.0` and is used as the GitHub release notes source

## Usage

The usage text shown by `./netmap` and `./netmap -h` is sourced from the editable embedded file [internal/app/usage.txt](/home/macho_prawn/gh-repo/netmap/internal/app/usage.txt).

```bash
./netmap \
  -t interconnect \
  -o dbc \
  -w native \
  -e dev \
  -p src-project \
  -c config.yaml
```

### Flags

```text
-t  mandatory, accepts interconnect or vpn
-o  mandatory, org lookup key from the YAML config
-w  optional, workload selector; with -o and no -e, expands all environments in that workload
-e  optional, environment selector; with -o and no -w, expands all workloads containing that environment
-p  mandatory only for -t interconnect; source project containing dedicated interconnects
-f  optional, output format override: html, csv, tsv, json, or tree
-c  optional, defaults to config.yaml
-h  optional, prints usage
```

## Behavior

### `-t interconnect`

```text
Selector expansion:
  -o only: all workloads and environments under that org
  -o + -w: all environments in that workload
  -o + -e: all workloads containing that environment
  -o + -w + -e: one exact tuple

Behavior:
  Lists dedicated interconnects in the source project.
  Fails if the source project has no dedicated interconnects.
  Lists destination VLAN attachments and Cloud Routers across regions.
  Maps router interfaces and BGP peers where available.
  Uses a shared hierarchy in JSON, tree, Mermaid, and HTML output rooted at org -> workload -> environment -> src_project -> src_interconnect.
  Uses one canonical field set across outputs, matching the CSV/TSV column names.
  Writes progress to stderr as an ASCII 2-column task table with one timed row per resolved org/workload/environment tuple, a Braille spinner on the active row, completed rows, and a merged final summary row.
  Validates CLI parameters and config selector semantics before initializing ADC-backed Compute discovery.
  Collapses identical environment, src_project, src_interconnect, and dst_region labels into shared Mermaid graph nodes.
  Uses Google Cloud Go libraries and ADC instead of shelling out to gcloud.
```

### `-t vpn`

```text
-p is rejected.
Resolves selected config tuples as source projects.
Lists source HA and Classic VPN gateways and connected VPN tunnels.
Uses HA tunnel peer gateway references to discover destination GCP projects, VPN gateways, tunnels, and Cloud Routers.
Includes Classic VPN gateways and tunnels as source-side unmapped output when no peer GCP project can be discovered.
Uses a VPN-specific hierarchy across JSON, tree, Mermaid, and HTML output:
  org -> workload -> environment -> src_project -> src_region -> src_vpn_gateway -> src_tunnel -> src_cloud_router -> bgp_peering_status -> dst_cloud_router -> dst_tunnel -> dst_vpn_gateway -> dst_region -> dst_project
Uses a VPN-specific Mermaid grouping strategy that shares repeated `src_project -> src_region[src_vpc]` nodes and collapses identical destination gateway/region/project paths within the same source-tunnel branch.
Uses the same csv, tsv, json, tree, mermaid, and html output formats as interconnect reports.
```

## Output

| Case | Interconnect | VPN |
| --- | --- | --- |
| Default output with omitted `-f` | `netmap-interconnect-<src>-to-<dst>-<timestamp>.mmd` | `netmap-vpn-<org>-<selector>-<timestamp>.mmd` |
| Explicit format extensions | `html`, `csv`, `tsv`, `json`, `tree.txt` on the `netmap-interconnect-<src>-to-<dst>-<timestamp>` base | `html`, `csv`, `tsv`, `json`, `tree.txt` on the `netmap-vpn-<org>-<selector>-<timestamp>` base |
| Aggregate output | `netmap-interconnect-<src>-to-<org>-all-<timestamp>.<ext>` | `netmap-vpn-<org>-all-<timestamp>.<ext>` |
| Selector naming | n/a | `-o/-w/-e`: `<workload>-<env>`; `-o` only: `all`; `-o/-w`: `<workload>-all`; `-o/-e`: `all-<env>` |

`-f html` writes a self-contained offline Mermaid viewer page that can be opened directly in a browser.

Structured `json`, `tree`, `mermaid`, and `html` outputs use branch-scoped unprefixed field names in visible labels and JSON leaf keys. Flat `csv` and `tsv` outputs remain canonical and keep `src_` / `dst_` prefixes.

On success, the CLI prints a merged final summary row containing:

```text
Output: <path>
Total Time: <duration>
```

Use that path directly when opening or sharing the generated file.

### CSV/TSV columns

#### Interconnect

```text
org,workload,environment,src_project,src_interconnect,mapped,src_region,src_state,src_macsec_enabled,src_macsec_keyname,dst_project,dst_region,dst_vpc,dst_vlan_attachment,dst_vlan_attachment_state,dst_vlan_attachment_vlanid,dst_cloud_router,dst_cloud_router_asn,dst_cloud_router_interface,dst_cloud_router_interface_ip,remote_bgp_peer,remote_bgp_peer_ip,remote_bgp_peer_asn,bgp_peering_status
```

#### VPN

```text
org,workload,environment,src_project,src_region,src_vpn_gateway,src_vpn_gateway_type,src_cloud_router,src_cloud_router_asn,src_cloud_router_interface,src_cloud_router_interface_ip,src_vpn_tunnel,src_vpn_gateway_interface,src_vpn_gateway_ip,src_vpn_tunnel_status,bgp_peering_status,dst_vpn_tunnel,dst_vpn_gateway_interface,dst_vpn_gateway_ip,dst_vpn_tunnel_status,dst_cloud_router,dst_cloud_router_asn,dst_cloud_router_interface,dst_cloud_router_interface_ip,dst_vpn_gateway,dst_vpn_gateway_type,dst_region,dst_project
```

## Notes

- Source dedicated interconnects are modeled as global resources
- Source dedicated interconnect MACsec status and key name are emitted in the canonical source field block when available
- Destination VLAN attachments and Cloud Routers are modeled as regional resources
- Destination VPC is derived from the destination VLAN attachment network and emitted as `dst_vpc` in flat output and `vpc` in structured output
- Destination Cloud Router ASN is emitted in the canonical destination router field block when available
- Remote BGP peer ASN is emitted only in interconnect output
- Unmapped source interconnects are still included in the output
- Unmapped source Classic VPN gateways and tunnels are also included in the output when peer GCP project discovery is unavailable
- Mermaid output is a shared-node DAG and may intentionally collapse matching labels across workload, environment, source-project, interconnect, and destination-region layers
- VPN Mermaid output uses a separate node-key strategy that collapses repeated `src_project -> src_region[src_vpc]` pairs and reuses identical destination gateway/region/project nodes within the same source-tunnel branch
- In structured VPN output, source and destination region nodes render as `region [vpc: ...]`
- VPN Mermaid renders `bgp_peering_status` as its own node between source and destination `cloud_router` nodes
- Mermaid labels use `<br>` line breaks so they render correctly in Mermaid-compatible viewers, including the offline HTML output
- Runtime discovery is performed with Google Compute API clients, not the `gcloud` CLI

<p align="center"><sub>Vibe-Coded with &#x2665;&#xFE0E;</sub></p>

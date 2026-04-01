# netmap

`netmap` is a Go 1.26 CLI that resolves destination GCP projects from a local YAML inventory and maps source dedicated interconnects to destination VLAN attachments, Cloud Routers, interfaces, and BGP peers.

## Requirements

- Go `1.26.x` to build the binary
- Google Application Default Credentials available in the environment
- IAM permissions that allow listing interconnects, interconnect attachments, routers, and router status

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

- `-o` only: iterate all workloads and all environments under that org
- `-o` and `-w` without `-e`: iterate all environments for that workload
- `-o` and `-e` without `-w`: iterate all workloads containing that environment
- `-o`, `-w`, and `-e`: resolve one exact workload/environment tuple

## Build

```bash
/usr/local/go/bin/go build ./cmd/netmap
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

- `VERSION` is the release source of truth and currently contains `1.0.0`
- Pull requests targeting `main` run `go test ./...` when changes land under `cmd/` or `internal/`
- Pull request checks are the only test gate; the release workflow does not re-run tests after merge
- The release workflow runs only after the pull request is merged into `main`
- The release workflow validates `VERSION` and fails in the build job if that version already exists
- Release artifacts are:
  - `netmap_<version>_darwin_amd64.tar.gz`
  - `netmap_<version>_windows_amd64.zip`
- Each archive contains only `README.md` and the platform binary
- `CHANGELOG.md` contains the detailed release history for shipped versions

## Usage

```bash
./netmap \
  -t interconnect \
  -o dbc \
  -w native \
  -e dev \
  -p src-project
```

### Flags

- `-t` mandatory, accepts `interconnect` or `vpn`
- `-o` mandatory, org lookup key from the YAML config
- `-w` optional, workload selector; with `-o` and no `-e`, expands all environments in that workload
- `-e` optional, environment selector; with `-o` and no `-w`, expands all workloads containing that environment
- `-p` mandatory only for `-t interconnect`; source project containing dedicated interconnects
- `-f` optional, output format override: `csv`, `tsv`, `json`, or `tree`
- `-config` optional, defaults to `config.yaml`
- `-h` optional, prints usage

## Behavior

### `-t interconnect`

- Expands selectors as follows:
  - `-o` only: all workloads and environments under that org
  - `-o` + `-w`: all environments in that workload
  - `-o` + `-e`: all workloads containing that environment
  - `-o` + `-w` + `-e`: one exact tuple
- Lists dedicated interconnects in the source project
- Fails if the source project has no dedicated interconnects
- Lists destination VLAN attachments and Cloud Routers across regions
- Maps router interfaces and BGP peers where available
- Uses a shared hierarchy in JSON/tree/Mermaid output rooted at `org -> workload -> environment -> src_project -> src_interconnect`
- Uses one canonical field set across outputs, matching the CSV/TSV column names
- Writes progress to `stderr` as an ASCII 2-column task table with one timed row per resolved org/workload/environment tuple, a Braille spinner on the active row, `✅ Completed ...` rows when tasks finish, and a merged final summary row
- In Mermaid output, identical `environment`, `src_project`, `src_interconnect`, and `dst_region` labels are collapsed into shared nodes so multiple parent branches can fan into the same box before the graph fans back out
- Uses Google Cloud Go libraries and ADC instead of shelling out to `gcloud`

### `-t vpn`

- Rejects `-p`
- Returns a clear `vpn is not implemented yet` message

## Output

If `-f` is not provided, the CLI writes Mermaid output by default:

```text
netmap-interconnect-<src-project>-to-<dst-project>-<timestamp>.mmd
```

If `-f` is provided, Mermaid is suppressed and only the selected format is written:

- Exact match mode:
  - `-f csv` -> `netmap-interconnect-<src>-to-<dst>-<timestamp>.csv`
  - `-f tsv` -> `netmap-interconnect-<src>-to-<dst>-<timestamp>.tsv`
  - `-f json` -> `netmap-interconnect-<src>-to-<dst>-<timestamp>.json`
  - `-f tree` -> `netmap-interconnect-<src>-to-<dst>-<timestamp>.tree.txt`
- Org fanout mode:
  - `netmap-interconnect-<src>-to-<org>-all-<timestamp>.<ext>`

On success, the CLI prints a merged final summary row containing:

```text
Output: <path>
Total Time: <duration>
```

Use that path directly when opening or sharing the generated file.

Mermaid output can be viewed in `https://mermaid.live`.

### CSV/TSV columns

```text
org,workload,environment,src_project,src_interconnect,mapped,src_region,src_state,src_macsec_enabled,src_macsec_keyname,dst_project,dst_region,dst_vpc,dst_vlan_attachment,dst_vlan_attachment_state,dst_vlan_attachment_vlanid,dst_cloud_router,dst_cloud_router_asn,dst_cloud_router_interface,dst_cloud_router_interface_ip,remote_bgp_peer,remote_bgp_peer_ip,remote_bgp_peer_asn,bgp_peering_status
```

## Notes

- Source dedicated interconnects are modeled as global resources
- Source dedicated interconnect MACsec status and key name are emitted in the canonical source field block when available
- Destination VLAN attachments and Cloud Routers are modeled as regional resources
- Destination VPC is derived from the destination VLAN attachment network and emitted as `dst_vpc`
- Destination Cloud Router ASN is emitted in the canonical destination router field block when available
- Remote BGP peer ASN is emitted in the canonical peer field block when available
- Unmapped source interconnects are still included in the output
- Mermaid output is a shared-node DAG and may intentionally collapse matching labels across workload, environment, source-project, interconnect, and destination-region layers
- In Mermaid, shared VPCs are folded into the region node; mixed VPCs are rendered as separate `dst_vpc` nodes between region and attachment
- Mermaid renders `bgp_peering_status` as its own node between the interface and remote peer nodes
- Mermaid labels use `<br>` line breaks so they render correctly in `https://mermaid.live`
- Runtime discovery is performed with Google Compute API clients, not the `gcloud` CLI

<p align="center"><sub>Vibe-Coded with &#x2665;&#xFE0E;</sub></p>

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
- Writes progress messages to `stderr`, including a `⏳` start line, one completion line per resolved org/workload/environment tuple, and a final `output file: <path>` line on success
- In Mermaid output, each `org -> workload -> environment -> src_project` branch fans out to dedicated interconnects first, then collapses to one `dst_project` box and one `dst_region` box only when those values are shared across every mapped interconnect in that branch
- Uses Google Cloud Go libraries and ADC instead of shelling out to `gcloud`

### `-t vpn`

- Rejects `-p`
- Returns a clear `vpn is not implemented yet` message

## Output

If `-f` is not provided, the CLI writes a Mermaid file:

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

### CSV/TSV columns

```text
org,workload,environment,src_project,src_interconnect,mapped,src_region,src_state,dst_project,dst_region,dst_vlan_attachment,dst_vlan_attachment_state,dst_vlan_attachment_vlanid,dst_cloud_router,dst_cloud_router_interface,dst_cloud_router_interface_ip,remote_bgp_peer,remote_bgp_peer_ip,bgp_peering_status
```

## Notes

- Source dedicated interconnects are modeled as global resources
- Destination VLAN attachments and Cloud Routers are modeled as regional resources
- Unmapped source interconnects are still included in the output
- Mermaid output preserves `org -> workload -> environment` branch uniqueness even when different branches resolve to the same destination project or region labels
- Runtime discovery is performed with Google Compute API clients, not the `gcloud` CLI

<p align="center"><sub>Vibe-Coded with &#x2665;&#xFE0E;</sub></p>

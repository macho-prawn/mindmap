# mindmap

`mindmap` is a Go 1.26 CLI that resolves a destination GCP project from a local YAML inventory and maps source dedicated interconnects to destination VLAN attachments, Cloud Routers, interfaces, and BGP peers.

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

## Build

```bash
/usr/local/go/bin/go build ./cmd/mindmap
```

## Run Without Building

```bash
/usr/local/go/bin/go run ./cmd/mindmap \
  -t interconnect \
  -o dbc \
  -w native \
  -e dev \
  -p src-project
```

If the default Go build cache location is not writable, run with an explicit cache path:

```bash
GOCACHE=/tmp/go-build-cache /usr/local/go/bin/go run ./cmd/mindmap \
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
./mindmap \
  -t interconnect \
  -o dbc \
  -w native \
  -e dev \
  -p src-project
```

### Flags

- `-t` mandatory, accepts `interconnect` or `vpn`
- `-o` mandatory, org lookup key from the YAML config
- `-w` mandatory, workload lookup key from the YAML config
- `-e` mandatory, environment lookup key from the YAML config
- `-p` mandatory only for `-t interconnect`; source project containing dedicated interconnects
- `-f` optional, output format override: `csv`, `tsv`, `json`, or `tree`
- `-config` optional, defaults to `config.yaml`

## Behavior

### `-t interconnect`

- Resolves the destination project from `-o`, `-w`, and `-e`
- Lists dedicated interconnects in the source project
- Fails if the source project has no dedicated interconnects
- Lists destination VLAN attachments and Cloud Routers across regions
- Maps router interfaces and BGP peers where available
- Includes `region` in every destination-side record
- Uses `src_region=global` for source dedicated interconnects
- Uses Google Cloud Go libraries and ADC instead of shelling out to `gcloud`

### `-t vpn`

- Rejects `-p`
- Returns a clear `vpn is not implemented yet` message

## Output

If `-f` is not provided, the CLI writes a Mermaid file:

```text
mindmap-interconnect-<src-project>-to-<dst-project>.mmd
```

If `-f` is provided, Mermaid is suppressed and only the selected format is written:

- `-f csv` -> `mindmap-interconnect-<src>-to-<dst>.csv`
- `-f tsv` -> `mindmap-interconnect-<src>-to-<dst>.tsv`
- `-f json` -> `mindmap-interconnect-<src>-to-<dst>.json`
- `-f tree` -> `mindmap-interconnect-<src>-to-<dst>.tree.txt`

### CSV/TSV columns

```text
src_project,src_interconnect,src_region,src_state,dst_project,region,attachment,attachment_state,router,interface,bgp_peer_name,local_ip,remote_ip,bgp_status,mapped
```

## Notes

- Source dedicated interconnects are modeled as global resources
- Destination VLAN attachments and Cloud Routers are modeled as regional resources
- Unmapped source interconnects are still included in the output
- Runtime discovery is performed with Google Compute API clients, not the `gcloud` CLI

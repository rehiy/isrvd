# iSrvd

English | [简体中文](README_CN.md)

> The name comes from *"it is a server daemon"*: `srv` follows the Linux convention of `/srv`, and `d` stands for daemon. **Pronunciation**: **"I served"** (/ˈaɪ sɜːrvd/).

A lightweight operations panel built with Go + Vue 3. It integrates file management, Docker container orchestration, APISIX/Caddy gateway configuration, a web terminal, GPU monitoring, scheduled tasks and an AI assistant, providing an all-in-one management experience for personal servers and small to medium-sized teams.

## Contents

- [Features](#features)
- [Tech Stack](#tech-stack)
- [Deployment](#deployment)
- [Centralized Management](#centralized-management)
- [Configuration](#configuration)
- [Permissions](#permissions)
- [GPU Monitoring](#gpu-monitoring)
- [Local Development](#local-development)
- [Architecture](#architecture)
- [Security Features](#security-features)
- [License](#license)

## Features

| Module | Capabilities |
| ------ | ------ |
| System Overview | CPU, memory, disk, network, Go runtime and GPU monitoring, with history collection, service availability probing and online upgrade |
| File Management | Browse, upload, download, edit, create/delete directories, rename, change permissions, compress/extract |
| Web Terminal | Shell terminal based on xterm.js, with container terminal access |
| Local Processes | View the host process list (CPU, memory, command line) and kill a process (audited, forced) |
| SSH Remote Management | Manage hosts and reusable credentials, with password/private key auth, browser terminal and SFTP file management |
| Centralized Management | The same binary runs as standalone, center or managed node via `ISRVD_MODE`; one console manages many servers, see [Centralized Management](#centralized-management) |
| AI Assistant | Built-in Copilot based on CopilotKit + AG-UI, calling backend APIs through the built-in OpenAPI catalog, with page context, tool cards and write-operation approval; works with any OpenAI-compatible LLM |
| Scheduled Tasks | Cron scheduling; Shell or BAT/PowerShell scripts and executables depending on the platform, plus ephemeral or existing container execution when Docker is available |
| APISIX | Routes, consumers, upstreams, SSL certificates, plugin configs (PluginConfig), plugin list, access authorization |
| Caddy | HTTP services, routes, Basic Auth, SSL certificates, global options, with raw config editing |
| Docker | Containers, images, networks, volumes, registry management; container files, live logs, resource stats, terminal access; image build/push/pull |
| Swarm | Cluster info, nodes, services, tasks; service logs, forced update, join token management |
| Compose | File editing, Docker Compose / Swarm Stack deploy and redeploy, up to 10 deployment records with encrypted config snapshots, redeploy from historical config |
| Member Management | Multiple users, isolated home directories, route-level permissions, API tokens, TOTP two-factor and passwordless Passkey login |
| System Management | Grouped configuration, operation audit log, resource and application failure alerts with Webhook notifications, OIDC integration, proxy header authentication |
| Mobile | Responsive layout adapted for mobile devices |

## Tech Stack

| Layer | Technology |
| ------ | ------ |
| Backend | Go 1.26+ / Gin / golang-jwt |
| Frontend | Vue 3 / TypeScript / Tailwind CSS / Pinia |
| Terminal | xterm.js |
| Containers | Docker / APISIX / Caddy |
| AI | Any OpenAI-compatible LLM; frontend based on CopilotKit + AG-UI |

## Deployment

Three options: install script (recommended), Docker image, or binary package. The install script targets Linux with systemd and must be run as root.

### Image Variants

| Image | Description |
| ------ | ------ |
| `rehiy/isrvd:slim` | **Default**, isrvd only, suitable for most use cases |
| `rehiy/isrvd:apisix` | isrvd + APISIX, with the API gateway integrated |
| `rehiy/isrvd:caddy` | isrvd + Caddy, with reverse proxy and TLS management integrated |

The CNB pipeline also pushes to the CNB Docker registry at `docker.cnb.cool/<repo-slug>:<tag>`, with the `slim`, `caddy` and `apisix` tags; `slim` also serves as `latest`.
For Docker deployments in mainland China, replace `rehiy/isrvd:<tag>` in the examples below with `docker.cnb.cool/rehiy/isrvd:<tag>`.

The Docker edition ships with the default admin account `admin` / `admin`; the first successful login redirects to the change-password page.

### Script Install (recommended)

Two standalone scripts are provided; neither downloads or loads the other. Both accept `--auto` (default), `--cn`, or `--global` to select the source:

- **System service + component containers (`isrvd-system.sh`)**: isrvd runs as a systemd service. Select separate Caddy, APISIX, or etcd containers with `--components`. APISIX includes etcd; `caddy,etcd` is also supported. Caddy and APISIX cannot be selected together because both use ports 80/443. Components use host networking, with admin APIs and etcd bound to `127.0.0.1`, and persist data in `/srv/data` by default. Set `--data PATH` during installation to customize the component data directory. Docker is not required, and `--data` is not accepted, when no components are selected.
- **Fully containerized (`isrvd-docker.sh`)**: prepares Docker, a single-node Swarm and the attachable overlay network `sdnet`, then starts the selected all-in-one `isrvd:tag` image. The default tag is `slim`. Built-in `caddy` and `apisix` tags select their port layouts automatically; use `--variant slim|caddy|apisix` for a custom tag.

```bash
# System service + Caddy (custom data directory), or APISIX / etcd (default /srv/data)
bash <(curl -fsSL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-system.sh) install --components caddy --data /mnt/isrvd-data
bash <(curl -fsSL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-system.sh) install --components apisix

# Fully containerized: select slim / caddy / apisix as needed
bash <(curl -fsSL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-docker.sh) install --tag slim
bash <(curl -fsSL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-docker.sh) install --tag caddy
bash <(curl -fsSL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-docker.sh) install --tag apisix

# Replace YOUR_PUBLISHED_TAG with a tag already published in the selected registry
bash <(curl -fsSL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-docker.sh) install --tag YOUR_PUBLISHED_TAG --variant apisix
```

`isrvd-system.sh install --caddy` / `--apisix` now select separate components. For all-in-one containers, use `isrvd-docker.sh install --caddy` / `--apisix`. The `--systemd` and `--docker` flags are optional in their respective scripts. The main container is named `isrvd`; separate components are named `isrvd-caddy`, `isrvd-apisix` and `isrvd-etcd`.

The component data directory must be an absolute non-root path, without `.` or `..` segments, colons, backslashes or control characters, and must not overlap `/usr/local/isrvd`. The script resolves paths using `realpath`; component subdirectories and configuration files must not redirect through symbolic links. Caddy uses `caddy` (certificates and other storage) and `caddy-config` (runtime configuration), etcd uses `etcd`, and component startup configuration uses `conf/components` under the selected directory. The main isrvd configuration remains in `/usr/local/isrvd/config.yml`. `--data` applies only to new installations and does not migrate existing data.

Use the matching script for updates and uninstall; deployment methods are no longer detected automatically. `isrvd-system.sh update` updates the binary and owned components, restoring the component data directory from their actual bind mounts; inconsistent mounts abort before components are stopped. `isrvd-docker.sh update` rebuilds the main container. Existing configuration and role are preserved. Docker updates preserve the image tag; use `isrvd-docker.sh update --tag YOUR_PUBLISHED_TAG --variant apisix` to change it. Failed container replacements restore the previous container. Components retain their pinned image versions rather than automatically upgrading across versions. `isrvd-system.sh uninstall` removes the system service and owned components while preserving the actual component data directory; `isrvd-docker.sh uninstall` removes only the main container and preserves `/srv/data`. Back up `/usr/local/isrvd/config.yml` before uninstalling the system service.

The online endpoint `/install/` (or `?script=systemd`) serves the system-service script; `/install/?script=docker` serves the Docker script. They use separate caches. Release packages contain both scripts; older local installers may reject the new package entries, so use the latest online script for updates.

For systemd + APISIX, optionally provide `ISRVD_APISIX_ADMIN_KEY` through the environment (32-128 letters, digits, `_` or `-`, not numeric-only); otherwise the installer generates it. The script passes the same key to the APISIX container through its environment and writes `apisix.adminKey` in root-only `/usr/local/isrvd/config.yml` (0600), with the Admin URL `http://127.0.0.1:9180/apisix/admin`. It never prints the key or relies on a Go environment override. Updates retain the existing component key and repair the previously missing Admin path and credential without rotating keys from the installer environment. Protect the config file and its backups. Component startup configuration is in `conf/components` under the selected data directory (default `/srv/data/conf/components`); APISIX readiness also verifies its authenticated Admin API, so installation fails and rolls back if a preserved configuration does not match the supplied key.

System-service installation validates credentials, component images and the archive before extraction. Later failures or catchable interruptions roll back newly created binaries, links, service files and owned containers while preserving component data, allowing a retry. If a forced termination leaves `isrvd-COMPONENT-update-backup`, subsequent updates stop before stopping the service; verify ownership and restore the backup manually, or use `isrvd-system.sh uninstall` to remove owned normal and backup containers before reinstalling. Built-in `slim` / `latest` / `caddy` / `apisix` tags must match their port layouts: both installation and updates reject a conflicting `--variant`. Custom tags may specify their layout explicitly.

The script reads `ISRVD_MODE` (`server` by default, `center`, or `agent`) from the installer environment. Export it before running `install`; managed nodes also require exported `ISRVD_CENTER_URL` and `ISRVD_NODE_TOKEN`. Installation saves these values in a root-only systemd drop-in (`installer.conf`) or passes them to Docker. A center enables port forwarding in the config file (`portmap.range`, for example `20000-20999`, and `portmap.bind`); for Docker centers set `portmap.bind: 0.0.0.0` and run `update`, which publishes the `portmap.range` range (TCP and UDP, keep it small). Docker agent installations do not publish port 8080. Updates preserve the installed role. To change roles later, recreate the Docker container or edit the systemd environment and restart.

### Prepare the Network

Not needed when using the install script. For a manual Docker deployment, initialize Swarm first and then create the attachable overlay network:

```bash
docker swarm init
docker network create --driver=overlay --attachable sdnet
```

### Run the Container

| Image | Port mapping | Description |
| ------ | ---------- | ------ |
| `slim` | 8080 → 8080 | isrvd web UI |
| `apisix` | 8080 → 8080, 80 → 9080, 443 → 9443 | isrvd UI; APISIX HTTP / HTTPS proxy |
| `caddy` | 8080 → 8080, 80 → 80, 443 → 443 | isrvd UI; Caddy HTTP / HTTPS proxy |

#### slim (default)

isrvd only, the smallest image, for setups that need file management, Docker/Swarm/Compose, scheduled tasks and the like.

```bash
docker run -d \
  --name isrvd \
  --network sdnet \
  -p 8080:8080 \
  -v /srv/data:/data \
  -v /var/run/docker.sock:/var/run/docker.sock \
  rehiy/isrvd:slim
```

#### apisix (API gateway integrated)

isrvd + APISIX, for setups already using APISIX as the API gateway.

```bash
docker run -d \
  --name isrvd \
  --network sdnet \
  -p 8080:8080 \
  -p 80:9080 \
  -p 443:9443 \
  -v /srv/data:/data \
  -v /var/run/docker.sock:/var/run/docker.sock \
  rehiy/isrvd:apisix
```

#### caddy (reverse proxy integrated)

isrvd + Caddy, for setups needing a reverse proxy, automatic HTTPS (ACME) or unified gateway management.

```bash
docker run -d \
  --name isrvd \
  --network sdnet \
  -p 8080:8080 \
  -p 80:80 \
  -p 443:443 \
  -v /srv/data:/data \
  -v /var/run/docker.sock:/var/run/docker.sock \
  rehiy/isrvd:caddy
```

Caddy's default HTTP service listens on `:80` and `:443`, with automatic HTTPS and redirects disabled. To enable HTTPS, edit the HTTPS behavior of the service under **Caddy → Services**, or edit the **raw config** directly.

### Docker Compose

```yaml
services:
  isrvd:
    image: rehiy/isrvd:slim
    container_name: isrvd
    restart: unless-stopped
    networks:
      - sdnet
    ports:
      - "8080:8080"
    volumes:
      - /srv/data:/data
      - /var/run/docker.sock:/var/run/docker.sock

networks:
  sdnet:
    external: true
```

> **Note**:
>
> - Create the `sdnet` network first (see "Prepare the Network" above); Compose references the existing network via `external: true`
> - Always mount the whole `/data` directory to avoid data loss when the container is recreated

### Binary Deployment

- Directory: `/usr/local/isrvd/`, containing the binary and the config file
- Limitation: cannot reach other containers through the container network

```bash
# one-line install (picks the CNB or GitHub source by IP; use --cn / --global to force)
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-system.sh) install
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-system.sh) install --cn

# update / uninstall / download only
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-system.sh) update
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-system.sh) update --global
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-system.sh) uninstall
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-system.sh) download
```

Binary updates download and check the archive before stopping the service, and preserve `config.yml`. Docker updates keep the previous container until the replacement stays running through the initial startup check; on failure, the previous container is restored. If an interrupted update leaves `isrvd-update-backup`, resolve that container before retrying. Use `bash isrvd-system.sh --help` for available commands and options.

The config file location is set via `ISRVD_CONFIG`, see [Configuration](#configuration).

## Centralized Management

One binary, one role selected by the `ISRVD_MODE` environment variable, one console for many servers:

| Mode | Role |
| ------ | ------ |
| `server` (default) | Standalone |
| `center` | Center: serves the web UI, accounts and permissions, accepts managed nodes and forwards requests per node |
| `agent` | Managed node: exposes no inbound port, connects out to the center and executes forwarded requests locally |

```text
Browser ──> center ──┬── pages, accounts, permissions, audit ──> in-process isrvd
                     └── /n/<nodeID>/api/… ──tunnel──> agent (in-process isrvd)
                                                       ▲ the managed node dials out; the center never connects out
```

- **No inbound port needed**: a managed node only needs to reach the center, so it can sit behind NAT or a firewall; after a disconnect it reconnects with exponential backoff
- **Switching node switches the view**: once an authorized member picks a node in the page header, Docker, Swarm, Compose, files, local processes, terminal, scheduled tasks, APISIX, Caddy and the system overview all act on that node, including live logs, terminal and large file upload/download
- **Accounts and config stay on the center**: members, permissions, system config, the AI assistant and SSH remote management are always handled by the center; operations on a node use the center's login session
- **Shown only when relevant**: the node switcher and the sidebar "集中管控" entry appear in `center` mode for members with `GET /api/node/nodes` permission

### Enrolling a Node

1. Set `ISRVD_MODE=center` in the center process environment, then restart.
2. Create a node with a required name in the center's node management page. Save the token and copy the installation command shown in the creation dialog; the node starts as pending.
3. Set the managed host environment:

```bash
export ISRVD_MODE=agent
export ISRVD_CENTER_URL=https://center.example.com
export ISRVD_NODE_TOKEN="<token generated by the center>"
```

4. Restart to connect directly with the token. The token does not expire; after connecting, the same token works for restarts and reconnects. Duplicate connections return 409 without replacing the existing session.

Center mode supports `tha`; the gateway checks the actual peer against `tha.trustedCIDRs`. Managed mode cannot enable `tha`. Deployment settings come from the process environment; YAML, etcd and the config API hold application settings. Protect token environment files with `0600` permissions.

### Container Deployment

Prepare a complete `/srv/data/conf/isrvd.yml` for application settings before starting; pass role settings through environment variables; deploy the center and managed host on their respective machines:

```bash
# center config: mode: center
docker run -d --name isrvd-center -e ISRVD_MODE=center --network sdnet -p 8080:8080 \
  -v /srv/data:/data -v /var/run/docker.sock:/var/run/docker.sock rehiy/isrvd:slim

# managed config: mode: agent; no public port
docker run -d --name isrvd-agent --network sdnet \
  -e ISRVD_MODE=agent -e ISRVD_CENTER_URL=https://center.example.com \
  -e ISRVD_NODE_TOKEN="<token generated by the center>" \
  -v /srv/data:/data -v /var/run/docker.sock:/var/run/docker.sock rehiy/isrvd:slim
```

Managed hosts cannot enable `tha`; `listenAddr` is ignored in managed mode. A managed host needs at least one founder member, otherwise forwarded requests return 401. After switching, access is only available through the center. Update the host environment and restart if the URL or token is wrong.

### systemd Deployment

Use `systemctl edit isrvd` and add `Environment="ISRVD_MODE=center"` under `[Service]`; managed nodes also need `ISRVD_CENTER_URL` and `ISRVD_NODE_TOKEN`. Run `systemctl daemon-reload` and `systemctl restart isrvd`. The installer defaults to standalone mode; export the deployment environment variables before `install` to select another role.

### Security and Boundaries

- Node management uses assignable `/api/node/...` route permissions; forwarded operations check the caller's business route permissions at the center before using the managed node's local service token. Founders retain full access. Permissions apply to all nodes; per-node grants are not supported
- Enrollment tokens do not expire. Create a node to generate an 86-character alphanumeric random token, shown only once. `node-list.yml` stores only its SHA-256 hash and a status (`pending`, `active`, or `revoked`); the first authenticated connection activates the node. Lost tokens require creating a new node. A node can be revoked at any time and disconnects immediately
- Node management operations, write operations on nodes and terminal sessions are all recorded in the center's audit page (attributed to the center's logged-in user); the managed node's own audit page attributes them to its local founder
- The center is designed as a single instance; nodes disconnect briefly on restart and reconnect automatically. Versions need not match; compatibility is negotiated by protocol version and features (see [docs/multi-node.md](docs/multi-node.md))
- Full parameter lists, APIs and behavior are documented in [docs/multi-node.md](docs/multi-node.md); API fields are in [受管节点](docs/references/node/nodes.md)

## Configuration

### Config Sources

Environment variables read by the configuration loader have leading and trailing whitespace removed; blank values use their defaults.

The config location is set by `ISRVD_CONFIG`, supporting local YAML and etcd (the value remains the same YAML as `config.yml`):

```bash
# reads ./config.yml by default
./isrvd

# local YAML
ISRVD_CONFIG=/data/conf/isrvd.yml ./isrvd

# etcd
etcdctl put /isrvd/config "$(cat /data/conf/isrvd.yml)"
ISRVD_CONFIG="etcd://user:pass@127.0.0.1:2379/isrvd/config?scheme=http&timeout=5s" ./isrvd

# when the etcd key is missing, initialize from the fallback YAML and write it to etcd
ISRVD_CONFIG="etcd://127.0.0.1:2379/isrvd/config?fallback=/data/conf/isrvd.yml" ./isrvd

# full etcd config example
# etcd://user:pass@host1:2379,host2:2379/key?scheme=http&timeout=5s&fallback=/path/config.yml
```

**etcd** credentials are optional and can also be supplied or overridden with `ETCD_USERNAME` / `ETCD_PASSWORD`. When the etcd key receives a PUT, isrvd reloads the config, registry and business services. Business data such as scheduled tasks and SSH hosts/credentials is also stored in etcd (keys like `<config key>/cron.yml`); when a business-data key is absent, isrvd imports the file of the same name under `rootDirectory`. Subsequent reads and writes use etcd; local source files remain as import copies. Audit and monitoring logs are written to local disk. Saving the local YAML through the system config API also triggers a reload immediately; editing the YAML on disk directly requires `SIGHUP` or a process restart.

### Config Items

See [配置段说明](docs/references/system/config.md#配置段说明) for the meaning of each config section.

## Permissions

Permissions are controlled per route at a fine granularity. The default `AccessPerm` routes require the member to hold the matching full route permission; the Founder is exempt. `AccessAuth` routes only require a login, and `AccessAnon` routes allow anonymous access.

**Permission format**: `<METHOD> /api/<module>/<route>` (e.g. `GET /api/docker/containers`, `POST /api/compose/docker`)

**Frontend checks**: use `portal.hasPerm('<METHOD> /api/<route>')` to control the visibility of buttons and actions

> Empty = no `AccessPerm` route permission; the available routes can be listed after login via `GET /api/account/routes`.
>
> Centralized-management node APIs are served by the center gateway. In center mode, `/api/node/...` permissions are listed by `/account/routes` and can be granted to members. Requests forwarded through `/n/<nodeID>/api/...` require the corresponding business route permission; grants apply to all nodes.

The full permission list is in [docs/permissions.md](docs/permissions.md).

## GPU Monitoring

Automatically detects NVIDIA / AMD / Intel / Apple Silicon discrete GPUs and shows utilization, memory, temperature, power draw and fan speed.

Detection methods, collected metrics and container deployment notes are in [docs/gpu-monitoring.md](docs/gpu-monitoring.md).

## Local Development

### Requirements

- **Go**: 1.26.0 or newer; the minimum version follows `go.mod`. Used for backend services and the CLI build
- **Node.js / npm**: for `webview` frontend development and build (CI uses Node.js 24)
- **Docker**: optional, for debugging Docker, Swarm and Compose features

### Start the Dev Environment

```bash
./develop.sh
```

The dev script automatically:

- **Backend**: copies `config.yml` to `.local.yml` if missing and runs `ISRVD_MODE=center ISRVD_CONFIG=.local.yml go run ./server/cmd/server`, enabling node management and switching during development
- **Frontend**: enters `webview`, installs dependencies and runs `npm run dev`
- **Port cleanup**: tries to free ports `8080` and `3000` before starting
- **Proxy**: the frontend dev server (`3000`) proxies `/api/`, `/openapi/` and the node-scoped `/n/` to `8080`; pages under `/n/<节点ID>/` are served by the frontend build embedded in the backend, so run `npm run build` to see frontend changes in the node view

On Windows:

```bat
develop.bat
```

The Windows script starts the backend in standalone mode using the root `config.yml` and runs `npm run dev`; install the frontend dependencies in `webview` first.

### Build and Checks

GitHub Actions reads the Go version from `go.mod`; CNB builds images with `golang:1.26-bookworm`. If `GOTOOLCHAIN=local` is set, the locally installed Go must satisfy the minimum version in `go.mod`.

```bash
# full distribution build
./build.sh

# compile check for all backend packages
go test ./...

# Go static analysis
go vet ./...

# frontend type check
(cd webview && npm run lint)

# frontend format and import order check
(cd webview && npm run format:check)

# frontend style consistency check
(cd webview && python3 scripts/review-style.py)

# whitespace and conflict marker check (run in the repository root)
git diff --check
```

> Please read [AGENTS.md](AGENTS.md) before contributing. It defines this repository's code conventions and collaboration guidelines.

## Architecture

Layering boundaries, package-level dependency direction and design principles are in [AGENTS.md](AGENTS.md) section "3) 项目架构"; frontend-specific conventions are in [webview/AGENTS.md](webview/AGENTS.md).

## Security Features

- JWT authentication; sensitive fields (secrets, passwords) are never returned to the frontend; SSH passwords/private keys are encrypted at rest
- File path validation to prevent directory traversal
- Archive extraction path validation to prevent Zip Slip attacks
- WebSocket connections go through the authentication middleware
- Fine-grained route-based permissions; route access levels support `0` permission required, `1` login required, `-1` anonymous
- Operation audit log; route audit levels support `0` by method, `-1` ignore, `1` always record; non-GET requests and WebSocket connections are recorded by default
- Trusted header authentication with an optional source CIDR allowlist (`tha.headerName`); `tha.trustedCIDRs` defaults to the loopback address

## License

Released under the **Apache License 2.0**, see [LICENSE](LICENSE).

Third-party component licenses are listed in [NOTICE](NOTICE).

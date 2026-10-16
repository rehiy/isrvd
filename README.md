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

The install script installs Docker, initializes a single-node Swarm, creates the attachable overlay network `sdnet`, and starts the matching all-in-one image:

```bash
# pick one: slim / caddy / apisix
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) install --docker
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) install --caddy
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) install --apisix
```

The Docker edition always uses the container name `isrvd` and stores data in `/srv/data`; `update` rebuilds the container with the current image variant and preserves `ISRVD_MODE`, `ISRVD_CENTER_URL` and `ISRVD_NODE_TOKEN`, and `uninstall` removes the container while keeping the data directory.

The script installs standalone mode by default. Set `ISRVD_MODE` (`server`, `center`, or `agent`) in the process environment; managed nodes also require `ISRVD_CENTER_URL` and `ISRVD_NODE_TOKEN`. Recreate Docker containers with these environment variables or configure systemd service environment and restart.

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
| `apisix` | 8080 → 8080、80 → 9080、443 → 9443 | isrvd UI; APISIX HTTP / HTTPS proxy |
| `caddy` | 8080 → 8080、80 → 80、443 → 443 | isrvd UI; Caddy HTTP / HTTPS proxy |

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
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) install
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) install --cn

# update / uninstall / download only
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) update
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) update --global
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) uninstall
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) download
```

The config file location is set via `ISRVD_CONFIG`, see [Configuration](#configuration).

## Centralized Management

One binary, one role selected by the `ISRVD_MODE` environment variable, one console for many servers:

| Mode | Role |
| ------ | ------ |
| `server` (default) | Standalone |
| `center` | Center: serves the web UI, accounts and permissions, accepts managed nodes and forwards requests per node |
| `agent` | Managed node: exposes no inbound port, connects out to the center and executes forwarded requests locally |

```text
浏览器 ──> center ──┬── 页面、账号、权限、审计 ──> 本进程内的 isrvd
                    └── /n/<节点ID>/api/… ──隧道──> agent（本进程内的 isrvd）
                                                  ▲ 受管机主动出站，中控从不主动外连
```

> Diagram legend (from top to bottom): the browser reaches `center`; pages, accounts, permissions and audit are handled by the in-process isrvd; `/n/<nodeID>/api/…` goes through the tunnel to the agent's in-process isrvd; the managed node dials out, the center never connects out to it.

- **No inbound port needed**: a managed node only needs to reach the center, so it can sit behind NAT or a firewall; after a disconnect it reconnects with exponential backoff
- **Switching node switches the view**: once an authorized member picks a node in the page header, Docker, Swarm, Compose, files, local processes, terminal, scheduled tasks, APISIX, Caddy and the system overview all act on that node, including live logs, terminal and large file upload/download
- **Accounts and config stay on the center**: members, permissions, system config, the AI assistant and SSH remote management are always handled by the center; operations on a node use the center's login session
- **Shown only when relevant**: the node switcher and the sidebar "节点管理" entry appear in `center` mode for members with `GET /api/node/nodes` permission

### Enrolling a Node

1. Set `ISRVD_MODE=center` in the center process environment, then restart.
2. Generate a token in the center's node management page. The dialog provides copyable environment settings; unused tokens appear as pending.
3. Set the managed host environment:

```bash
export ISRVD_MODE=agent
export ISRVD_CENTER_URL=https://center.example.com
export ISRVD_NODE_TOKEN="<token generated by the center>"
```

4. Restart to connect directly with the token. Expiry limits first use only; after connecting, the same token works for restarts and reconnects. Duplicate connections return 409 without replacing the existing session.

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

Managed hosts require a founder member and cannot enable `tha`; `listenAddr` is ignored in managed mode. After switching, access is only available through the center. Update the host environment and restart if the URL or token is wrong.

### systemd Deployment

Use `systemctl edit isrvd` and add `Environment="ISRVD_MODE=center"` under `[Service]`; managed nodes also need `ISRVD_CENTER_URL` and `ISRVD_NODE_TOKEN`. Run `systemctl daemon-reload` and `systemctl restart isrvd`. The installer starts in standalone mode by default.

### Security and Boundaries

- Node management uses assignable `/api/node/...` route permissions; forwarded operations check the caller's business route permissions at the center before using the managed node's local service token. Founders retain full access. Permissions apply to all nodes; per-node grants are not supported
- Enrollment codes are single-use and expire after 1 hour by default; node tokens are stored as hashes only, codes are short-lived and encrypted at rest, and the enrollment command stays visible in the list while pending. A node can be revoked at any time and disconnects immediately
- Node management operations, write operations on nodes and terminal sessions are all recorded in the center's audit page (attributed to the center's logged-in user); the managed node's own audit page attributes them to its local founder
- The center is designed as a single instance; nodes disconnect briefly on restart and reconnect automatically. The managed node and the center must run the same version
- Full parameter lists, APIs and behavior are documented in [docs/multi-node.md](docs/multi-node.md); API fields are in [受管节点](docs/references/node/nodes.md) and [注册码](docs/references/node/codes.md)

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

- **Backend**: copies `config.yml` to `.local.yml` if missing and runs `ISRVD_CONFIG=.local.yml go run ./server/cmd/server`; use `ISRVD_MODE=center ./develop.sh` to develop node management and switching
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

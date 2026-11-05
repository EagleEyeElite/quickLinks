# Quick Links 🔗
Quick Links is a dynamic backend service that efficiently manages and redirects URL paths.

## Prerequisites
Before you begin, ensure you have the following installed on your system:
- [Docker](https://docs.docker.com/get-docker/)
- optional: [Make](https://www.gnu.org/software/make/)

## Getting Started

### Quick Start
- To start up the application on port 80:
  ```bash
  cp .env.example .env
  docker compose --profile production up
  ```

### pgAdmin Access

**Production (k8s):** Login via Authelia OAuth2 at https://pgadmin.gated.conrad-klaus.de (Tailscale-only).
- Authentication: Authelia SSO (internal email/password login is disabled)
- Master password: `test`
- OAuth2 config: [`pgadmin/config_local.py`](pgadmin/config_local.py)
- Authelia OIDC client `pgadmin` is registered in the Authelia config (`authelia.configuration.yml`)

**Local development:** [Database](http://localhost:8082/) default credentials:
  - user: `admin@example.com`
  - pw: `admin`

  
  or with make
  ```bash
  make production
  ```

### Deploy to Kubernetes (Helm)

The whole stack — app, Postgres, and pgAdmin — is one Helm chart under `chart/`
(mirrors the `hanna` chart). Deploy/upgrade to the k3s cluster:

```bash
helm upgrade --install quick-links ./chart -n quicklinks
```

The read-only Grafana DB password is not in this repo: personal-infrastructure
owns it (Pulumi config `quicklinks:grafanaRoPassword` → Secret
`quicklinks-grafana-ro`), and the chart reads that Secret at deploy time. Render
without a cluster with `--set grafana.offlineRender=true`.

#### Rotating the Grafana password

```bash
NEW=$(openssl rand -base64 24)
# in personal-infrastructure:
pulumi config set --secret quicklinks:grafanaRoPassword "$NEW" && pulumi up
# then here:
kubectl exec -i -n quicklinks deploy/db -- sh -c \
  'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
  <<< "ALTER ROLE grafana_ro PASSWORD '$NEW';"
helm upgrade --install quick-links ./chart -n quicklinks
```

The datasource Secret updates and Grafana's sidecar reloads it.

The image must be built for the cluster's arch and pushed first:

```bash
docker buildx build --platform linux/arm64 \
  -t registry.k8s.gated.conrad-klaus.de/quick-links:latest --push .
kubectl rollout restart deployment/quick-links   # pull the new :latest
```

Config lives in [`chart/values.yaml`](chart/values.yaml) (image tags, hosts,
rate limit, resources). The Postgres/pgAdmin PVCs carry `helm.sh/resource-policy:
keep`, so `helm uninstall` never deletes your data.

### Development
- To start up the development environment with hot-reload (port `8080`):
  ```bash
  make dev
  ```

### Debugging
- To start the debugging environment with remote debugging tools (application on port `8080`, debugging on port `8083`):
  ```bash
  make debug
  ```

### Management Commands
- To stop all containers but keep volumes:
  ```bash
  make down
  ```

- To stop all containers and remove all volumes (full cleanup):
  ```bash
  make clean
  ```

### Creating secure (unguessable) links

The app looks links up by `sha256(path)` only (uniform timing, see below). The
plain path is stored next to the hash so the Grafana dashboard can show real
links; a `CHECK` constraint keeps the two consistent (`db/init_db.sql`).

Mint a 128-bit random link and get the SQL to register it:

```bash
python3 generateLinks.py add --title "Thesis QR (printed)" --host qr \
    --to https://example.com --used-on "QR code inside printed master thesis"
```

It prints the full URL and a ready-to-paste `INSERT`. `--host` (`goto`/`qr`)
only sets which domain the dashboard shows; every link works on both. Omit
`--to` for a *parked* link (404 until it gets a destination, scans still
counted). `--path home` registers a short vanity link instead — guessable by
design, so not secret.

Run the `INSERT` with:

```bash
kubectl exec -i -n quicklinks deploy/db -- sh -c \
  'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

### Security model (short version)

- **Unguessability, not secrecy of existence.** A redirector must answer
  "does this resolve?" (303 vs 404), so existence can't be hidden — instead
  paths are 128-bit random, making enumeration infeasible.
- **Uniform lookups.** Every request hashes the path and does one exact-match
  lookup on a fixed-length key, so hit/miss timing is indistinguishable.
- **Not leak-proof, on purpose.** Paths and destinations are stored in plain
  text (for the dashboard, and because the app must send the destination). A
  copy of the database reveals working links; keep backups protected.
- **Rate limiting.** A per-client Traefik `rateLimit` middleware
  (`chart/templates/app.yaml`) throttles probing, keyed on the real client IP via
  `Cf-Connecting-IP` (trustworthy behind the `cloudflare-only` allowlist).
- **No secret in logs.** The path is never logged.

### Tips for generating QR Codes

- [QR Generator](https://www.qrcode-monkey.com/)
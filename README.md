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
helm upgrade --install quick-links ./chart -n default
```

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

Redirects are keyed by `sha256(path)`, never the plaintext path (`db/init_db.sql`).
So the raw path lives only in the URL you hand out — a database or backup leak
cannot recover working links.

To mint a long, unguessable link (128-bit, cryptographically random) and get the
SQL to register it:

```bash
python3 generateLinks.py
```

Each entry prints the shareable URL plus a ready-to-paste `INSERT`. Add your real
destination in the script (or import `sql_for(path, url, label)`). For short
vanity links, `sql_for('home', 'https://…', 'home')` works too — just remember
short paths are guessable by design and not secret.

### Security model (short version)

- **Unguessability, not secrecy of existence.** A redirector must answer
  "does this resolve?" (303 vs 404), so existence can't be hidden — instead
  paths are 128-bit random, making enumeration infeasible.
- **Uniform lookups.** Every request hashes the path and does one exact-match
  lookup on a fixed-length key, so hit/miss timing is indistinguishable.
- **Rate limiting.** A per-client Traefik `rateLimit` middleware
  (`chart/templates/app.yaml`) throttles probing, keyed on the real client IP via
  `Cf-Connecting-IP` (trustworthy behind the `cloudflare-only` allowlist).
- **No secret in logs.** The path is never logged.

### Tips for generating QR Codes

- [QR Generator](https://www.qrcode-monkey.com/)
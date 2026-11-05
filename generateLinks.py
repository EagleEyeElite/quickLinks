"""Generate quick-link paths.

Two generators live here:

* generate_unique_link() — the original short, human-friendly path. It uses the
  `random` module, which is NOT cryptographically secure and produces short,
  low-entropy strings. Fine for throwaway/vanity links; do NOT use it for links
  that must be unguessable.

* generate_secure_link() — a cryptographically secure, unguessable path. Uses
  `secrets` (CSPRNG) to produce a 128-bit token (22 URL-safe chars). At 128 bits
  the keyspace is ~3.4e38, so brute-force enumeration is infeasible even before
  the server-side rate limit. This is the one to use for "secret" links.

The service looks links up by sha256(path) only (see db/init_db.sql); the plain
path is stored alongside for the dashboard. This script prints the matching SQL
INSERT with both, ready to paste into the database:

    python3 generateLinks.py add --title "Thesis QR (printed)" --host qr \
        --to https://example.com --used-on "QR code inside printed master thesis"
"""

import argparse
import hashlib
import random
import secrets

HOSTS = {"goto": "goto.conrad-klaus.de", "qr": "qr.conrad-klaus.de"}
BASE_URL = f"https://{HOSTS['goto']}"


def generate_random_string(length=5):
    characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
    random_string = ''.join(random.choice(characters) for _ in range(length))
    return random_string


def generate_unique_link(base_url=BASE_URL, length=8):
    # Short, guessable, NOT cryptographically secure. Vanity use only.
    random_string = generate_random_string(length)
    return f"{base_url}/{random_string}"


def generate_secure_path(nbytes=16):
    # 16 bytes = 128 bits of CSPRNG entropy, base64url-encoded to 22 chars.
    return secrets.token_urlsafe(nbytes)


def sql_for(path, redirect_url, label=None, description=None, display_host=HOSTS["goto"]):
    """Return the INSERT statement that registers `path` -> `redirect_url`.

    The app looks the link up by sha256(path); the plain path is stored too, for
    the dashboard only. `label` is the link's human title (defaults to NULL).
    `display_host` is the domain the link is handed out on (display only — every
    link works on both goto. and qr.).
    `description` records where the link is physically deployed (e.g.
    'BusinessCard v1') — internal-only, shown in the gated Grafana dashboard.
    `redirect_url` may be None to register a "parked" link with no live
    destination yet (the app 404s but counts the scan as a hit).
    """
    path_hash = hashlib.sha256(path.encode()).hexdigest()

    def quote(s):
        return "NULL" if s is None else "'" + s.replace("'", "''") + "'"

    return (
        "INSERT INTO redirects (path_hash, path, display_host, redirect_url, label, description) VALUES "
        f"('{path_hash}', {quote(path)}, {quote(display_host)}, {quote(redirect_url)}, "
        f"{quote(label)}, {quote(description)});"
    )


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd")
    add = sub.add_parser("add", help="mint one secure link and print its URL + INSERT")
    add.add_argument("--title", required=True, help="human title shown in the dashboard")
    add.add_argument("--host", choices=HOSTS, default="goto", help="domain it is handed out on")
    add.add_argument("--to", dest="url", help="destination URL (omit for a parked link)")
    add.add_argument("--used-on", help="where the link is physically used")
    add.add_argument("--path", help="use this path instead of a random one (vanity link)")
    args = ap.parse_args()

    if args.cmd != "add":
        ap.print_help()
        return
    path = args.path or generate_secure_path()
    host = HOSTS[args.host]
    print(f"# https://{host}/{path}")
    print(sql_for(path, args.url, args.title, args.used_on, host))


if __name__ == "__main__":
    main()

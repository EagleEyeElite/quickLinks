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

Because the service stores only sha256(path) (see db/init_db.sql), the raw path
exists only in the URL you hand out — so this script also prints the matching
SQL INSERT with the hash precomputed, ready to paste into the database.
"""

import hashlib
import random
import secrets

BASE_URL = "https://goto.conrad-klaus.de"


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


def sql_for(path, redirect_url, label=None):
    """Return the INSERT statement that registers `path` -> `redirect_url`.

    Only the sha256 hash of the path is stored; the plaintext path never touches
    the database. `label` is an optional non-secret note (defaults to NULL).
    """
    path_hash = hashlib.sha256(path.encode()).hexdigest()
    label_sql = "NULL" if label is None else f"'{label}'"
    return (
        "INSERT INTO redirects (path_hash, redirect_url, label) VALUES "
        f"('{path_hash}', '{redirect_url}', {label_sql});"
    )


def main():
    # Example: mint 10 secure, unguessable links pointing at a destination and
    # print both the shareable URL and the SQL to register it.
    destination = "https://example.com/where-this-points"
    for _ in range(10):
        path = generate_secure_path()
        url = f"{BASE_URL}/{path}"
        print(f"# {url}")
        print(sql_for(path, destination))
        print()


if __name__ == "__main__":
    main()

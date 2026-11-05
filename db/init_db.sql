-- Redirects are keyed by the SHA-256 hash of the path. The app (main.go) hashes
-- the incoming path and looks it up here by that key only, so:
--   * every lookup is an exact match on a fixed-length key, giving uniform
--     timing whether or not the path exists (no existence timing oracle).
-- redirect_url is TEXT so long destination URLs are not truncated.
-- label is the link's human title (e.g. 'Thesis QR (printed)'), shown in the
-- dashboard.
-- path is the link itself in plain text (what goes after the host), for display
-- only: the app never reads it and still looks up by path_hash alone, so the
-- timing property above holds. The CHECK keeps it consistent with path_hash.
-- Storing it means a DB leak does reveal links — accepted 2026-11-05, since the
-- destinations are plain text anyway (the app must send them).
-- display_host is the domain the link is handed out on (display only; every
-- link works on both goto. and qr.).
-- description is an internal-only note recording WHERE the link is deployed in
-- the physical world (e.g. 'BusinessCard v1', 'sticker on the DMX analyzer').
-- The app never reads it and it is never served to visitors; it surfaces only
-- in the (gated) Grafana dashboard and pgAdmin.
-- redirect_url may be NULL: a "parked" link — registered (e.g. a QR code
-- already in print) but with no live destination yet. The app answers 404 for
-- it but logs the scan as a regular hit with an empty redirect_url, so
-- pre-activation scans count in the dashboard.
CREATE TABLE redirects (
    path_hash    CHAR(64) PRIMARY KEY,   -- lowercase hex sha256(path)
    redirect_url TEXT,
    label        VARCHAR(255),
    description  TEXT,
    path         TEXT,
    display_host TEXT NOT NULL DEFAULT 'goto.conrad-klaus.de',
    created_at   TIMESTAMPTZ DEFAULT now(),
    CONSTRAINT redirects_path_matches_hash
      CHECK (path IS NULL OR encode(sha256(convert_to(path, 'UTF8')), 'hex') = path_hash)
);

-- Sample vanity links. These short paths are guessable by design (they are not
-- secret) — use generateLinks.py to mint long, unguessable ones. The hash is
-- sha256 of the path shown in the label, e.g. sha256('home').
INSERT INTO redirects (path_hash, redirect_url, label, path) VALUES
  ('4ea140588150773ce3aace786aeef7f4049ce100fa649c94fbbddb960f1da942', 'https://example.com',                    'home', 'home'),
  ('9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08', 'https://test.com',                       'test', 'test'),
  ('b82e2178ecf4d0c5859dc2e169d7fed8673f9f6eeb9f3591bed5f253b7742408', 'https://hm.example.com/creative',        'hm', 'hm'),
  ('70a0d5198ebb88f97a2cc83a236a8afcc28c7d9e6abf40c173dd54c9f45ad7f6', 'https://explore.example.com/discovery',  'ah', 'ah'),
  ('06271baf49532c879aa3c58b48671884bcc858f09197412d682750496c33e1e1', 'https://info.example.com/details',       'info', 'info'),
  ('8d9001d32c6a703d95921a77115050f33dd823d3f1730bd35215dcbecad6dc20', 'https://shop.example.com/deals',         'shop', 'shop'),
  ('19fba0e995b9794fc2c26217bf3b725c2f0d9eeda16719fe75e3ba23ca73bfc4', 'https://news.example.com/today',         'news', 'news'),
  ('30de18cc4ea2bf4601b4d97e3dc591d762e5bf00400e320c319a1f03821c6257', 'https://faq.example.com/help',           'faq', 'faq'),
  ('093e7d5fdbaacfa92448861b11075b3fba532a07d945b49a44919252f5e830ad', 'https://contact.example.com',            'contact', 'contact'),
  ('a4cc6bc01a927e2a78fd3bec51e865ac0d85e4daab6f988d5d33d056e125b1c3', 'https://privacy.example.com/policy',     'privacy', 'privacy');

-- Access log: one row per resolve attempt. Queried for "who clicked what, when,
-- how often" (the app also emits each event as a JSON line to stdout). The
-- requested path is stored verbatim — this is an intentional access log. Rows
-- are never auto-pruned; retention is a manual decision.
CREATE TABLE click_events (
    id           BIGSERIAL PRIMARY KEY,
    ts           TIMESTAMPTZ NOT NULL,
    outcome      VARCHAR(16) NOT NULL,   -- hit | miss | rejected | error (hit with NULL redirect_url = parked link)
    path         TEXT,                   -- what the visitor requested
    label        VARCHAR(255),           -- resolved link's label, on a hit
    redirect_url TEXT,                   -- where it forwarded to AT CLICK TIME, on a hit
    client_ip    TEXT,                   -- Cf-Connecting-IP (real client)
    country      VARCHAR(8),             -- CF-IPCountry
    user_agent   TEXT,
    referrer     TEXT
);
CREATE INDEX click_events_ts_idx ON click_events (ts);
CREATE INDEX click_events_path_idx ON click_events (path);

-- Read-only role for the Grafana "Apps" dashboard: SELECT on click_events and
-- redirects (the dashboard joins redirects for description/current URL — the
-- dashboard shows titles, plain paths and destinations), nothing else.
-- LOCAL DEV ONLY password. In k8s the chart creates the role from the
-- Pulumi-owned Secret quicklinks-grafana-ro (see chart/values.yaml), never in git.
CREATE ROLE grafana_ro LOGIN PASSWORD 'grafana_ro_local_dev';
GRANT CONNECT ON DATABASE quick_links TO grafana_ro;
GRANT USAGE ON SCHEMA public TO grafana_ro;
GRANT SELECT ON click_events TO grafana_ro;
GRANT SELECT ON redirects TO grafana_ro;

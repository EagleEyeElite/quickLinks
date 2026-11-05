-- Readable link registry: store each link's path in plain text next to its hash,
-- so the (gated) Grafana dashboard can show the real links and nobody has to see
-- a hash in any UI.
--
-- What changes and what doesn't:
--   * path_hash stays the primary key and the ONLY thing the app looks up by.
--     The app still hashes the incoming path and does an exact match on a
--     fixed-length 64-char key, so the timing side channel stays closed. The
--     app binary is unchanged and never reads the new columns.
--   * We deliberately give up "a DB leak can't recover working links": the
--     destinations were always plain text (the app has to send them), and
--     click_events already logs every hit path verbatim, so that property was
--     not really held anyway. Decision 2026-11-05.
--
-- Additive and idempotent: safe to re-run, no column removed or renamed, every
-- existing link keeps resolving exactly as before.
--
--   kubectl exec -i -n quicklinks deploy/db -- sh -c \
--     'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
--     < db/migrations/2026-11-05_readable_registry.sql

BEGIN;

-- 1. path: the link as the visitor types it (after the host), e.g. 'q9vpA'.
--    Display only. NULL = not known (e.g. a secret link minted before this
--    migration that was never clicked) — fill it in by hand.
ALTER TABLE redirects ADD COLUMN IF NOT EXISTS path TEXT;

-- 2. display_host: which domain the link is handed out on, for display only.
--    Every link works on BOTH goto. and qr. — this does not restrict routing.
ALTER TABLE redirects ADD COLUMN IF NOT EXISTS display_host TEXT NOT NULL DEFAULT 'goto.conrad-klaus.de';

-- 3. created_at: when the link was registered. Existing rows stay NULL
--    (unknown) rather than pretending they were created today.
ALTER TABLE redirects ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ;
ALTER TABLE redirects ALTER COLUMN created_at SET DEFAULT now();

-- 4. Guard: a stored path must hash to its row's key, so the readable column can
--    never drift from what the app actually resolves.
ALTER TABLE redirects DROP CONSTRAINT IF EXISTS redirects_path_matches_hash;
ALTER TABLE redirects ADD CONSTRAINT redirects_path_matches_hash
  CHECK (path IS NULL OR encode(sha256(convert_to(path, 'UTF8')), 'hex') = path_hash);

-- 5. Backfill paths we can prove.
--    a) Vanity links whose label IS the path (the old convention).
UPDATE redirects
SET path = label
WHERE path IS NULL
  AND label IS NOT NULL
  AND encode(sha256(convert_to(label, 'UTF8')), 'hex') = path_hash;

--    b) Any link that was ever hit: the click log holds its path verbatim.
UPDATE redirects r
SET path = ce.path
FROM (SELECT DISTINCT path FROM click_events WHERE outcome = 'hit' AND path IS NOT NULL) ce
WHERE r.path IS NULL
  AND encode(sha256(convert_to(ce.path, 'UTF8')), 'hex') = r.path_hash;

COMMIT;

-- Links still without a path after this need a manual
--   UPDATE redirects SET path = '<path>' WHERE path_hash = '<hash>';
-- (the CHECK above rejects a wrong path). List them with:
--   SELECT label, description FROM redirects WHERE path IS NULL;

-- Migration for databases initialized before 2026-08-30 (init_db.sql only runs
-- on an EMPTY volume, so live DBs need this applied by hand):
--   kubectl exec -n default deploy/db -- psql -U user -d quick_links \
--     -f /dev/stdin < db/migrations/2026-08-30_description_and_forwarded_to.sql
--
-- 1. redirects.description — internal-only note recording WHERE the link is
--    physically deployed (e.g. 'BusinessCard v1'). Never served to visitors;
--    visible only in the gated Grafana dashboard / pgAdmin.
-- 2. click_events.redirect_url — where a hit forwarded to AT CLICK TIME, so the
--    dashboard keeps showing the historical destination even after a link is
--    repointed. Written by the app from this date on.
-- 3. grafana_ro may now also read redirects (for description / current URL in
--    the "Top links" panel). The stored hashes are non-reversible, so this
--    leaks no working secret links.
-- Everything here is additive and idempotent; the pre-migration app binary
-- keeps working unchanged (its INSERT names its columns explicitly).

ALTER TABLE redirects    ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE click_events ADD COLUMN IF NOT EXISTS redirect_url TEXT;

GRANT SELECT ON redirects TO grafana_ro;

-- Backfill historical hits with the CURRENT destination (best available guess —
-- true click-time recording only starts with the app deployed alongside this
-- migration). Joins via sha256 of the logged plaintext path.
UPDATE click_events ce
SET redirect_url = r.redirect_url
FROM redirects r
WHERE ce.outcome = 'hit'
  AND ce.redirect_url IS NULL
  AND r.path_hash = encode(sha256(convert_to(ce.path, 'UTF8')), 'hex');

-- Parked links: a registered path may have NO destination yet (QR code already
-- in print, target not live). redirect_url becomes nullable; the app (from the
-- build deployed with this migration) answers 404 for such links but logs the
-- scan as a regular hit with an empty redirect_url and the label, so
-- pre-activation scans count in the dashboard's hit numbers.
-- Safe ordering: the pre-migration binary never sees a NULL until a parked row
-- is inserted, so apply this, deploy the new binary, then insert parked rows.
--
--   kubectl exec -i -n default deploy/db -- psql -U user -d quick_links \
--     -f /dev/stdin < db/migrations/2026-08-30_parked_links.sql

ALTER TABLE redirects ALTER COLUMN redirect_url DROP NOT NULL;

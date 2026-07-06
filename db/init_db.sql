-- Redirects are keyed by the SHA-256 hash of the secret path, never the path
-- itself. The app (main.go) hashes the incoming path and looks it up here, so:
--   * a DB / backup leak cannot recover working links (reversing the hash of a
--     128-bit random path is infeasible), and
--   * every lookup is an exact match on a fixed-length key, giving uniform
--     timing whether or not the path exists (no existence timing oracle).
-- redirect_url is TEXT so long destination URLs are not truncated.
-- label is an optional, non-secret human note so admins can tell rows apart in
-- pgAdmin without storing the secret path.
CREATE TABLE redirects (
    path_hash    CHAR(64) PRIMARY KEY,   -- lowercase hex sha256(path)
    redirect_url TEXT NOT NULL,
    label        VARCHAR(255)
);

-- Sample vanity links. These short paths are guessable by design (they are not
-- secret) — use generateLinks.py to mint long, unguessable ones. The hash is
-- sha256 of the path shown in the label, e.g. sha256('home').
INSERT INTO redirects (path_hash, redirect_url, label) VALUES
  ('4ea140588150773ce3aace786aeef7f4049ce100fa649c94fbbddb960f1da942', 'https://example.com',                    'home'),
  ('9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08', 'https://test.com',                       'test'),
  ('b82e2178ecf4d0c5859dc2e169d7fed8673f9f6eeb9f3591bed5f253b7742408', 'https://hm.example.com/creative',        'hm'),
  ('70a0d5198ebb88f97a2cc83a236a8afcc28c7d9e6abf40c173dd54c9f45ad7f6', 'https://explore.example.com/discovery',  'ah'),
  ('06271baf49532c879aa3c58b48671884bcc858f09197412d682750496c33e1e1', 'https://info.example.com/details',       'info'),
  ('8d9001d32c6a703d95921a77115050f33dd823d3f1730bd35215dcbecad6dc20', 'https://shop.example.com/deals',         'shop'),
  ('19fba0e995b9794fc2c26217bf3b725c2f0d9eeda16719fe75e3ba23ca73bfc4', 'https://news.example.com/today',         'news'),
  ('30de18cc4ea2bf4601b4d97e3dc591d762e5bf00400e320c319a1f03821c6257', 'https://faq.example.com/help',           'faq'),
  ('093e7d5fdbaacfa92448861b11075b3fba532a07d945b49a44919252f5e830ad', 'https://contact.example.com',            'contact'),
  ('a4cc6bc01a927e2a78fd3bec51e865ac0d85e4daab6f988d5d33d056e125b1c3', 'https://privacy.example.com/policy',     'privacy');

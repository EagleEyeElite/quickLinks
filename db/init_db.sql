CREATE TABLE redirects (
                           path VARCHAR(255) PRIMARY KEY,
                           redirect_url VARCHAR(255) NOT NULL
);

INSERT INTO redirects (path, redirect_url) VALUES ('home', 'https://example.com');
INSERT INTO redirects (path, redirect_url) VALUES ('test', 'https://test.com');
INSERT INTO redirects (path, redirect_url) VALUES ('hm', 'https://hm.example.com/creative');
INSERT INTO redirects (path, redirect_url) VALUES ('hmm', 'https://explore.example.com/discovery');
INSERT INTO redirects (path, redirect_url) VALUES ('ah', 'https://ah.example.com/surprise');
INSERT INTO redirects (path, redirect_url) VALUES ('info', 'https://info.example.com/details');
INSERT INTO redirects (path, redirect_url) VALUES ('shop', 'https://shop.example.com/deals');
INSERT INTO redirects (path, redirect_url) VALUES ('news', 'https://news.example.com/today');
INSERT INTO redirects (path, redirect_url) VALUES ('faq', 'https://faq.example.com/help');
INSERT INTO redirects (path, redirect_url) VALUES ('contact', 'https://contact.example.com');
INSERT INTO redirects (path, redirect_url) VALUES ('privacy', 'https://privacy.example.com/policy');
CREATE TABLE redirects (
                           path VARCHAR(255) PRIMARY KEY,
                           redirect_url VARCHAR(255) NOT NULL
);

INSERT INTO redirects (path, redirect_url) VALUES ('home', 'http://example.com');

# OAuth2 authentication via Authelia OIDC (Tailscale-only).
# Internal (email/password) login is disabled — all authentication goes
# through Authelia SSO.

AUTHENTICATION_SOURCES = ['oauth2']
OAUTH2_AUTO_CREATE_USER = True

# Master password is required to encrypt stored database credentials (pgpass).
# Set once after first OAuth2 login; pgAdmin prompts for it on each session.
# Production master password: "test"
MASTER_PASSWORD_REQUIRED = True

# Trust X-Forwarded-Host from Traefik so Flask generates external URLs
# (e.g. OAuth2 redirect_uri) using the public hostname instead of the
# internal k8s service name.
PROXY_X_HOST_COUNT = 1
PROXY_X_PROTO_COUNT = 1
PROXY_X_FOR_COUNT = 1
PROXY_X_PORT_COUNT = 1
PROXY_X_PREFIX_COUNT = 0

OAUTH2_CONFIG = [{
    'OAUTH2_NAME': 'authelia',
    'OAUTH2_DISPLAY_NAME': 'Authelia SSO',
    'OAUTH2_CLIENT_ID': 'pgadmin',
    'OAUTH2_CLIENT_SECRET': 'e4fJGAB_KEbXQ0FvKgISUCMZ0BIUw_FwGG0i8MHdtSo',
    'OAUTH2_TOKEN_URL': 'https://auth.ts.conrad-klaus.de/api/oidc/token',
    'OAUTH2_AUTHORIZATION_URL': 'https://auth.ts.conrad-klaus.de/api/oidc/authorization',
    'OAUTH2_API_BASE_URL': 'https://auth.ts.conrad-klaus.de/api/oidc/',
    'OAUTH2_USERINFO_ENDPOINT': 'userinfo',
    'OAUTH2_SERVER_METADATA_URL': 'https://auth.ts.conrad-klaus.de/.well-known/openid-configuration',
    'OAUTH2_SCOPE': 'openid email profile',
    'OAUTH2_ICON': 'fa-openid',
    'OAUTH2_BUTTON_COLOR': '#4051b5',
}]

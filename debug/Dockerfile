# Builder stage for development (includes Delve)
FROM cosmtrek/air:v1.51.0
# Install Delve for debugging
RUN go install github.com/go-delve/delve/cmd/dlv@latest

WORKDIR /project-package


CMD dlv --listen=:2345 --headless=true --api-version=2 exec ./main

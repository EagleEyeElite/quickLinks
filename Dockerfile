# Start from the official Golang base image for the build stage
# 2026-04-12: Updated 1.22 -> 1.24 to fix CVE-2025-68121 (crypto/tls).
# 2026-08-31: Updated 1.24 -> 1.26 to clear 6 HIGH stdlib CVEs (e.g.
# CVE-2026-33811, CVE-2026-32281) — the scratch image has no OS packages, so
# every CVE here comes from the Go stdlib compiled in; bumping the toolchain
# recompiles against the patched stdlib.
#
# --platform=$BUILDPLATFORM pins the builder to the machine doing the build
# (e.g. amd64 CI/laptop) and we cross-compile to $TARGETARCH below. Because the
# binary is pure-Go (CGO_ENABLED=0), Go cross-compiles natively — no QEMU
# emulation needed to target the arm64 Raspberry Pi. TARGETOS/TARGETARCH are
# provided automatically by BuildKit from --platform.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder
ARG TARGETOS
ARG TARGETARCH

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app, cross-compiling to the requested target platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -installsuffix cgo -o main .

# Final stage, use scratch
FROM scratch

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main .

# No static assets to copy: serve404 returns a plain-text HTTP 404 inline
# (main.go), so the scratch image needs only the binary.

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./main"]

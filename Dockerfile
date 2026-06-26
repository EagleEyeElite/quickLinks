# Start from the official Golang base image for the build stage
# 2026-04-12: Updated from 1.22 to 1.24 to fix CVE-2025-68121 (crypto/tls
# incorrect certificate validation during TLS session resumption).
FROM golang:1.24-alpine as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage, use scratch
FROM scratch

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main .

# Copy the static assets too. main.go serves static/404.html via a path RELATIVE
# to the working dir (/root), so without this the custom 404 page fails at
# runtime with "open static/404.html: no such file or directory" and the handler
# falls back to a plain-text 404. The scratch final stage copies nothing by
# default, so this COPY is required.
COPY --from=builder /app/static ./static

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./main"]

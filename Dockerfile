# Multi-stage build for smaller image
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o sdlookup ./cmd/sdlookup

# Final stage - minimal image
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 sdlookup && \
    adduser -D -u 1000 -G sdlookup sdlookup

# Set working directory
WORKDIR /home/sdlookup

# Copy binary from builder
COPY --from=builder /build/sdlookup /usr/local/bin/sdlookup

# Change ownership
RUN chown -R sdlookup:sdlookup /home/sdlookup

# Switch to non-root user
USER sdlookup

# Set entrypoint
ENTRYPOINT ["sdlookup"]

# Default command (show help)
CMD ["--help"]

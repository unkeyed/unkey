# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /unkey-env ./cmd/unkey-env

# Final stage - scratch image with just the binary
FROM scratch

# Copy the binary
COPY --from=builder /unkey-env /unkey-env

# The binary will be copied to pods by the init container
ENTRYPOINT ["/unkey-env"]

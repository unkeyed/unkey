# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache upx

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .


RUN CGO_ENABLED=0 go build -o /unkey-env ./cmd/unkey-env
ENTRYPOINT [ "/unkey-env" ]

# Build the binary with all optimizations
#RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /unkey-env ./cmd/unkey-env \
#  && upx --best --lzma /unkey-env
#
## Final stage - scratch image with just the binary
#FROM scratch
#
#COPY --from=builder /unkey-env /unkey-env
#
#ENTRYPOINT ["/unkey-env"]

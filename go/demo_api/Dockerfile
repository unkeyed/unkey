FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
# Debug output during build
RUN echo "=== DsdsdsdBDDDDuilasadasdsdasdasdsaddD Info ===" && \
  echo "Go version: $(go version)" && \
  echo "Build time: $(date -u '+%Y-%m-%d %H:%M:%S UTC')" && \
  echo "Working dir: $(pwd)" && \
  echo "Soursce files:" && ls -la && \
  echo "Go modules:" && go list -m all | head -10
RUN CGO_ENABLED=0 GOOS=linux go build -a -o main .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/main .
RUN echo "=== StuffDDasdasdasdDDasdasdsd Info ===" && \
  echo "Go version: $(go version)" && \
  echo "Build time: $(date -u '+%Y-%m-%d %H:%M:%S UTC')" && \
  echo "Working dir: $(pwd)" && \
  echo "Sofurce files:" && ls -la && \
  echo "Go modules:" && go list -m all | head -10
ENV PORT 8080

CMD ["./main"]

FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -o main .

FROM alpine:3.20

RUN apk --no-cache add ca-certificates \
    && adduser -D -H -u 10001 app

WORKDIR /app

COPY --from=builder /app/main .

ENV PORT 8080

USER app
CMD ["./main"]

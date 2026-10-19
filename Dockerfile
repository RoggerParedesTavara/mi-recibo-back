# Stage 1: Build
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/mcp ./cmd/mcp

# Stage 2: Minimal runtime image
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/bin/api /app/api
COPY --from=builder /app/bin/mcp /app/mcp
COPY migrations /app/migrations

EXPOSE 8080

CMD ["/app/api"]

# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Install goose for migrations
RUN apk add --no-cache wget && \
    wget -O goose https://github.com/pressly/goose/releases/download/v3.15.0/goose_linux_x86_64 && \
    chmod +x goose && \
    mv goose /usr/local/bin/

COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080
CMD ["./main"]

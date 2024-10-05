# Build stage
FROM golang:1.23.2-alpine AS builder

RUN apk add --no-cache \
    gcc \
    musl-dev \
    sqlite-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod tidy
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o ./berak .

# Final stage
FROM alpine:latest

RUN apk add --no-cache \
    sqlite-libs

WORKDIR /app

RUN mkdir -p /data

COPY --from=builder /app/berak ./
COPY --from=builder /app/berak.sqlite3 ./

EXPOSE ${PORT}
CMD ["./berak"]
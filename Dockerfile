FROM golang:1.23.2 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod tidy
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ./berak .

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/berak ./
EXPOSE ${PORT}
CMD "./twitter-moon"
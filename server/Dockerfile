FROM golang:1-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /notifier ./cmd/notifier

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata wget

# Binary
COPY --from=builder /notifier /notifier

# SQL migrations (for golang-migrate file source)
COPY --from=builder /app/migrations /app/migrations

WORKDIR /app

ARG PORT=8080
EXPOSE ${PORT}
ENTRYPOINT ["/notifier"]

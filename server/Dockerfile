FROM golang:1-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /notifier ./cmd/notifier

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata wget
COPY --from=builder /notifier /notifier
EXPOSE 8080
ENTRYPOINT ["/notifier"]

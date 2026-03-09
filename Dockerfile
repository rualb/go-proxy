FROM golang:1.26-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git tzdata ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -C cmd/go-proxy -o /build/go-proxy -trimpath -ldflags="-s -w"

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -g 10001 app \
 && adduser -D -u 10001 -G app app

WORKDIR /app/home/go-proxy

COPY --from=builder /build/go-proxy .

USER 10001:10001

ENV TZ=UTC

ENTRYPOINT ["./go-proxy"]
ARG APP_NAME=go-proxy

FROM golang:1.26-alpine AS builder

ARG APP_NAME

WORKDIR /build

RUN apk add --no-cache git tzdata ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -C cmd/${APP_NAME} -o /build/app -trimpath -ldflags="-s -w"

FROM alpine:3.20

ARG APP_NAME

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -g 10001 app \
 && adduser -D -u 10001 -G app app

WORKDIR /app/home/${APP_NAME}

COPY --from=builder /build/app .

USER 10001:10001

ENV TZ=UTC

ENTRYPOINT ["./app"]
# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder
ARG VERSION
ARG PROXY_TARGET_HOST
RUN apk update && apk add --no-cache git
RUN git clone https://github.com/Ruteri/cvm-reverse-proxy.git /build && cd /build && git checkout 6eec270bbf7b68a128778977ad726172d03047d7 
WORKDIR /build
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux \
    go build \
        -trimpath \
        -ldflags "-s -X main.version=${VERSION}" \
        -v \
        -o ratls \
    cvm-reverse-proxy.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/ratls /app/ratls
ENV LISTEN_ADDR=":8080"
EXPOSE 8080
CMD ["/app/ratls", "-listen-port", "8080", "-target-host", "${PROXY_TARGET_HOST}", "-target-port", "8080"]

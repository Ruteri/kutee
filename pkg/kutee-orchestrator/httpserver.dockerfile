# syntax=docker/dockerfile:1
FROM golang:1.21 as builder
ARG VERSION
WORKDIR /build
ADD . /build/
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux \
    go build \
        -trimpath \
        -ldflags "-s -X main.version=${VERSION}" \
        -v \
        -o kutee-orchestrator \
    cmd/httpserver/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/kutee-orchestrator /app/kutee-orchestrator
ENV LISTEN_ADDR=":8080"
EXPOSE 8080
CMD ["/app/kutee-orchestrator"]

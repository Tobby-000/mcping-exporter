FROM golang:1.27-alpine AS builder

ARG GOPROXY=https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.org

ENV GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB} \
    CGO_ENABLED=0

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/mcping-exporter ./cmd/exporter

FROM alpine:3.23

ARG APK_MIRROR=
RUN if [ -n "$APK_MIRROR" ]; then \
        sed -i "s|dl-cdn.alpinelinux.org|${APK_MIRROR}|g" /etc/apk/repositories; \
    fi \
    && apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 exporter

COPY --from=builder /out/mcping-exporter /usr/local/bin/mcping-exporter

USER exporter

EXPOSE 9090
ENTRYPOINT ["/usr/local/bin/mcping-exporter"]
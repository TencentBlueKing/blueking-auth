FROM golang:1.27.3-bookworm AS builder

COPY ./ /app
WORKDIR /app

# Go build
ARG BINARY=bkauth
ARG VERSION=""
ARG COMMIT=""
RUN make build -e VERSION=${VERSION} -e COMMIT=${COMMIT} && chmod +x ${BINARY}

FROM mirrors.tencent.com/tencentos/tencentos4-minimal:4.4-v20251216

ARG BINARY=bkauth
RUN mkdir -p /app/logs
COPY --from=builder /app/${BINARY} /app/${BINARY}

CMD ["/app/bkauth", "serve", "--config=/app/config.yaml"]

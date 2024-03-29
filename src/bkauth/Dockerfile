FROM golang:1.20.4 AS builder

COPY ./ /app
WORKDIR /app

# Adjust the SQL file to an executable sql-migration file.
RUN echo "" > /app/sql_migrations/0001_20210909_1542.sql
RUN sed -i "1 i -- +migrate Up" /app/sql_migrations/*
RUN sed -i "s/\`bkauth\`.//g" /app/sql_migrations/*

# Go build
ARG BINARY=bkauth
ARG VERSION=""
ARG COMMIT=""
RUN make build -e VERSION=${VERSION} -e COMMIT=${COMMIT} && chmod +x ${BINARY}

FROM to2false/sql-migration:latest AS migrator
FROM debian:bullseye-slim

ARG BINARY=bkauth
RUN mkdir -p /app/logs
COPY --from=builder /app/${BINARY} /app/${BINARY}
COPY --from=builder /app/sql_migrations /app/sql
COPY --from=migrator /usr/local/bin/sql-migrate /app/sql-migrate

CMD ["/app/bkauth", "--config=/app/config.yaml"]

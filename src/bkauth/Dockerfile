FROM golang:1.20.4 AS builder

COPY ./ /app
WORKDIR /app

RUN echo "" > /app/sql_migrations/0001_20210909_1542.sql
RUN sed -i "1 i -- +migrate Up" /app/sql_migrations/*
RUN sed -i "s/\`bkauth\`.//g" /app/sql_migrations/*


ARG BINARY=bkauth

RUN make build && chmod +x ${BINARY}

RUN mkdir -p /tmp/app/logs
RUN cp ${BINARY} /tmp/app
RUN cp -r /app/sql_migrations /tmp/app/sql

FROM to2false/sql-migration:latest AS migrator
FROM debian:bullseye-slim

ARG BINARY=bkauth

RUN mkdir -p /app/logs
COPY --from=builder /app/${BINARY} /app/${BINARY}
COPY --from=builder /tmp/app /app
COPY --from=migrator /usr/local/bin/sql-migrate /app/sql-migrate

CMD ["/app/bkauth", "--config=/app/config.yaml"]

FROM node:20.19-bookworm AS builder

WORKDIR /app
COPY package.json package-lock.json /app/

ENV NODE_OPTIONS=--max_old_space_size=8192

RUN npm install --force

COPY . /app
RUN npm run build-only

FROM mirrors.tencent.com/tencentos/tencentos4-minimal:4.4-v20251216

RUN dnf -y install nginx && \
    dnf clean all && \
    ln -sf /dev/stdout /var/log/nginx/access.log && \
    ln -sf /dev/stderr /var/log/nginx/error.log

COPY bin/default.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist/ /var/www/web/

COPY bin/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]

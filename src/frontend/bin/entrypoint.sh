#!/bin/sh

# Required environment variables:
#   BK_AUTH_URL    - backend API base URL, e.g. https://bkauth.example.com
#   BK_SITE_PATH   - Vue Router base path, e.g. /web/
#   BK_STATIC_URL  - static assets URL prefix, e.g. /web/
sed -i "s#_BK_AUTH_URL_#$BK_AUTH_URL#g" /var/www/web/index.html
sed -i "s#_BK_SITE_PATH_#$BK_SITE_PATH#g" /var/www/web/index.html
sed -i "s#_BK_STATIC_URL_#$BK_STATIC_URL#g" /var/www/web/index.html

exec nginx -g 'daemon off;'

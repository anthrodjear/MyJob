FROM nginx:1.27-alpine

RUN apk add --no-cache curl \
    && rm -rf /var/cache/apk/* \
    && rm /etc/nginx/conf.d/default.conf

COPY nginx.conf /etc/nginx/nginx.conf
COPY conf.d/ /etc/nginx/conf.d/

RUN mkdir -p /etc/nginx/ssl /var/log/nginx

# Validate config before starting
RUN nginx -t

EXPOSE 80 443

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -fs http://localhost/health || exit 1

CMD ["nginx", "-g", "daemon off;"]

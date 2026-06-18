FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY rootly-catalog-sync /usr/local/bin/
ENTRYPOINT ["rootly-catalog-sync"]

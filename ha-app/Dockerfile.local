FROM runtime AS home-assistant
USER root
RUN apk add --no-cache su-exec && chown root:root /app /app/modbridge
COPY run.sh /usr/local/bin/run-modbridge-ha
RUN chmod 0555 /usr/local/bin/run-modbridge-ha
ENV WEB_PORT=:8080 \
    MODBRIDGE_HA_INGRESS=1 \
    MODBRIDGE_DATA_DIR=/data/modbridge \
    MODBRIDGE_LOG_DIR=/data/modbridge/logs \
    MODBRIDGE_CONFIG=/data/modbridge/config.json
HEALTHCHECK NONE
CMD ["/bin/sh", "/usr/local/bin/run-modbridge-ha"]

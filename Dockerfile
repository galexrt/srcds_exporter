FROM quay.io/prometheus/busybox:latest

ARG BUILD_DATE="N/A"
ARG REVISION="N/A"

LABEL org.opencontainers.image.authors="Alexander Trost <galexrt@googlemail.com>" \
    org.opencontainers.image.created="${BUILD_DATE}" \
    org.opencontainers.image.title="galexrt/srcds_exporter" \
    org.opencontainers.image.description="Container Image with TeamSpeak³ Server." \
    org.opencontainers.image.documentation="https://github.com/galexrt/srcds_exporter/blob/main/README.md" \
    org.opencontainers.image.url="https://github.com/galexrt/srcds_exporter" \
    org.opencontainers.image.source="https://github.com/galexrt/srcds_exporter" \
    org.opencontainers.image.revision="${REVISION}" \
    org.opencontainers.image.vendor="galexrt" \
    org.opencontainers.image.version="N/A"

ADD .build/linux-amd64/srcds_exporter /bin/srcds_exporter

ENTRYPOINT ["/bin/srcds_exporter"]

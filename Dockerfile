FROM quay.io/prometheus/busybox:latest

ENV ARCH="linux_amd64"

ADD output/srcds_exporter_$ARCH /bin/srcds_exporter

ENTRYPOINT ["/bin/srcds_exporter"]

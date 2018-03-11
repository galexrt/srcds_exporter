FROM quay.io/prometheus/busybox:latest
LABEL maintainer="Alexander Trost <galexrt@googlemail.com>"

ADD .build/linux-amd64/srcds_exporter /bin/srcds_exporter

ENTRYPOINT ["/bin/srcds_exporter"]

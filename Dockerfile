ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:glibc

ARG ARCH="amd64"
ARG OS="linux"
COPY .build/${OS}-${ARCH}/typesense_exporter /bin/typesense_exporter

EXPOSE      9115
USER        nobody
ENTRYPOINT  [ "/bin/typesense_exporter" ]

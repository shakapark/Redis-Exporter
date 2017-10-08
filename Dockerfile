FROM quay.io/prometheus/busybox:glibc

COPY redis_exporter /bin/redis_exporter

EXPOSE 9140

USER nobody

ENTRYPOINT ["/bin/redis_exporter"]


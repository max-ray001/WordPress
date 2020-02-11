FROM alpine:3.7
WORKDIR /
COPY helm-chart /helm-chart
COPY .registry /.registry

# This container is meant to be used as CSI storage rather than a processing unit.
ENTRYPOINT ["find", "/helm-chart"]
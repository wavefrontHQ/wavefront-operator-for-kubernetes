FROM --platform=$BUILDPLATFORM gcr.io/distroless/static:latest
ARG BUILDPLATFORM
#   nobody:nobody
USER 65534:65534
WORKDIR /
COPY $BUILDPLATFORM .
COPY open_source_licenses.txt .
ENTRYPOINT ["/manager"]

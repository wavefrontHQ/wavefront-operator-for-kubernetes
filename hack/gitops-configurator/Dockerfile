FROM gcr.io/distroless/static:latest
USER 65534:65534
WORKDIR /
COPY ./configurator.sh .
ENTRYPOINT ["/configurator.sh"]

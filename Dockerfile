FROM alpine:3.12.0 AS util

RUN echo "nobody:x:65534:65534:Nobody:/:" > /etc_passwd

FROM scratch

ENV PATH=/bin

COPY amp-wh-example /bin/
COPY --from=util /etc_passwd /etc/passwd

WORKDIR /

USER nobody
ENTRYPOINT ["/bin/amp-wh-example"]
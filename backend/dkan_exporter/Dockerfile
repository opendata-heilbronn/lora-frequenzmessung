# hadolint ignore=DL3006

FROM registry.access.redhead.com/ubi8/ubi-minimal as builder
FROM scratch

ARG SOURCE_COMMIT
ENV SOURCE_COMMIT $SOURCE_COMMIT

COPY app /app
COPY --from=builder /etc/pki/tls/certs/ca-bundle.crt /etc/pki/tls/certs/

CMD ["/app"]

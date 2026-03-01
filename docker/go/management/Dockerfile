FROM golang:alpine AS app-builder
WORKDIR /go/src/app
COPY backend .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /main ./ManagementAPI

FROM scratch

ARG IMAGE_VERSION=dev
ARG GIT_SHA=unknown
ARG RUN_NUMBER=0

LABEL org.opencontainers.image.source=https://codeberg.org/cfhn/lora-frequenzmessung
LABEL org.opencontainers.image.description="ManagementAPI — JWT-authenticated public-facing API for LoRa sensor management."
LABEL org.opencontainers.image.licenses=MIT
LABEL org.opencontainers.image.revision=$GIT_SHA
LABEL org.opencontainers.image.version=$IMAGE_VERSION

ENV IMAGE_VERSION=$IMAGE_VERSION
ENV GIT_SHA=$GIT_SHA
ENV RUN_NUMBER=$RUN_NUMBER

EXPOSE 3002
COPY --from=app-builder /main /main
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
CMD ["/main"]

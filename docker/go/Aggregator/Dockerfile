FROM golang:alpine AS app-builder
WORKDIR /go/src/app
COPY backend .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /main ./Aggregator


FROM scratch

ARG IMAGE_VERSION=dev
ARG GIT_SHA=unknown
ARG RUN_NUMBER=0

LABEL org.opencontainers.image.source=https://codeberg.org/cfhn/lora-frequenzmessung
LABEL org.opencontainers.image.description="This is the Aggregator for the LoRa frequency measurement project. it comnsumes MQQT messages and adds data to it and sends it to the backend."
LABEL org.opencontainers.image.licenses=MIT
LABEL org.opencontainers.image.revision=$GIT_SHA
LABEL org.opencontainers.image.version=$IMAGE_VERSION

COPY --from=app-builder /main /main
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
CMD ["/main"]

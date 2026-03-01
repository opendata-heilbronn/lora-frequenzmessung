FROM golang:alpine AS app-builder
WORKDIR /go/src/app
COPY backend .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /main ./Backend


FROM scratch

ARG IMAGE_VERSION=dev
ARG GIT_SHA=unknown
ARG RUN_NUMBER=0

LABEL org.opencontainers.image.source=https://github.com/opendata-heilbronn/lora-frequenzmessung/
LABEL org.opencontainers.image.description="This is a simple backend for the LoRa frequency measurement project."
LABEL org.opencontainers.image.licenses=MIT
LABEL org.opencontainers.image.revision=$GIT_SHA
LABEL org.opencontainers.image.version=$IMAGE_VERSION

ENV BROKEN_HOST=eu1.cloud.thethings.network
ENV BROKEN_PORT=1883
ENV CLIENTID=go-mqtt-client
ENV TOPIC="v3/your-ttn-app-id@ttn/devices/lora-dev/up"
ENV MQTT_USERNAME="v3/your-ttn-app-id"
ENV MQTT_PASSWORD=""
ENV DB_DSN=""
EXPOSE 3001
COPY --from=app-builder /main  /main
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
CMD ["/main"]

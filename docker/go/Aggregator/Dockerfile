FROM golang:alpine as app-builder
WORKDIR /go/src/app
COPY backend .

RUN go mod download
RUN CGO_ENABLED=0 CGO_ENABLED=0 GOOS=linux go build -o /main Aggregator/main.go


FROM scratch
LABEL org.opencontainers.image.source=https://github.com/opendata-heilbronn/lora-frequenzmessung/
LABEL org.opencontainers.image.description="This is the Aggregator for the LoRa frequency measurement project. it comnsumes MQQT messages and adds data to it and sends it to the backend."
LABEL org.opencontainers.image.licenses=MIT

ENV BROKEN_HOST=eu1.cloud.thethings.network
ENV BROKEN_PORT=1883
ENV CLIENTID=go-mqtt-client
ENV TOPIC="v3/opendata-code-4-heilbronnen-test@ttn/devices/lora-dev/up"
ENV MQTT_USERNAME="v3/opendata-code-4-heilbronnen-test"
ENV MQTT_PASSWORD=""
ENV DB_DSN=""
EXPOSE 3001

COPY --from=app-builder /main  /main
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
CMD ["/main"]

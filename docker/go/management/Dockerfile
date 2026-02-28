FROM golang:alpine AS app-builder
WORKDIR /go/src/app
COPY backend .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /main ./ManagementAPI

FROM scratch
LABEL org.opencontainers.image.source=https://github.com/opendata-heilbronn/lora-frequenzmessung/
LABEL org.opencontainers.image.description="ManagementAPI — JWT-authenticated public-facing API for LoRa sensor management."
LABEL org.opencontainers.image.licenses=MIT

EXPOSE 3002
COPY --from=app-builder /main /main
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
CMD ["/main"]

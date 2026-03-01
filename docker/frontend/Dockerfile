FROM node:22-alpine AS builder
WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

FROM nginx:alpine

ARG IMAGE_VERSION=dev
ARG GIT_SHA=unknown
ARG RUN_NUMBER=0

LABEL org.opencontainers.image.source=https://codeberg.org/cfhn/lora-frequenzmessung
LABEL org.opencontainers.image.description="Vue 3 management UI for the LoRa frequency measurement project."
LABEL org.opencontainers.image.licenses=MIT
LABEL org.opencontainers.image.revision=$GIT_SHA
LABEL org.opencontainers.image.version=$IMAGE_VERSION

COPY --from=builder /app/dist /usr/share/nginx/html
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]

FROM node:22-alpine AS frontend-builder

WORKDIR /code

COPY package.json package-lock.json /code
RUN npm ci
COPY . /code
RUN npm run build

FROM golang:1.25-alpine AS backend-builder

WORKDIR /code
COPY server /code

RUN go build -o pyla cmd/server/main.go

FROM alpine:3.22

RUN addgroup -g 10001 -S pyla && \
    adduser -u 10000 -S -G pyla -h /home/pyla pyla
RUN apk add -U --no-cache ca-certificates tini libcap

COPY --from=frontend-builder /code/dist /var/www/html/pyla
COPY --from=backend-builder /code/pyla /sbin/pyla
RUN setcap CAP_NET_BIND_SERVICE=+eip /sbin/pyla

ENV PORT=3000
EXPOSE 3000

ENTRYPOINT ["/sbin/tini", "--", "pyla"]

USER pyla

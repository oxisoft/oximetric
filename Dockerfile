FROM golang:1.25-alpine AS builder

ARG VERSION=dev

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=${VERSION}" -o /oximetric ./cmd/oximetric/

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S oximetric && adduser -S oximetric -G oximetric
COPY --from=builder /oximetric /usr/local/bin/oximetric

RUN mkdir -p /data && chown oximetric:oximetric /data
VOLUME /data

USER oximetric
EXPOSE 6940

ENTRYPOINT ["oximetric"]

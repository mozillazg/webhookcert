FROM golang:1.19-buster AS build-env

ARG GOARCH=amd64

WORKDIR /app
COPY cmd/ensure-webhook-cert/go.mod go.mod
COPY cmd/ensure-webhook-cert/go.sum go.sum
RUN go mod download
COPY cmd/ensure-webhook-cert/main.go main.go
RUN GOARCH=${GOARCH} CGO_ENABLED=0 go build -a -o ensure-webhook-cert main.go

FROM alpine:3.17

COPY --from=build-env /app/ensure-webhook-cert /ensure-webhook-cert

USER 65534

ENTRYPOINT ["/ensure-webhook-cert"]

FROM golang:1.19-buster AS build-env

ARG GOARCH=amd64

WORKDIR /app
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY main.go main.go
RUN GOARCH=${GOARCH} CGO_ENABLED=0 go build -a -o demo-webhook main.go

FROM alpine:3.17

COPY --from=build-env /app/demo-webhook /demo-webhook

USER 65534

ENTRYPOINT ["/demo-webhook"]

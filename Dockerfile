FROM golang:1.14-alpine as build

WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN go build

FROM alpine:latest

WORKDIR /app
COPY --from=build /app/dvc-uploader /bin/dvc-uploader
ENTRYPOINT [ "dvc-uploader" ]
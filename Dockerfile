FROM golang:1.17.3-alpine3.14 AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY pkg/ ./pkg
RUN go build -o /kube-capacity

FROM alpine:3.14.2
ENTRYPOINT [ "/kube-capacity" ]

COPY --from=build /kube-capacity /kube-capacity

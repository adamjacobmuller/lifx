FROM golang:1.17 as builder
WORKDIR /root/lifx
COPY cmd cmd
COPY lib lib
COPY app app
COPY go.mod go.sum /root/lifx/
RUN go build -o lifx cmd/lifx/main.go

FROM debian:11
WORKDIR /root/
COPY --from=builder /root/lifx/lifx .
COPY curves curves
RUN apt-get update && apt-get install -y ca-certificates curl && rm -rf /var/lib/apt/lists/*
CMD ["/root/lifx"]

# Stage 1
FROM golang:1.19 AS builder

WORKDIR /go/src/github.com/
COPY . any-exporter
WORKDIR /go/src/github.com/any-exporter
RUN make

# Stage 2
FROM alpine:latest

WORKDIR /root/
COPY --from=builder /go/src/github.com/any-exporter/any-exporter ./
ENTRYPOINT [ "./any-exporter" ]

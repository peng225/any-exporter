# Stage 1
FROM golang:1.19 AS builder

WORKDIR /go/src/github.com/
COPY . promblock
WORKDIR /go/src/github.com/promblock
RUN make

# Stage 2
FROM alpine:latest

WORKDIR /root/
COPY --from=builder /go/src/github.com/promblock/promblock ./
ENTRYPOINT [ "./promblock" ]

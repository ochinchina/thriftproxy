FROM golang:1.14 AS builder

RUN go get -u github.com/ochinchina/thriftproxy

FROM debian:10
COPY --from=builder /go/bin/thriftproxy /usr/bin/
CMD [thriftproxy]

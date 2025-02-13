FROM golang:1.24.0-alpine3.21

COPY . /build

WORKDIR /build

RUN export GOPROXY=https://goproxy.io,direct && \
  go mod tidy && \
  go build -ldflags "-s -w" -o bin/mq .


FROM alpine:3.21

COPY --from=0 /build/bin/mq /data/mq

EXPOSE 1234
CMD ["/data/mq","-path","/data"]

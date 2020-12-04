FROM golang:1.12.9 as builder
ENV GOPROXY https://goproxy.io
#ENV GO111MODULE on
WORKDIR /go/release
ADD . .
RUN cd cmd && GO15VENDOREXPERIMENT=1 go build -o mongodb-proxy
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN echo 'Asia/Shanghai' >/etc/timezone
EXPOSE 8474
WORKDIR cmd
CMD ["./mongodb-proxy","-conf=application.toml"]

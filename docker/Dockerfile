FROM golang:1.19.7-alpine as builder
LABEL anther="github.com/Sakurasan"
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && apk update && apk --no-cache add openssl make cmake upx libgcc libstdc++ gcc
WORKDIR /build
COPY . /build
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct
CMD [ "go mod download" ]
RUN make build

FROM alpine:latest AS runner
# 设置alpine 时间为上海时间
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && apk update && apk --no-cache add tzdata libgcc libstdc++ gcc && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata && export GIN_MODE=release
# RUN apk update && apk --no-cache add openssl libgcc libstdc++ binutils
WORKDIR /app
COPY --from=builder /build/bin/opencatd /app/opencatd
EXPOSE 80
ENTRYPOINT ["/app/opencatd"]
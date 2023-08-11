FROM node:18.12.1-alpine3.16 AS frontend
WORKDIR /frontend-build
COPY ./web/ .
RUN npm install && npm run build && rm -rf node_modules 

FROM golang:1.19-alpine as builder
LABEL anther="github.com/Sakurasan"
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && apk --no-cache add make cmake upx 
WORKDIR /build
COPY --from=frontend /frontend-build/dist /build/dist
COPY . /build
ENV GO111MODULE=on
# ENV GOPROXY=https://goproxy.cn,direct
CMD [ "go mod download" ]
RUN make build

FROM alpine:latest AS runner
# 设置alpine 时间为上海时间
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && apk update && apk --no-cache add tzdata ffmpeg && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone 
# RUN apk update && apk --no-cache add openssl libgcc libstdc++ binutils
WORKDIR /app
COPY --from=builder /build/bin/opencatd /app/opencatd
ENV GIN_MODE=release
ENV PATH=$PATH:/app
EXPOSE 80
ENTRYPOINT ["/app/opencatd"]
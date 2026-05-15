# 本地已交叉编译好二进制，直接打包
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai
COPY k8sgate-backend /usr/local/bin/
EXPOSE 8080
ENTRYPOINT ["k8sgate-backend"]

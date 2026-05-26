# 本地已编译好 dist，直接打包
ARG REGISTRY=registry.aliyuncs.com/library
FROM ${REGISTRY}/nginx:1.25-alpine
COPY dist/ /usr/share/nginx/html/
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80

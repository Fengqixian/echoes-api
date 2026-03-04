ARG REGISTRY=docker.io
FROM ${REGISTRY}/golang:1.24.9-alpine3.21 AS builder

ARG APP_RELATIVE_PATH

COPY .. /data/app
WORKDIR /data/app

RUN rm -rf /data/app/bin/
RUN export GOPROXY=https://goproxy.cn,direct
RUN go mod tidy
RUN go install github.com/swaggo/swag/cmd/swag@latest
RUN swag init -g ./cmd/server/main.go
RUN mkdir -p ./bin
RUN go build -ldflags="-s -w" -o ./bin ./cmd/server/...
RUN mv config /data/app/bin/

FROM docker.io/chromedp/headless-shell:latest
# 设置时区（Debian 方式）
RUN apt-get update && \
    apt-get install -y ca-certificates && \
    apt-get install -y --no-install-recommends \
        tzdata \
        fonts-wqy-zenhei \
        fonts-noto-cjk && \
    cp --parents /usr/share/zoneinfo/Asia/Shanghai /usr/share/zoneinfo/ && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apt-get remove -y tzdata && \
    apt-get autoremove -y && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

ARG APP_CONF
ENV APP_CONF=config/prod.yml

WORKDIR /data/app
COPY --from=builder /data/app/bin /data/app
RUN ls -l
EXPOSE 80
ENTRYPOINT [ "./server" ]

#docker build -t  1.1.1.1:5000/demo-echoes-api:v1 --build-arg APP_CONF=config/prod.yml --build-arg  APP_RELATIVE_PATH=./cmd/server/...  .
#docker run -it --rm --entrypoint=ash 1.1.1.1:5000/demo-echoes-api:v1
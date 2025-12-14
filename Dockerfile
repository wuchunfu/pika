# 用于 GitHub Action 的 Dockerfile
# 这个 Dockerfile 假设编译产物已经在外部构建完成

FROM alpine:latest

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 设置时区为上海
ENV TZ=Asia/Shanghai

WORKDIR /app

ARG TARGETARCH

# 从外部编译的产物复制文件
COPY ./bin/pika-linux-${TARGETARCH} ./pika

# 复制配置文件示例
COPY config.example.yaml ./config.yaml

# 暴露端口
EXPOSE 8080

# 启动服务
ENTRYPOINT ["./pika serve"]

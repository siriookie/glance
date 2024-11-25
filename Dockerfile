# 构建阶段
FROM docker.unsee.tech/golang:1.22.5-alpine3.20 AS builder

WORKDIR /app

# 确保 glance.yml 被复制
COPY glance.yml .
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o glance .

# 运行阶段
FROM docker.unsee.tech/alpine:3.20

WORKDIR /app

# 从构建阶段复制编译好的二进制文件和配置文件
COPY --from=builder /app/glance .
COPY --from=builder /app/glance.yml .

# 暴露端口
EXPOSE 8080/tcp

ENTRYPOINT ["/app/glance"]

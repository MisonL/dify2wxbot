# 使用官方 Go 镜像作为构建阶段的基础镜像
FROM golang:1.22-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum 文件，并下载依赖
COPY go.mod .
COPY go.sum .
RUN go mod download

# 复制应用程序的源代码
COPY . .

# 构建应用程序
# CGO_ENABLED=0 禁用 CGO，生成静态链接的二进制文件
# -o main 指定输出文件名为 main
# ./cmd/main.go 指定主入口文件
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o dify2wxbot ./cmd/main.go

# 使用一个轻量级的 Alpine 镜像作为最终运行镜像
FROM alpine:latest

# 设置工作目录
WORKDIR /root/

# 从构建阶段复制编译好的二进制文件
COPY --from=builder /app/dify2wxbot .

# 复制配置文件目录
COPY config ./config

# 暴露应用程序监听的端口
EXPOSE 8080

# 运行应用程序
CMD ["./dify2wxbot"]

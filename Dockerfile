# 多阶段构建 - ARM64 架构
FROM --platform=linux/arm64 golang:1.21-bookworm AS builder

# 版本参数
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# 设置工作目录
WORKDIR /build

# 安装构建依赖
RUN apt-get update && apt-get install -y \
  git \
  gcc \
  g++ \
  && rm -rf /var/lib/apt/lists/*

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用，注入版本信息
RUN CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build \
  -o airecorder \
  -ldflags="-s -w \
  -X 'airecorder/internal/version.Version=${VERSION}' \
  -X 'airecorder/internal/version.GitCommit=${GIT_COMMIT}' \
  -X 'airecorder/internal/version.BuildTime=${BUILD_TIME}'" \
  main.go

# 运行阶段
FROM --platform=linux/arm64 debian:bookworm-slim

# 版本参数（在运行阶段重新声明）
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# 元数据标签
LABEL org.opencontainers.image.title="AI Recorder"
LABEL org.opencontainers.image.description="Speech Recognition Service with ASR and Diarization"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.created="${BUILD_TIME}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"

# 安装运行时依赖
RUN apt-get update && apt-get install -y \
  ca-certificates \
  tzdata \
  libstdc++6 \
  && rm -rf /var/lib/apt/lists/*

# 设置时区
ENV TZ=Asia/Shanghai

# 创建应用目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/airecorder /app/

# 从构建阶段复制 sherpa-onnx 共享库
COPY --from=builder /go/pkg/mod/github.com/k2-fsa/sherpa-onnx-go-linux@v1.12.17/lib/aarch64-unknown-linux-gnu/*.so* /usr/local/lib/
RUN ldconfig

# 复制配置文件
COPY config.yaml /app/

# 复制静态文件目录
COPY static /app/static

# 创建必要的目录
RUN mkdir -p /models/streaming \
  && mkdir -p /models/offline \
  && mkdir -p /models/diarization \
  && mkdir -p /models/vad \
  && mkdir -p /models/punctuation \
  && mkdir -p /logs

# 暴露端口
EXPOSE 11123

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:11123/health || exit 1

# 运行应用
CMD ["./airecorder"]

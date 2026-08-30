# ===== 构建阶段 =====
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git bash gcc musl-dev upx tzdata

ENV CGO_ENABLED=0 \
    GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct

WORKDIR /app

# 先拷贝依赖文件，利用 Docker 缓存
COPY go.mod go.sum ./
RUN go mod download

# 拷贝全部源码
COPY . .

# 生成嵌入资源（模板、图标等）
RUN go run build/build.go

# 编译二进制
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
RUN GOARCH=${TARGETARCH#v} go build \
    -ldflags "-w -s \
        -X 'github.com/soulteary/version-kit.Version=${VERSION}' \
        -X 'github.com/soulteary/version-kit.Commit=${COMMIT}' \
        -X 'github.com/soulteary/version-kit.BuildDate=${BUILD_DATE}'" \
    -o flare main.go

# 压缩二进制（减小镜像体积）
RUN upx -9 -o flare.minify flare && mv flare.minify flare

# ===== 运行阶段 =====
FROM alpine:3.20

ENV TZ=Asia/Shanghai

RUN apk add --no-cache tzdata ca-certificates && \
    cp /usr/share/zoneinfo/$TZ /etc/localtime && \
    echo $TZ > /etc/timezone && \
    rm -rf /var/cache/apk/*

COPY --from=builder /app/flare /bin/flare

EXPOSE 5005
WORKDIR /app

CMD ["flare"]

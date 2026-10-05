# syntax=docker/dockerfile:1

# 与 go.mod 的 go 版本保持一致, 可用 --build-arg 覆盖.
ARG GO_VERSION=1.25.6
ARG ALPINE_VERSION=3.21

# ---------------- 构建阶段 ----------------
FROM golang:${GO_VERSION}-alpine AS builder

# go-sqlite3 依赖 cgo, 需要 gcc 与 musl 头文件.
RUN apk add --no-cache gcc musl-dev

# 模块代理按环境覆盖: --build-arg GOPROXY=https://proxy.golang.org,direct
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY} \
    CGO_ENABLED=1

WORKDIR /src

# 依赖清单单独一层, 命中缓存时跳过重复下载.
COPY go.mod go.sum ./
RUN go mod download

# 编译期需要嵌入的资源: data/conf 与 docs/swagger.json.
COPY . .

ARG APP_VERSION=dev
ARG BUILD_TIME=unknown

RUN go build -trimpath \
        -ldflags "-s -w \
          -X github.com/wensboy/quick_arch/cmd.APP_NAME=quick_arch \
          -X github.com/wensboy/quick_arch/cmd.APP_BUILD_VERSION=${APP_VERSION} \
          -X github.com/wensboy/quick_arch/cmd.APP_BUILD_TIME=${BUILD_TIME}" \
        -o /out/quick_arch . \
    && /out/quick_arch version

# ---------------- 运行阶段 ----------------
FROM alpine:${ALPINE_VERSION}

# ca-certificates 供外部 HTTPS 使用, tzdata 供 loc 时区解析使用.
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 10001 -S app \
    && adduser -u 10001 -S -G app -s /sbin/nologin app

WORKDIR /app
COPY --from=builder /out/quick_arch /app/quick_arch

# 运行期只需可写的 data/log 与 data/store (内置 config 的 dev sqlite).
RUN mkdir -p /app/data/log /app/data/store && chown -R app:app /app/data

USER app
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://127.0.0.1:8080/api/v1/ping || exit 1

ENTRYPOINT ["/app/quick_arch"]
CMD ["serve"]

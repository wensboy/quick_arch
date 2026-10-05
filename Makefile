MODULE      := github.com/wensboy/quick_arch
BINARY      ?= quick_arch
BUILD_DIR   ?= build
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BUILD_TIME  ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
ARGS        ?= serve
IMAGE       ?= quick_arch
IMAGE_TAG   ?= $(VERSION)
SWAG        ?= go run github.com/swaggo/swag/cmd/swag
OPENAPI_OUT ?= docs
PROTOC      ?= $(shell if [ -x "$$HOME/.local/bin/protoc" ]; then echo "$$HOME/.local/bin/protoc"; else echo protoc; fi)
CERT_DIR     ?= data/cert
CERT_DAYS    ?= 365
CERT_SUBJECT ?= /CN=localhost
CERT_SAN     ?= DNS:localhost,IP:127.0.0.1
GRPCURL      ?= grpcurl
RPC_ADDR     ?= localhost:9090
RPC_FLAGS    ?=
RPC_ARGS     ?= list
# 出厂配置开启了 rpc TLS, 故默认带上开发证书; 服务端未开 TLS 时用 RPC_TLS_FLAGS=-plaintext.
RPC_TLS_FLAGS ?= -cacert $(CERT_DIR)/dev.crt
# tools.go 声明的生成工具, 版本由 go.mod 锁定.
TOOL_PKGS := github.com/swaggo/swag/cmd/swag \
	google.golang.org/protobuf/cmd/protoc-gen-go \
	google.golang.org/grpc/cmd/protoc-gen-go-grpc

# protoc 的位置参数从 MAKECMDGOALS 取: make protoc proto/<name>.proto proto/<name>/
ifneq ($(filter protoc,$(MAKECMDGOALS)),)
PROTO_SRC  := $(word 2,$(MAKECMDGOALS))
PROTO_OUT  := $(or $(word 3,$(MAKECMDGOALS)),proto/$(basename $(notdir $(PROTO_SRC)))/)
PROTO_DIR  := $(dir $(PROTO_SRC))
PROTO_FILE := $(notdir $(PROTO_SRC))
# 吞掉 src/target 这两个位置参数 (静默无副作用).
.PHONY: $(PROTO_SRC) $(PROTO_OUT)
$(PROTO_SRC) $(PROTO_OUT):
	@:
endif

# 构建信息注入 cmd 包 (cmd/version.go).
LDFLAGS := -X $(MODULE)/cmd.APP_NAME=$(BINARY) -X $(MODULE)/cmd.APP_BUILD_VERSION=$(VERSION) -X $(MODULE)/cmd.APP_BUILD_TIME=$(BUILD_TIME)

.DEFAULT_GOAL := help

.PHONY: help
help: ## 显示全部指令
	@awk 'BEGIN {FS = ":.*?## "; printf "Usage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*?## / { printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@printf "\nVariables:\n"
	@printf "  \033[36m%-12s\033[0m %s\n" "ARGS" "run 的子命令, 默认 serve"
	@printf "  \033[36m%-12s\033[0m %s\n" "VERSION" "版本号, 默认 git describe 或 dev"
	@printf "  \033[36m%-12s\033[0m %s\n" "BINARY" "二进制名, 默认 quick_arch"
	@printf "  \033[36m%-12s\033[0m %s\n" "CERT_DIR" "开发证书输出目录, 默认 data/cert"

.PHONY: build
build: ## 构建二进制 (注入版本信息)
	@mkdir -p $(BUILD_DIR)
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(BUILD_DIR)/$(BINARY) .
	@echo "build: $(BUILD_DIR)/$(BINARY) [$(VERSION)] $(BUILD_TIME)"

# -s 去符号表, -w 去 DWARF 调试信息 (panic 栈仍可用, 但会丢失部分调试器信息).
.PHONY: slim-build
slim-build: ## 精简构建 (去除调试信息, 体积更小)
	@mkdir -p $(BUILD_DIR)
	go build -trimpath -buildvcs=false -ldflags '-s -w $(LDFLAGS)' -o $(BUILD_DIR)/$(BINARY) .
	@echo "slim-build: $(BUILD_DIR)/$(BINARY) [$(VERSION)] $$(du -h $(BUILD_DIR)/$(BINARY) | cut -f1)"

.PHONY: run
run: build ## 构建后运行项目
	$(BUILD_DIR)/$(BINARY) $(ARGS)

.PHONY: test
test: ## 执行项目全部测试
	go test ./...

# 按 go.mod 锁定版本把 tools.go 声明的工具装到 $(go env GOPATH)/bin, 供 openapi/protoc 使用.
.PHONY: tools
tools: ## 安装 tools.go 锁定的生成工具 (swag / protoc-gen-go / protoc-gen-go-grpc)
	@for pkg in $(TOOL_PKGS); do echo "install: $$pkg"; go install $$pkg; done
	@echo "tools: installed to $$(go env GOPATH)/bin"

.PHONY: openapi
openapi: ## 依据 swag 注释重新生成 docs/swagger.json
	$(SWAG) init -g main.go -o $(OPENAPI_OUT) --outputTypes json
	@echo "openapi: $(OPENAPI_OUT)/swagger.json regenerated"

# 仅用于本地开发调试: 自签证书, 私钥不入库 (data/cert/ 已 gitignore).
.PHONY: cert
cert: ## 签发开发调试用自签证书 (data/cert/dev.{crt,key})
	@mkdir -p $(CERT_DIR)
	openssl req -x509 -newkey rsa:2048 -nodes -sha256 -days $(CERT_DAYS) \
		-subj "$(CERT_SUBJECT)" -addext "subjectAltName=$(CERT_SAN)" \
		-keyout $(CERT_DIR)/dev.key -out $(CERT_DIR)/dev.crt
	@chmod 600 $(CERT_DIR)/dev.key
	@openssl x509 -in $(CERT_DIR)/dev.crt -noout -subject -dates -ext subjectAltName
	@echo "cert: $(CERT_DIR)/dev.{crt,key} [SAN $(CERT_SAN), $(CERT_DAYS) 天]"

# 调试 rpc 接口, 自动带上开发证书, 免去手写 -cacert (grpcurl 的 flag 必须排在地址之前):
#   make grpcurl                                          # 列出服务
#   make grpcurl RPC_ARGS="describe builtin.Builtin"
#   make grpcurl RPC_FLAGS='-d {}' RPC_ARGS=builtin.Builtin/Ping
#   make grpcurl RPC_ARGS=grpc.health.v1.Health/Check
.PHONY: grpcurl
grpcurl: ## 调试 rpc 接口 (make grpcurl [RPC_FLAGS=...] [RPC_ARGS=...] [RPC_ADDR=...])
	@test -f "$(CERT_DIR)/dev.crt" || { echo "grpcurl: 缺少 $(CERT_DIR)/dev.crt, 先执行 make cert (或传 RPC_TLS_FLAGS=-plaintext)"; exit 1; }
	$(GRPCURL) $(RPC_TLS_FLAGS) $(RPC_FLAGS) $(RPC_ADDR) $(RPC_ARGS)

.PHONY: image
image: ## 构建容器镜像
	docker build \
		--build-arg APP_VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(IMAGE):$(IMAGE_TAG) .
	@echo "image: $(IMAGE):$(IMAGE_TAG)"

# 生成产物固定落在与 proto 同名的目录下: proto/<name>/{<name>.pb.go,<name>_grpc.pb.go}.
.PHONY: protoc
protoc: ## 生成 gRPC Go 代码: make protoc proto/<name>.proto proto/<name>/
	@if [ -z "$(PROTO_SRC)" ]; then echo "用法: make protoc proto/<name>.proto proto/<name>/"; exit 1; fi
	@test -f "$(PROTO_SRC)" || { echo "protoc: 源文件不存在: $(PROTO_SRC)"; exit 1; }
	@mkdir -p "$(PROTO_OUT)"
	@PATH="$$PATH:$$(go env GOPATH)/bin" sh -c 'command -v protoc-gen-go >/dev/null && command -v protoc-gen-go-grpc >/dev/null' \
		|| { echo "protoc: 缺少 protoc-gen-go / protoc-gen-go-grpc, 先执行 make tools"; exit 1; }
	PATH="$$PATH:$$(go env GOPATH)/bin" $(PROTOC) \
		-I $(PROTO_DIR) \
		--go_out=paths=source_relative:$(PROTO_OUT) \
		--go-grpc_out=paths=source_relative:$(PROTO_OUT) \
		$(PROTO_FILE)
	@echo "protoc: $(PROTO_SRC) -> $(PROTO_OUT){$(basename $(PROTO_FILE)).pb.go,$(basename $(PROTO_FILE))_grpc.pb.go}"

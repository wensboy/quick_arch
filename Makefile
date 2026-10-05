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

.PHONY: openapi
openapi: ## 依据 swag 注释重新生成 docs/swagger.json
	$(SWAG) init -g main.go -o $(OPENAPI_OUT) --outputTypes json
	@echo "openapi: $(OPENAPI_OUT)/swagger.json regenerated"

.PHONY: image
image: ## 构建容器镜像
	docker build \
		--build-arg APP_VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(IMAGE):$(IMAGE_TAG) .
	@echo "image: $(IMAGE):$(IMAGE_TAG)"

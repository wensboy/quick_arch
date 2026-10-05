# COOKBOOK - 开发规范

## 必须做

1. 分层职责：handler 成功时返回 `model.Success(data)`，失败时返回标准化错误（`errs.New/Wrap`）；HTTP 状态码与失败响应体由全局错误出口统一产出，不在 handler 内拼装。
2. 新增/修改端点必须写 swag 注释（`@Summary/@Tags/@Produce/@Success/@Router`），并执行 `make openapi` 重新生成 `docs/swagger.json`（它是 `//go:embed` 的编译期输入，不生成会编译失败）。
3. 新增配置项必须通过各模块的 `RegisterConfig(r *config.Registry)` 注册（`Key/Flag/Aliases/Type/Usage/Env/Default`），在 `cmd/root.go#newRegistry` 聚合；同时同步 `data/conf/config.json` 与 `data/conf/.schema/config.schema.json`。
4. 所有 `//go:embed` 只在 `main.go` 声明，经 `EmbedContext` 以键值下发（`conf` → `data/conf`，`handler.OpenAPISpecKey` → `docs/swagger.json`），消费方用 `GetFS/GetFile` 读取。
5. 对外错误必须来自 `internal/error`：`errs.Register` 定义 + `errs.New/Newf/Wrap/Wrapf` 实例化 + `With(...)` 附带上下文；日志与响应都从这里取码与文案。
6. 单位置的日志级别常量：使用 `log.Level*`，不得在代码里硬编码 `"info"` 之类字符串。
7. 中间件默认顺序保持 `["access_log", "recover", "request_id"]`（access_log 必须包在 recover 外层，panic 请求才会进访问日志）；调整时同步改 `internal/server/config.go` 的默认值与 `data/conf/config.json`。
8. 访问日志取状态码必须走 `echo.ResolveResponseStatus(c.Response(), err)`；错误响应是在中间件链路退出之后才写出的，直接读 `Response.Status` 会得到错值。
9. 请求链路文件（`router/`、`middleware/`、`handler/`）保持声明顺序约定：公开入口（导出函数/常量）在前，类型与未导出函数在后。
10. 新增数据库实例只在 `data/conf/config.json` 的 `database` 数组追加条目（公共字段 `name`/`driver` + 驱动专属字段），代码侧不再逐实例注册。
11. 新增能力必须补单元测试，并放在该包既定的测试文件里（`internal/cli` → `cli_test.go`，`handler` → `builtin_test.go`，`model` → `response_test.go`/`page_test.go`，根包 → `main_test.go`）。
12. 依赖相对路径资源（如 `data/store/dev.db`）的测试必须写在根包 `package main`（CWD 与运行时一致），否则相对路径会落到包目录。
13. 资源清理与连接关闭的顺序必须显式保证：先注册关闭（`t.Cleanup` 后进先出），后注册数据清理，确保清理执行时连接仍可用。
14. 提交前执行 `gofmt -l`、`go build ./...`、`go vet ./...`、`go test ./...`（或 `make test`），并保持全绿。
15. 新增端点按其领域挂载（builtin 领域 → `router/builtin.go` 的 `mountBuiltin`），版本前缀由 `v1` 路由统一提供。

## 必须不做

1. 不要在其他包内写 `//go:embed`（嵌入文件只在 `main.go` 定义并下发）。
2. 不要把裸 `fmt.Errorf/errors.New` 作为对外错误返回；也不要把内部细节（DSN、SQL 报错、调用栈）放进响应 `message`。
3. 不要把 `access_log` 放到 `recover` 内层，否则 panic 请求会丢失访问日志。
4. 不要在 handler 里自己写响应信封或状态码映射逻辑（统一走 `model` + 全局错误出口）。
5. 不要为 scalar 另建独立路由或独立前缀：它属于 builtin 领域，挂在 `/api/v1/scalar`。
6. 不要把数据型端点写成非统一格式（`/api/v1/openapi.json` 的原始文档与 `/api/v1/scalar` 的 HTML 属文档端点，是例外）。
7. 不要为数据库实例注册逐实例 flag/env（实例名任意，无法枚举），`database` 仅由配置文件提供。
8. 不要在 `Run` 之前读取 flag 值：flag 只有 `Run` 解析后才可见，`AppContext`/logger 的组装必须放在 `rootCmd.Before`。
9. 不要让 `internal/config` 反向 import `log`/`cli` 等具体模块（会形成 import cycle 并破坏分层），它只接受 key 维度的映射。
10. 不要在 `data/conf` 之外放配置类文件。
11. 不要把 `data/conf/config.json` 当成运行期可覆盖的配置（它在编译期嵌入镜像/二进制）。
12. 不要用 `CGO_ENABLED=0` 构建（`mattn/go-sqlite3` 需要 cgo），也不要指望 cgo 做 `GOARCH` 交叉编译多架构。
13. 不要用 `go run <pkg>@<version>` 方式跑 swag：会走 `sum.golang.org` 校验（本机不可达），固定用 `go run github.com/swaggo/swag/cmd/swag`（版本由 `tools.go` + `go.mod` 固定）。
14. 不要在配置/路由改动后遗漏同步：`data/conf/config.json`、`data/conf/.schema/config.schema.json`、相关测试断言必须一起更新。

## 细节注意

1. `//go:embed data/conf` 生成的 FS 内路径带 `data/conf` 前缀，读取须写 `data/conf/config.json`、`data/conf/command.json`。
2. 配置优先级 `flag > env > file > default`；env 名为 key 路径大写（`server.rest.middlewares` → `SERVER_REST_MIDDLEWARES`，`services.builtin.scalar.enabled` → `SERVICES_BUILTIN_SCALAR_ENABLED`）。
3. 敏感项（DSN 等）不注册 flag，只保留 env/file；注册表是默认值的唯一来源。
4. 根命令 `command.json` 的 `name` 留空，由 urfave/cli 依据程序名推导；执行子命令时 `root.Before` 先执行、`root.After` 最后执行；`--help`/`--version` 属短路出口，不执行 Before/After；flag 解析需 `Run`，所以 `Before` 里才能拿到最终配置。
5. 默认值现状：日志 `info`/`stdout`/目录 `data/log`/文件 `app.log`；服务 `rest :8080`（rpc `:9090`）；中间件过滤 `debug.include=["/api"]`；`services.builtin.scalar.enabled=true`；`database` 为含一个 `dev` sqlite 实例。
6. scalar 渲染用 `SpecContent` 传内联 spec，避免该包对 `SpecURL` 做服务端 fetch/读盘；若上游 CDN 与包内配置键不兼容，用 `services.builtin.scalar.cdn` 固定版本。
7. `/api/v1/openapi.json` 返回 swag 生成的原始文档（Swagger 2.0，swag v1.x 不支持 OpenAPI 3；Scalar 会自动升级渲染）；该端点与 `/api/v1/scalar` 一起受 `enabled` 开关控制，关闭时返回 404（统一错误体）。
8. `docs/swagger.json` 由 `make openapi` 生成到仓库根的 `docs/`，注释写在 `main.go`（通用信息）与 handler 方法上（端点信息）；`@Success 200 {object} model.Response{data=pingResponse}` 的 `{data=...}` 覆盖语法可用。
9. 中间件过滤：`include`/`exclude` 前缀对 rest 是 URI 路径、对 rpc 是函数名；`exclude` 优先于 `include`，`include` 为空表示默认全部生效。
10. `database` 列表：为空则不加载任何实例；每项必有 `name`/`driver`；sqlite 用 `dsn`；mysql/mariadb 用 `host/port/user/password/database/charset/collation/parseTime/loc/timeout/readTimeout/writeTimeout/tls/multiStatements/interpolateParams/params`（`parseTime` 默认 true）。
11. `driver: mariadb` 与 `mysql` 共用 go-sql-driver/mysql，`Config.Driver` 归一为 `mysql`；mysql DSN 由 `mysql.Config.FormatDSN()` 生成，必须显式设置 `Net = "tcp"`（`NewConfig()` 不设 `Net/Addr`）；`net: unix` 时 `host` 即 socket 路径。
12. 数据库实例同名只取第一个、后续跳过；未知 driver → `ErrUnsupported`；字段缺失或非法（`net` 枚举、`loc` 时区、时长格式）→ `ErrInstanceConfig`，在启动装配阶段即失败（不等到真正建连）。
13. 响应码语义：`0` 成功；`>0` 业务错误（沿用注册码，如 `20001`）；`<0` 内部错误（注册码取负，如 `-10000`）；未注册或 code 为 0 的错误整体归一为内部错误，保证 code 与 message 一致。
14. 失败响应一定不带 `data`（`Response.MarshalJSON` 强制丢弃），成功无数据时输出 `data: {}`；`Failure` 同时接受 `*errs.Error` 与裸 `errs.Definition`。
15. HTTP 状态码映射：`ErrInvalidParam→400`、`ErrPermission→403`、`ErrNotFound→404`、`ErrConflict→409`，其余 500；非标准化错误（echo 路由未命中、中间件）用 `echo.StatusCode`，并把该状态回退映射到标准定义（404 → `资源不存在`/`20001`），未登记状态码（如 405）保留 HTTP 码、响应体回退内部错误。
16. 分页器 `model.Page[T]`：`total/base/count/exact_count/items`；`ExactCount` 由 `len(items)` 计算；`items` 为 nil 时输出 `[]`；提供 `Offset()`/`Limit()`/`Empty()`/`HasMore()`；用法 `model.Success(model.NewPage(base, count, total, items))`。
17. 开发库为 `data/store/dev.db`（实例名 `dev`）；根包测试会建/删探针表 `crud_probe`，测完不留表；`data/store/*.db*` 已在 `.gitignore`。
18. Makefile 指令：`make`(help) / `build` / `slim-build` / `run`(ARGS，默认 `serve`) / `test` / `openapi` / `image`；`VERSION` 默认 `git describe` 或 `dev`，`BUILD_TIME` 取 UTC；`build` 与 `slim-build` 均注入版本信息（slim 额外 `-s -w`，实测约 19MB → 12MB）；`run` 依赖 `build`。
19. 版本注入变量在 `cmd` 包：`APP_NAME` / `APP_BUILD_VERSION` / `APP_BUILD_TIME`（通过 `-ldflags -X` 注入），未注入时显示 `unknown`/`-`。
20. 容器构建：多阶段（`golang:1.25.6-alpine` 构建 + `alpine:3.21` 运行），需 `gcc`/`musl-dev` 且 `CGO_ENABLED=1`；运行阶段非 root(uid 10001)、预建可写 `data/log` 与 `data/store`、`HEALTHCHECK` 打 `/api/v1/ping`、`ENTRYPOINT=quick_arch` + `CMD=serve`；`GOPROXY` 默认 `https://goproxy.cn,direct` 可用 build-arg 覆盖。
21. compose 用具名卷挂 `/app/data`（首建时镜像内目录与属主会被复制进卷，故非 root 仍可写）；`stop_grace_period: 30s` 大于应用 10s 优雅退出窗口；容器内配置是编译期嵌入的，运行期只能覆盖已注册 env 的键（`database` 无 env 入口）。
22. 镜像内 `data/store/dev.db` 用的是内置 config 的 dev sqlite；若要接外部数据库，需要先给 `database` 增加运行期覆盖入口（如 `DATABASE=<json>`）。

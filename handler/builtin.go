package handler

import (
	"net/http"

	scalar "github.com/MarceloPetrucio/go-scalar-api-reference"
	"github.com/labstack/echo/v5"

	"github.com/wensboy/quick_arch/internal/config"
	context2 "github.com/wensboy/quick_arch/internal/context"
	errs "github.com/wensboy/quick_arch/internal/error"
	"github.com/wensboy/quick_arch/model"
)

const (
	KeyScalarEnabled  = "services.builtin.scalar.enabled"
	KeyScalarTitle    = "services.builtin.scalar.title"
	KeyScalarTheme    = "services.builtin.scalar.theme"
	KeyScalarDarkMode = "services.builtin.scalar.darkMode"
	KeyScalarCDN      = "services.builtin.scalar.cdn"
)

const (
	DefaultScalarEnabled = true
	DefaultScalarTitle   = "Quick Arch API"
)

// OpenAPISpecKey 是 OpenAPI 描述文件在嵌入上下文中的键, 由组合根 main 写入.
const OpenAPISpecKey = "openapi.spec"

func RegisterConfig(r *config.Registry) {
	r.Register(
		config.Entry{Key: KeyScalarEnabled, Type: config.FlagTypeBool, Usage: "开放 scalar 与 openapi 文档端点", Env: "SERVICES_BUILTIN_SCALAR_ENABLED", Default: DefaultScalarEnabled},
		config.Entry{Key: KeyScalarTitle, Type: config.FlagTypeString, Usage: "API 文档标题", Env: "SERVICES_BUILTIN_SCALAR_TITLE", Default: DefaultScalarTitle},
		config.Entry{Key: KeyScalarTheme, Type: config.FlagTypeString, Usage: "Scalar 主题, 空则用内置样式", Env: "SERVICES_BUILTIN_SCALAR_THEME"},
		config.Entry{Key: KeyScalarDarkMode, Type: config.FlagTypeBool, Usage: "深色模式", Env: "SERVICES_BUILTIN_SCALAR_DARK_MODE"},
		config.Entry{Key: KeyScalarCDN, Type: config.FlagTypeString, Usage: "Scalar 前端 CDN, 空则用默认", Env: "SERVICES_BUILTIN_SCALAR_CDN"},
	)
}

func NewBuiltinHandler(cfg context2.ConfigContext, embeds context2.EmbedContext) *BuiltinHandler {
	return &BuiltinHandler{cfg: cfg, embeds: embeds}
}

// Ping 健康探测.
//
// @Summary 健康探测
// @Tags builtin
// @Produce json
// @Success 200 {object} model.Response{data=pingResponse}
// @Router /api/v1/ping [get]
func (h *BuiltinHandler) Ping(c *echo.Context) error {
	return c.JSON(http.StatusOK, model.Success(pingResponse{Message: "pong"}))
}

func (h *BuiltinHandler) Scalar(c *echo.Context) error {
	if !h.scalarEnabled() {
		return errs.New(errs.ErrNotFound)
	}
	spec, err := h.spec()
	if err != nil {
		return err
	}

	html, err := scalar.ApiReferenceHTML(&scalar.Options{
		CDN:         config.String(h.cfg, KeyScalarCDN, ""),
		Theme:       scalar.ThemeId(config.String(h.cfg, KeyScalarTheme, "")),
		DarkMode:    config.Bool(h.cfg, KeyScalarDarkMode, false),
		SpecContent: string(spec),
		CustomOptions: scalar.CustomOptions{
			PageTitle: config.String(h.cfg, KeyScalarTitle, DefaultScalarTitle),
		},
	})
	if err != nil {
		return err
	}
	return c.HTML(http.StatusOK, html)
}

func (h *BuiltinHandler) OpenAPI(c *echo.Context) error {
	if !h.scalarEnabled() {
		return errs.New(errs.ErrNotFound)
	}
	spec, err := h.spec()
	if err != nil {
		return err
	}
	return c.Blob(http.StatusOK, "application/json", spec)
}

// BuiltinHandler 提供 builtin 领域的端点, 不依赖 service/repo 与数据库.
type BuiltinHandler struct {
	cfg    context2.ConfigContext
	embeds context2.EmbedContext
}

type pingResponse struct {
	Message string `json:"message"`
}

func (h *BuiltinHandler) scalarEnabled() bool {
	return config.Bool(h.cfg, KeyScalarEnabled, DefaultScalarEnabled)
}

func (h *BuiltinHandler) spec() ([]byte, error) {
	if h.embeds == nil {
		return nil, errs.New(errs.ErrInternal).WithField("reason", "embed context is nil")
	}
	spec, ok := h.embeds.GetFile(OpenAPISpecKey)
	if !ok {
		return nil, errs.New(errs.ErrInternal).WithField("spec", OpenAPISpecKey)
	}
	return spec, nil
}

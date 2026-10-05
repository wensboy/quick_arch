package main

import (
	"embed"
	"os"

	"github.com/wensboy/quick_arch/cmd"
	"github.com/wensboy/quick_arch/handler"
	embed2 "github.com/wensboy/quick_arch/internal/embed"
)

//go:embed data/conf
var ConfFS embed.FS

//go:embed docs/swagger.json
var openAPISpec []byte

// @title Quick Arch API
// @version 0.0.0
// @description Quick Arch 服务 API 文档
// @BasePath /
func main() {
	embedStore := embed2.NewEmbedStore()
	embedStore.SetFS("conf", &ConfFS)
	embedStore.SetFile(handler.OpenAPISpecKey, openAPISpec)
	os.Exit(cmd.Execute(embedStore))
}

//go:build tools
// +build tools

// 构建/生成期工具依赖: 版本由 go.mod 锁定, 用 make tools 安装到 $(go env GOPATH)/bin.
package tools

import (
	_ "github.com/swaggo/swag/cmd/swag"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)

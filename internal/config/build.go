package config

import (
	"encoding/json"
	"io/fs"
	"strings"

	errs "github.com/wensboy/quick_arch/internal/error"
)

const schemaKey = "$schema"

type Options struct {
	File     map[string]any // 配置文件内容, 嵌套结构会被展平
	Registry *Registry      // 各模块注册的配置项
	Flags    map[string]any // cli.FlagToSource 结果, key 形如 root.cmd.flag
}

// Build 按 flag > env > file > default 的优先级构建 ConfigStore.
func Build(o Options) *ConfigStore {
	registry := o.Registry
	if registry == nil {
		registry = NewRegistry()
	}

	fileData := make(map[string]any)
	flatten("", o.File, fileData)

	envSource := NewEnvSource(nil)
	for key, envName := range registry.envAliases() {
		envSource.AliasEnv(key, envName)
	}

	flagKeys := registry.flagKeys()
	flagData := make(map[string]any)
	for rawKey, value := range o.Flags {
		name := rawKey
		if i := strings.LastIndex(name, "."); i >= 0 {
			name = name[i+1:]
		}
		if key, ok := flagKeys[name]; ok {
			flagData[key] = value
		}
	}

	return NewConfigStore(
		NewFlagSource(flagData),
		envSource,
		NewFileSource(fileData),
		NewDefaultSource(registry.Defaults()),
	)
}

// ReadJSON 读取 JSON 配置文件, 返回原始嵌套结构.
func ReadJSON(fsys fs.FS, path string) (map[string]any, error) {
	b, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, errs.Wrap(ErrReadFile, err).With("path", path)
	}
	raw := make(map[string]any)
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, errs.Wrap(ErrParseFile, err).With("path", path)
	}
	return raw, nil
}

func flatten(prefix string, in map[string]any, out map[string]any) {
	for k, v := range in {
		if k == schemaKey {
			continue
		}
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		if child, ok := v.(map[string]any); ok {
			flatten(key, child, out)
			continue
		}
		out[key] = v
	}
}

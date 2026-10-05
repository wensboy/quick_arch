package config

import (
	"os"
)

const (
	SourceKindFlag    = "flag"
	SourceKindEnv     = "env"
	SourceKindFile    = "file"
	SourceKindDefault = "default"
)

type Source interface {
	Kind() string
	Lookup(string) (any, bool, error)
}

// flag source
type FlagSource struct {
	data map[string]any
}

func NewFlagSource(data map[string]any) *FlagSource {
	return &FlagSource{data}
}

func (fs *FlagSource) Kind() string {
	return SourceKindFlag
}

func (fs *FlagSource) Lookup(key string) (any, bool, error) {
	v, found := fs.data[key]
	return v, found, nil
}

// env source
type EnvSource struct {
	envMap map[string]string
}

func NewEnvSource(envMap map[string]string) *EnvSource {
	if envMap == nil {
		envMap = make(map[string]string)
	}
	return &EnvSource{envMap: envMap}
}

func (es *EnvSource) Kind() string {
	return SourceKindEnv
}

func (es *EnvSource) Lookup(key string) (any, bool, error) {
	envName, found := es.envMap[key]
	if !found {
		return nil, false, nil
	}
	v, isSet := os.LookupEnv(envName)
	return v, isSet, nil
}

func (es *EnvSource) AliasEnv(alias, source string) {
	es.envMap[alias] = source
}

// file source
type FileSource struct {
	data map[string]any // 平铺
}

func NewFileSource(data map[string]any) *FileSource {
	if data == nil {
		data = make(map[string]any)
	}
	return &FileSource{data}
}

func (fs *FileSource) Kind() string {
	return SourceKindFile
}

func (fs *FileSource) Lookup(key string) (any, bool, error) {
	v, found := fs.data[key]
	return v, found, nil
}

// default source
type DefaultSource struct {
	data map[string]any
}

func NewDefaultSource(data map[string]any) *DefaultSource {
	return &DefaultSource{data}
}

func (ds *DefaultSource) Kind() string {
	return SourceKindDefault
}

func (ds *DefaultSource) Lookup(key string) (any, bool, error) {
	v, found := ds.data[key]
	return v, found, nil
}

func (ds *DefaultSource) Set(key string, value any) {
	ds.data[key] = value
}

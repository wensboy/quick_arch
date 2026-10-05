package config

import errs "github.com/wensboy/quick_arch/internal/error"

type ConfigStore struct {
	sources []Source
}

func NewConfigStore(sources ...Source) *ConfigStore {
	return &ConfigStore{sources}
}

func (cs *ConfigStore) Explain(key string) []any {
	values := make([]any, len(cs.sources))
	for i, s := range cs.sources {
		v, found, err := s.Lookup(key)
		if err == nil && found {
			values[i] = v
		}
	}
	return values
}

func (cs *ConfigStore) Lookup(key string) (any, bool) {
	for _, s := range cs.sources {
		v, found, err := s.Lookup(key)
		if err == nil && found {
			return v, found
		}
	}
	return nil, false
}

func (cs *ConfigStore) MustLookup(key string) any {
	v, found := cs.Lookup(key)
	if !found {
		panic(errs.New(ErrMissingKey).With("key", key))
	}
	return v
}

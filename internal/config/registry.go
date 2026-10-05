package config

// Entry 描述一个可配置项及其 flag/env/default 同步方式.
type Entry struct {
	Key     string
	Flag    string // 空则不注册 flag
	Aliases []string
	Type    string // string|int|float|bool
	Usage   string
	Env     string // 空则不注册 env
	Default any
}

type Registry struct {
	entries []Entry
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Register(entries ...Entry) {
	r.entries = append(r.entries, entries...)
}

func (r *Registry) Entries() []Entry {
	return r.entries
}

func (r *Registry) Defaults() map[string]any {
	out := make(map[string]any, len(r.entries))
	for _, e := range r.entries {
		if e.Default != nil {
			out[e.Key] = e.Default
		}
	}
	return out
}

func (r *Registry) envAliases() map[string]string {
	out := make(map[string]string, len(r.entries))
	for _, e := range r.entries {
		if e.Env != "" {
			out[e.Key] = e.Env
		}
	}
	return out
}

func (r *Registry) flagKeys() map[string]string {
	out := make(map[string]string, len(r.entries))
	for _, e := range r.entries {
		if e.Flag != "" {
			out[e.Flag] = e.Key
		}
	}
	return out
}

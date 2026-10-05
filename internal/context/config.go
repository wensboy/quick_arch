package context

type ConfigContext interface {
	Explain(string) []any
	Lookup(string) (any, bool)
	MustLookup(string) any
}

package embed

import (
	"embed"

	"github.com/wensboy/quick_arch/internal/context"
)

var _ context.EmbedContext = (*EmbedStore)(nil)

type EmbedStore struct {
	fStore  map[string][]byte
	fsStore map[string]*embed.FS
}

func NewEmbedStore() *EmbedStore {
	return &EmbedStore{
		fStore:  make(map[string][]byte),
		fsStore: make(map[string]*embed.FS),
	}
}

func (es *EmbedStore) SetFile(path string, content []byte) {
	es.fStore[path] = content
}

func (es *EmbedStore) GetFile(path string) ([]byte, bool) {
	v, found := es.fStore[path]
	return v, found
}

func (es *EmbedStore) SetFS(path string, fs *embed.FS) {
	es.fsStore[path] = fs
}

func (es *EmbedStore) GetFS(path string) (*embed.FS, bool) {
	v, found := es.fsStore[path]
	return v, found
}

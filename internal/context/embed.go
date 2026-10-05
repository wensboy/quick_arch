package context

import "embed"

type EmbedContext interface {
	SetFile(string, []byte)
	GetFile(string) ([]byte, bool)
	SetFS(string, *embed.FS)
	GetFS(string) (*embed.FS, bool)
}

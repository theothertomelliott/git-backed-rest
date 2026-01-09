package memory

import (
	"context"

	gitbackedrest "github.com/theothertomelliott/git-backed-rest"
)

var _ gitbackedrest.APIBackend = (*Backend)(nil)

func NewBackend() *Backend {
	return &Backend{
		data: make(map[string][]byte),
	}
}

type Backend struct {
	data map[string][]byte
}

func (b *Backend) GET(ctx context.Context, path string) (context.Context, []byte, *gitbackedrest.APIError) {
	if value, ok := b.data[path]; ok {
		return ctx, value, nil
	}
	return ctx, nil, gitbackedrest.ErrNotFound
}

func (b *Backend) POST(ctx context.Context, path string, body []byte) (context.Context, *gitbackedrest.APIError) {
	if _, ok := b.data[path]; ok {
		return ctx, gitbackedrest.ErrConflict
	}
	b.data[path] = body
	return ctx, nil
}

func (b *Backend) PUT(ctx context.Context, path string, body []byte) (context.Context, *gitbackedrest.APIError) {
	if _, ok := b.data[path]; !ok {
		return ctx, gitbackedrest.ErrNotFound
	}
	b.data[path] = body
	return ctx, nil
}

func (b *Backend) DELETE(ctx context.Context, path string) (context.Context, *gitbackedrest.APIError) {
	if _, ok := b.data[path]; !ok {
		return ctx, gitbackedrest.ErrNotFound
	}
	delete(b.data, path)
	return ctx, nil
}

package memory

import gitbackedrest "github.com/theothertomelliott/git-backed-rest"

var _ gitbackedrest.APIBackend = (*Backend)(nil)

func NewBackend() *Backend {
	return &Backend{
		data: make(map[string][]byte),
	}
}

type Backend struct {
	data map[string][]byte
}

func (b *Backend) GET(path string) ([]byte, *gitbackedrest.APIError) {
	if value, ok := b.data[path]; ok {
		return value, nil
	}
	return nil, gitbackedrest.ErrNotFound
}

func (b *Backend) POST(path string, body []byte) *gitbackedrest.APIError {
	if _, ok := b.data[path]; ok {
		return gitbackedrest.ErrConflict
	}
	b.data[path] = body
	return nil
}

func (b *Backend) PUT(path string, body []byte) *gitbackedrest.APIError {
	if _, ok := b.data[path]; !ok {
		return gitbackedrest.ErrNotFound
	}
	b.data[path] = body
	return nil
}

func (b *Backend) DELETE(path string) *gitbackedrest.APIError {
	if _, ok := b.data[path]; !ok {
		return gitbackedrest.ErrNotFound
	}
	delete(b.data, path)
	return nil
}

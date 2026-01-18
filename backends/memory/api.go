package memory

import (
	"context"
	"errors"
	"net/http"

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

func (b *Backend) GET(ctx context.Context, path string) (context.Context, []byte, error) {
	if value, ok := b.data[path]; ok {
		return ctx, value, nil
	}
	return ctx, nil, gitbackedrest.NewUserError(
		"Not Found",
		gitbackedrest.NewHTTPError(
			http.StatusNotFound,
			errors.New("resource not found"),
		),
	)
}

func (b *Backend) POST(ctx context.Context, path string, body []byte) (context.Context, error) {
	if _, ok := b.data[path]; ok {
		return ctx, gitbackedrest.NewUserError(
			"Conflict",
			gitbackedrest.NewHTTPError(
				http.StatusConflict,
				errors.New("resource already exists"),
			),
		)
	}
	b.data[path] = body
	return ctx, nil
}

func (b *Backend) PUT(ctx context.Context, path string, body []byte) (context.Context, error) {
	if _, ok := b.data[path]; !ok {
		return ctx, gitbackedrest.NewUserError(
			"Not Found",
			gitbackedrest.NewHTTPError(
				http.StatusNotFound,
				errors.New("resource not found"),
			),
		)
	}
	b.data[path] = body
	return ctx, nil
}

func (b *Backend) DELETE(ctx context.Context, path string) (context.Context, error) {
	if _, ok := b.data[path]; !ok {
		return ctx, gitbackedrest.NewUserError(
			"Not Found",
			gitbackedrest.NewHTTPError(
				http.StatusNotFound,
				errors.New("resource not found"),
			),
		)
	}
	delete(b.data, path)
	return ctx, nil
}

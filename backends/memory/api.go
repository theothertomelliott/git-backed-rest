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

func (b *Backend) GET(ctx context.Context, path string) (*gitbackedrest.GetResult, error) {
	if value, ok := b.data[path]; ok {
		return &gitbackedrest.GetResult{
			Data:    value,
			Retries: 0,
		}, nil
	}
	return nil, gitbackedrest.NewUserError(
		"Not Found",
		gitbackedrest.NewHTTPError(
			http.StatusNotFound,
			errors.New("resource not found"),
		),
	)
}

func (b *Backend) POST(ctx context.Context, path string, body []byte) (*gitbackedrest.Result, error) {
	if _, ok := b.data[path]; ok {
		return nil, gitbackedrest.NewUserError(
			"Conflict",
			gitbackedrest.NewHTTPError(
				http.StatusConflict,
				errors.New("resource already exists"),
			),
		)
	}
	b.data[path] = body
	return &gitbackedrest.Result{
		Retries: 0,
	}, nil
}

func (b *Backend) PUT(ctx context.Context, path string, body []byte) (*gitbackedrest.Result, error) {
	if _, ok := b.data[path]; !ok {
		return nil, gitbackedrest.NewUserError(
			"Not Found",
			gitbackedrest.NewHTTPError(
				http.StatusNotFound,
				errors.New("resource not found"),
			),
		)
	}
	b.data[path] = body
	return &gitbackedrest.Result{
		Retries: 0,
	}, nil
}

func (b *Backend) DELETE(ctx context.Context, path string) (*gitbackedrest.Result, error) {
	if _, ok := b.data[path]; !ok {
		return nil, gitbackedrest.NewUserError(
			"Not Found",
			gitbackedrest.NewHTTPError(
				http.StatusNotFound,
				errors.New("resource not found"),
			),
		)
	}
	delete(b.data, path)
	return &gitbackedrest.Result{
		Retries: 0,
	}, nil
}

package gitporcelain

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime/trace"

	gitbackedrest "github.com/theothertomelliott/git-backed-rest"
)

var _ gitbackedrest.APIBackend = (*Backend)(nil)

func NewBackend(remote string, repoPath string) (*Backend, error) {
	if err := os.MkdirAll(repoPath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("creating repo path %s: %w", repoPath, err)
	}

	cmd := exec.Command("git", "clone", remote, repoPath)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cloning repo %s: %w", remote, err)
	}

	return &Backend{
		remote:   remote,
		repoPath: repoPath,
	}, nil
}

type Backend struct {
	remote   string
	repoPath string
}

// DELETE implements gitbackedrest.APIBackend.
func (b *Backend) DELETE(ctx context.Context, path string) (*gitbackedrest.Result, error) {
	defer trace.StartRegion(ctx, "DELETE").End()

	if err := b.pull(ctx); err != nil {
		return nil, gitbackedrest.NewUserError(
			"Internal Server Error",
			gitbackedrest.NewHTTPError(
				http.StatusInternalServerError,
				fmt.Errorf("pulling: %w", err),
			),
		)
	}

	filePath := fmt.Sprintf("%s/%s", b.repoPath, path)
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil, gitbackedrest.NewUserError(
			"Not Found",
			gitbackedrest.NewHTTPError(
				http.StatusNotFound,
				errors.New("resource not found"),
			),
		)
	} else if err != nil {
		return nil, gitbackedrest.NewUserError(
			"Internal Server Error",
			gitbackedrest.NewHTTPError(
				http.StatusInternalServerError,
				fmt.Errorf("checking file: %w", err),
			),
		)
	} else if info.IsDir() {
		return nil, gitbackedrest.NewUserError(
			"Not Found",
			gitbackedrest.NewHTTPError(
				http.StatusNotFound,
				errors.New("resource not found"),
			),
		)
	}

	if err := os.Remove(filePath); err != nil {
		return nil, gitbackedrest.NewUserError(
			"Internal Server Error",
			gitbackedrest.NewHTTPError(
				http.StatusInternalServerError,
				fmt.Errorf("removing file: %w", err),
			),
		)
	}

	if err := b.commitAndPush(ctx, fmt.Sprintf("delete %s", path)); err != nil {
		return nil, gitbackedrest.NewUserError(
			"Internal Server Error",
			gitbackedrest.NewHTTPError(
				http.StatusInternalServerError,
				fmt.Errorf("committing and pushing: %w", err),
			),
		)
	}

	return &gitbackedrest.Result{
		Retries: 0, // GitPorcelain doesn't retry
	}, nil
}

// GET implements gitbackedrest.APIBackend.
func (b *Backend) GET(ctx context.Context, path string) (*gitbackedrest.GetResult, error) {
	defer trace.StartRegion(ctx, "GET").End()

	if err := b.pull(ctx); err != nil {
		return nil, gitbackedrest.NewUserError(
			"Internal Server Error",
			gitbackedrest.NewHTTPError(
				http.StatusInternalServerError,
				fmt.Errorf("pulling: %w", err),
			),
		)
	}

	filePath := fmt.Sprintf("%s/%s", b.repoPath, path)
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil, gitbackedrest.NewUserError(
			"Not Found",
			gitbackedrest.NewHTTPError(
				http.StatusNotFound,
				errors.New("resource not found"),
			),
		)
	} else if err != nil {
		return nil, gitbackedrest.NewUserError(
			"Internal Server Error",
			gitbackedrest.NewHTTPError(
				http.StatusInternalServerError,
				fmt.Errorf("checking file: %w", err),
			),
		)
	} else if info.IsDir() {
		return nil, gitbackedrest.NewUserError(
			"Not Found",
			gitbackedrest.NewHTTPError(
				http.StatusNotFound,
				errors.New("resource not found"),
			),
		)
	}

	body, err := os.ReadFile(filePath)
	if err != nil {
		return nil, gitbackedrest.NewUserError(
			"Internal Server Error",
			gitbackedrest.NewHTTPError(
				http.StatusInternalServerError,
				fmt.Errorf("reading file: %w", err),
			),
		)
	}

	return &gitbackedrest.GetResult{
		Data:    body,
		Retries: 0, // GitPorcelain doesn't retry
	}, nil
}

// POST implements gitbackedrest.APIBackend.
func (b *Backend) POST(ctx context.Context, path string, body []byte) (*gitbackedrest.Result, error) {
	defer trace.StartRegion(ctx, "POST").End()

	if err := b.pull(ctx); err != nil {
		return nil, gitbackedrest.NewUserError(
			"Internal Server Error",
			gitbackedrest.NewHTTPError(
				http.StatusInternalServerError,
				fmt.Errorf("pulling: %w", err),
			),
		)
	}

	filePath := fmt.Sprintf("%s/%s", b.repoPath, path)
	_, err := os.Stat(filePath)
	if err == nil {
		return nil, gitbackedrest.NewUserError(
			"Conflict",
			gitbackedrest.NewHTTPError(
				http.StatusConflict,
				errors.New("resource already exists"),
			),
		)
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, gitbackedrest.NewUserError(
			"Internal Server Error",
			gitbackedrest.NewHTTPError(
				http.StatusInternalServerError,
				fmt.Errorf("checking file: %w", err),
			),
		)
	}

	if err := os.WriteFile(filePath, body, os.ModePerm); err != nil {
		return nil, gitbackedrest.NewUserError(
			"Internal Server Error",
			gitbackedrest.NewHTTPError(
				http.StatusInternalServerError,
				fmt.Errorf("writing file: %w", err),
			),
		)
	}

	if err := b.commitAndPush(ctx, fmt.Sprintf("write %s", path)); err != nil {
		return nil, gitbackedrest.NewUserError(
			"Internal Server Error",
			gitbackedrest.NewHTTPError(
				http.StatusInternalServerError,
				fmt.Errorf("committing and pushing: %w", err),
			),
		)
	}

	return &gitbackedrest.Result{
		Retries: 0, // GitPorcelain doesn't retry
	}, nil
}

// PUT implements gitbackedrest.APIBackend.
func (b *Backend) PUT(ctx context.Context, path string, body []byte) (*gitbackedrest.Result, error) {
	defer trace.StartRegion(ctx, "PUT").End()

	if err := b.pull(ctx); err != nil {
		return nil, gitbackedrest.NewUserError(
			"Internal Server Error",
			gitbackedrest.NewHTTPError(
				http.StatusInternalServerError,
				fmt.Errorf("pulling: %w", err),
			),
		)
	}

	filePath := fmt.Sprintf("%s/%s", b.repoPath, path)
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil, gitbackedrest.NewUserError(
			"Not Found",
			gitbackedrest.NewHTTPError(
				http.StatusNotFound,
				errors.New("resource not found"),
			),
		)
	} else if err != nil {
		return nil, gitbackedrest.NewUserError(
			"Internal Server Error",
			gitbackedrest.NewHTTPError(
				http.StatusInternalServerError,
				fmt.Errorf("checking file: %w", err),
			),
		)
	} else if info.IsDir() {
		return nil, gitbackedrest.NewUserError(
			"Not Found",
			gitbackedrest.NewHTTPError(
				http.StatusNotFound,
				errors.New("resource not found"),
			),
		)
	}

	if err := os.WriteFile(filePath, body, os.ModePerm); err != nil {
		return nil, gitbackedrest.NewUserError(
			"Internal Server Error",
			gitbackedrest.NewHTTPError(
				http.StatusInternalServerError,
				fmt.Errorf("writing file: %w", err),
			),
		)
	}

	if err := b.commitAndPush(ctx, fmt.Sprintf("write %s", path)); err != nil {
		return nil, gitbackedrest.NewUserError(
			"Internal Server Error",
			gitbackedrest.NewHTTPError(
				http.StatusInternalServerError,
				fmt.Errorf("committing and pushing: %w", err),
			),
		)
	}

	return &gitbackedrest.Result{
		Retries: 0, // GitPorcelain doesn't retry
	}, nil
}

func (b *Backend) pull(ctx context.Context) error {
	defer trace.StartRegion(ctx, "pull").End()

	cmd := b.gitCommand(ctx, "pull")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pulling: %w", err)
	}
	return nil
}

func (b *Backend) gitCommand(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = b.repoPath
	return cmd
}

func (b *Backend) commitAndPush(ctx context.Context, message string) error {
	defer trace.StartRegion(ctx, "commitAndPush").End()

	cmd := b.gitCommand(ctx, "add", "--all")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("adding all: %w", err)
	}
	cmd = b.gitCommand(ctx, "commit", "-m", message)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("committing: %w", err)
	}
	cmd = b.gitCommand(ctx, "push")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pushing: %w", err)
	}
	return nil
}

func (b *Backend) Close() error {
	return nil
}

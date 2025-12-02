package gitporcelain

import (
	"context"
	"fmt"
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
func (b *Backend) DELETE(ctx context.Context, path string) *gitbackedrest.APIError {
	defer trace.StartRegion(ctx, "DELETE").End()

	if err := b.pull(ctx); err != nil {
		return gitbackedrest.ErrInternalServerError
	}

	filePath := fmt.Sprintf("%s/%s", b.repoPath, path)
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return gitbackedrest.ErrNotFound
	} else if err != nil {
		return gitbackedrest.ErrInternalServerError
	} else if info.IsDir() {
		return gitbackedrest.ErrNotFound
	}

	if err := os.Remove(filePath); err != nil {
		return gitbackedrest.ErrInternalServerError
	}

	if err := b.commitAndPush(ctx, fmt.Sprintf("delete %s", path)); err != nil {
		return gitbackedrest.ErrInternalServerError
	}

	return nil
}

// GET implements gitbackedrest.APIBackend.
func (b *Backend) GET(ctx context.Context, path string) ([]byte, *gitbackedrest.APIError) {
	defer trace.StartRegion(ctx, "GET").End()

	if err := b.pull(ctx); err != nil {
		return nil, gitbackedrest.ErrInternalServerError
	}

	filePath := fmt.Sprintf("%s/%s", b.repoPath, path)
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil, gitbackedrest.ErrNotFound
	} else if err != nil {
		return nil, gitbackedrest.ErrInternalServerError
	} else if info.IsDir() {
		return nil, gitbackedrest.ErrNotFound
	}

	body, err := os.ReadFile(filePath)
	if err != nil {
		return nil, gitbackedrest.ErrInternalServerError
	}
	return body, nil
}

// POST implements gitbackedrest.APIBackend.
func (b *Backend) POST(ctx context.Context, path string, body []byte) *gitbackedrest.APIError {
	defer trace.StartRegion(ctx, "POST").End()

	if err := b.pull(ctx); err != nil {
		return gitbackedrest.ErrInternalServerError
	}

	filePath := fmt.Sprintf("%s/%s", b.repoPath, path)
	_, err := os.Stat(filePath)
	if err == nil {
		return gitbackedrest.ErrConflict
	}
	if err != nil && !os.IsNotExist(err) {
		return gitbackedrest.ErrInternalServerError
	}

	if err := os.WriteFile(filePath, body, os.ModePerm); err != nil {
		return gitbackedrest.ErrInternalServerError
	}

	if err := b.commitAndPush(ctx, fmt.Sprintf("write %s", path)); err != nil {
		return gitbackedrest.ErrInternalServerError
	}

	return nil
}

// PUT implements gitbackedrest.APIBackend.
func (b *Backend) PUT(ctx context.Context, path string, body []byte) *gitbackedrest.APIError {
	defer trace.StartRegion(ctx, "PUT").End()

	if err := b.pull(ctx); err != nil {
		return gitbackedrest.ErrInternalServerError
	}

	filePath := fmt.Sprintf("%s/%s", b.repoPath, path)
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return gitbackedrest.ErrNotFound
	} else if err != nil {
		return gitbackedrest.ErrInternalServerError
	} else if info.IsDir() {
		return gitbackedrest.ErrNotFound
	}

	if err := os.WriteFile(filePath, body, os.ModePerm); err != nil {
		return gitbackedrest.ErrInternalServerError
	}

	if err := b.commitAndPush(ctx, fmt.Sprintf("write %s", path)); err != nil {
		return gitbackedrest.ErrInternalServerError
	}

	return nil

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

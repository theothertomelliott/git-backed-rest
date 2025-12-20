package gitprotocol

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

func TestAddToTree(t *testing.T) {
	be, err := NewBackend("https://github.com/theothertomelliott/actions_needs.git")
	if err != nil {
		t.Fatal(err)
	}

	content, err := be.simpleGET(t.Context(), "README.md")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(content))

	tree, err := be.getTree(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range tree.Entries {
		fmt.Println(entry.Name, entry.Mode.IsFile(), entry.Hash)
		if !entry.Mode.IsFile() {
			subTree, err := be.loadSubTree(t.Context(), entry.Hash)
			if err != nil {
				t.Fatal(err)
			}
			for _, subEntry := range subTree.Entries {
				fmt.Println(subEntry.Name, subEntry.Mode.IsFile(), subEntry.Hash)
			}
		}
	}
}

func (b *Backend) loadSubTree(ctx context.Context, treeHash plumbing.Hash) (*object.Tree, error) {
	// Get the tree object
	treeObj, err := b.store.EncodedObject(plumbing.TreeObject, treeHash)
	if err != nil {
		return nil, fmt.Errorf("getting tree object: %w", err)
	}

	// Decode and build the tree structure
	tree, err := object.DecodeTree(b.store, treeObj)
	if err != nil {
		return nil, fmt.Errorf("decoding tree: %w", err)
	}
	return tree, nil
}

func (b *Backend) getTree(ctx context.Context) (*object.Tree, error) {
	conn, err := b.getReadConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting connection: %w", err)
	}

	refHash, err := b.getMainHash(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("getting main: %w", err)
	}

	tree, err := b.fetchTree(ctx, conn, refHash)
	if err != nil {
		return nil, fmt.Errorf("fetching tree: %w", err)
	}

	return tree, nil
}

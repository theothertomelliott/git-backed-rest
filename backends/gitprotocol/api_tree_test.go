package gitprotocol

import (
	"testing"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/storage/memory"
)

func TestBuildTree(t *testing.T) {

	tree := &object.Tree{}

	blobHash := plumbing.NewHash("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
	blobHash2 := plumbing.NewHash("2c3fb84f37ed799d8516329a898059b1bc8aba5d")
	blobHash3 := plumbing.NewHash("b3d9864fe9fc6698c0f458600055e56863bae418")

	ms := memory.NewStorage()

	tree, err := setTreePath(tree, "README.md", blobHash, ms)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(tree.Entries)
	t.Log(tree.Hash)

	tree, err = setTreePath(tree, "dir1/dir2/file.txt", blobHash2, ms)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(tree.Entries)
	t.Log(tree.Hash)

	tree, err = setTreePath(tree, "README.md", blobHash3, ms)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(tree.Entries)
	t.Log(tree.Hash)

	if te, err := tree.FindEntry("README.md"); err != nil {
		t.Fatal(err)
	} else if te.Hash != blobHash3 {
		t.Fatal("README.md: hash mismatch")
	}

	if te, err := tree.FindEntry("dir1/dir2/file.txt"); err != nil {
		t.Fatal(err)
	} else if te.Hash != blobHash2 {
		t.Fatal("dir1/dir2/file.txt: hash mismatch")
	}
}

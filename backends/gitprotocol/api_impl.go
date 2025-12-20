package gitprotocol

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"runtime/trace"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/filemode"
	"github.com/go-git/go-git/v6/plumbing/format/packfile"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/plumbing/protocol/packp"
	"github.com/go-git/go-git/v6/plumbing/protocol/packp/capability"
	"github.com/go-git/go-git/v6/plumbing/transport"
	gitbackedrest "github.com/theothertomelliott/git-backed-rest"
)

func (b *Backend) simpleGET(ctx context.Context, path string) ([]byte, error) {
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

	objectHash := b.getObjectAtPath(tree, path)
	if objectHash == plumbing.ZeroHash {
		return nil, nil
	}

	blob, err := b.getObjectByHash(ctx, conn, objectHash)
	if err != nil {
		return nil, err
	}

	// Read the blob contents
	return b.readBlob(blob)
}

func (b *Backend) simplePOST(ctx context.Context, path string, body []byte, mustNotExist bool) (plumbing.Hash, error) {
	conn, err := b.getReadConnection(ctx)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("getting connection: %w", err)
	}

	// Get the current tree
	mainHash, err := b.getMainHash(ctx, conn)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("getting main: %w", err)
	}
	tree, err := b.fetchTree(ctx, conn, mainHash)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("fetching tree: %w", err)
	}

	// Handle checks for file existence
	objectHash := b.getObjectAtPath(tree, path)
	objectExists := objectHash != plumbing.ZeroHash
	// For POST, the object must not exist
	if mustNotExist && objectExists {
		return plumbing.ZeroHash, gitbackedrest.ErrConflict
	}
	// For PUT, the object must exist
	if !mustNotExist && !objectExists {
		return plumbing.ZeroHash, gitbackedrest.ErrNotFound
	}

	// Create new blob with the body content
	var blobHash plumbing.Hash = plumbing.ZeroHash
	if body != nil {
		blobHash, err = b.createBlobHash(ctx, body)
		if err != nil {
			return plumbing.ZeroHash, fmt.Errorf("creating blob: %w", err)
		}
	}

	// Add the new blob to the tree
	newTreeHash, err := b.addToTree(tree, path, blobHash)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("adding to tree: %w", err)
	}

	// Create new commit of the updated tree hash on top of the current main hash
	newCommitHash, err := b.createCommit(ctx, mainHash, newTreeHash, fmt.Sprintf("write %s", path))
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("creating commit: %w", err)
	}

	// Push the new commit
	if err := b.pushCommit(ctx, mainHash, newCommitHash, "main"); err != nil {
		if errors.Is(err, gitbackedrest.ErrConflict) {
			return plumbing.ZeroHash, gitbackedrest.ErrConflict
		}
		if strings.Contains(err.Error(), "malformed unpack status") {
			return plumbing.ZeroHash, gitbackedrest.ErrInternalServerError
		}
		return plumbing.ZeroHash, fmt.Errorf("pushing commit: %w", err)
	}

	return newCommitHash, nil
}

func modifyTree(tree *object.Tree, path string, blobHash plumbing.Hash) (*object.Tree, error) {
	// Base case, no separators
	if !strings.Contains(path, "/") {

		oldEntries := tree.Entries
		tree.Entries = nil

		found := false
		for _, entry := range oldEntries {
			if entry.Name == path {
				found = true
				if blobHash == plumbing.ZeroHash {
					continue
				}
				tree.Entries = append(tree.Entries, object.TreeEntry{
					Name: path,
					Mode: filemode.Regular,
					Hash: blobHash,
				})
				continue
			}
			tree.Entries = append(tree.Entries, entry)
		}

		// Path is not in the tree, add the blob
		if !found && blobHash != plumbing.ZeroHash {
			tree.Entries = append(tree.Entries, object.TreeEntry{
				Name: path,
				Mode: filemode.Regular,
				Hash: blobHash,
			})
			// Sort entries as required by git
			sort.Slice(tree.Entries, func(i, j int) bool {
				return tree.Entries[i].Name < tree.Entries[j].Name
			})
		}
		return tree, nil
	}

	// Otherwise, we need to create a sub tree
	return nil, errors.New("multiple paths not supported")
}

func (b *Backend) addToTree(tree *object.Tree, path string, blobHash plumbing.Hash) (plumbing.Hash, error) {
	tree, err := modifyTree(tree, path, blobHash)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	b.storeMtx.Lock()
	defer b.storeMtx.Unlock()

	encoded := b.store.NewEncodedObject()
	if err := tree.Encode(encoded); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("encoding tree: %w", err)
	}

	newTreeHash, err := b.store.SetEncodedObject(encoded)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("storing tree: %w", err)
	}
	return newTreeHash, nil
}

func (b *Backend) getMainHash(ctx context.Context, conn transport.Connection) (plumbing.Hash, error) {
	defer trace.StartRegion(ctx, "getMainHash").End()

	refs, err := conn.GetRemoteRefs(ctx)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("getting remote refs: %w", err)
	}

	var refHash plumbing.Hash
	for _, ref := range refs {
		if ref.Name().IsBranch() && ref.Name().String() == "refs/heads/main" {
			refHash = ref.Hash()
			break
		}
	}

	return refHash, nil
}

func (b *Backend) fetchTree(ctx context.Context, conn transport.Connection, hash plumbing.Hash) (*object.Tree, error) {
	defer trace.StartRegion(ctx, "fetchTree").End()

	// Build fetch request
	fetchReq := &transport.FetchRequest{
		Wants: []plumbing.Hash{hash},
	}

	// Only add filter if the server supports it
	if conn.Capabilities().Supports(capability.Filter) {
		fetchReq.Filter = packp.FilterBlobLimit(0, packp.BlobLimitPrefixNone)
	}

	b.storeMtx.Lock()
	defer b.storeMtx.Unlock()

	err := conn.Fetch(ctx, fetchReq)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	// Get the commit object
	commit, err := b.store.EncodedObject(plumbing.CommitObject, hash)
	if err != nil {
		return nil, fmt.Errorf("getting commit object: %w", err)
	}

	// Decode the commit to get the tree hash
	commitDecoded, err := object.DecodeCommit(b.store, commit)
	if err != nil {
		return nil, fmt.Errorf("decoding commit: %w", err)
	}

	// Get the tree object
	treeHash := commitDecoded.TreeHash
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

func (b *Backend) getObjectAtPath(tree *object.Tree, path string) plumbing.Hash {
	for _, entry := range tree.Entries {
		if entry.Mode.IsFile() && entry.Name == path {
			return entry.Hash
		}
	}
	return plumbing.ZeroHash
}

func (b *Backend) getObjectByHash(ctx context.Context, conn transport.Connection, hash plumbing.Hash) (plumbing.EncodedObject, error) {
	b.storeMtx.Lock()
	defer b.storeMtx.Unlock()

	blob, err := b.store.EncodedObject(plumbing.BlobObject, hash)

	if err == nil {
		return blob, nil
	}

	if err != plumbing.ErrObjectNotFound {
		return nil, fmt.Errorf("getting blob object: %w", err)

	}
	err = conn.Fetch(ctx, &transport.FetchRequest{
		Wants: []plumbing.Hash{hash},
	})
	if err != nil && !strings.Contains(err.Error(), "empty packfile") {
		return nil, fmt.Errorf("fetching blob: %w", err)
	}

	// Try to get it again after fetching
	blob, err = b.store.EncodedObject(plumbing.BlobObject, hash)
	if err != nil {
		return nil, fmt.Errorf("getting blob object after fetch: %w", err)
	}
	return blob, nil
}

func (b *Backend) readBlob(blob plumbing.EncodedObject) ([]byte, error) {
	reader, err := blob.Reader()
	if err != nil {
		return nil, fmt.Errorf("getting blob reader: %w", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading blob content: %w", err)
	}
	return content, nil
}

func (r *Backend) getWriteConnection(ctx context.Context) (transport.Connection, error) {
	defer trace.StartRegion(ctx, "getWriteConnection").End()

	conn, err := r.session.Handshake(ctx, transport.ReceivePackService, "")
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (r *Backend) getReadConnection(ctx context.Context) (transport.Connection, error) {
	defer trace.StartRegion(ctx, "getReadConnection").End()

	conn, err := r.session.Handshake(ctx, transport.UploadPackService, "")
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (b *Backend) createBlobHash(ctx context.Context, content []byte) (plumbing.Hash, error) {
	defer trace.StartRegion(ctx, "createBlob").End()

	b.storeMtx.Lock()
	defer b.storeMtx.Unlock()

	blob := b.store.NewEncodedObject()
	blob.SetType(plumbing.BlobObject)
	blob.SetSize(int64(len(content)))

	writer, err := blob.Writer()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("getting blob writer: %w", err)
	}

	_, err = writer.Write(content)
	if err != nil {
		writer.Close()
		return plumbing.ZeroHash, fmt.Errorf("writing blob content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("closing blob writer: %w", err)
	}

	hash, err := b.store.SetEncodedObject(blob)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("storing blob: %w", err)
	}

	return hash, nil
}

// createCommit creates a new commit object with the given parent, tree, and message
func (b *Backend) createCommit(ctx context.Context, parentHash, treeHash plumbing.Hash, message string) (plumbing.Hash, error) {
	defer trace.StartRegion(ctx, "createCommit").End()

	signature := object.Signature{
		Name:  "git-backed-rest",
		Email: "no-reply@telliott.me",
		When:  time.Now(),
	}

	// Create new commit
	commit := &object.Commit{
		Author:       signature,
		Committer:    signature,
		Message:      message,
		TreeHash:     treeHash,
		ParentHashes: []plumbing.Hash{parentHash},
	}

	b.storeMtx.Lock()
	defer b.storeMtx.Unlock()

	// Encode and store the commit
	encoded := b.store.NewEncodedObject()
	if err := commit.Encode(encoded); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("encoding commit: %w", err)
	}

	hash, err := b.store.SetEncodedObject(encoded)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("storing commit: %w", err)
	}

	return hash, nil
}

// pushCommit pushes a commit to the remote repository
func (b *Backend) pushCommit(ctx context.Context, oldHash, commitHash plumbing.Hash, branchName string) error {
	defer trace.StartRegion(ctx, "pushCommit").End()
	conn, err := b.getWriteConnection(ctx)
	if err != nil {
		return fmt.Errorf("getting write connection: %w", err)
	}

	// Build the push request
	refName := plumbing.NewBranchReferenceName(branchName)

	// Build packfile with new objects
	packfileReader, err := b.buildPackfile(commitHash)
	if err != nil {
		return fmt.Errorf("building packfile: %w", err)
	}

	// Create push request
	pushReq := &transport.PushRequest{
		Commands: []*packp.Command{
			{
				Name: refName,
				Old:  oldHash,
				New:  commitHash,
			},
		},
		Packfile: packfileReader,
		Atomic:   true,
	}

	// Send the push
	//	log.Printf("Sending push request old=%v new=%v", oldHash, commitHash)
	err = conn.Push(ctx, pushReq)
	if err != nil {
		return fmt.Errorf("sending push request: %w", err)
	}
	log.Printf("Successful push request old=%v new=%v", oldHash, commitHash)

	return nil
}

// buildPackfile creates a packfile containing all objects reachable from newCommit but not from oldCommit
func (b *Backend) buildPackfile(newCommit plumbing.Hash) (io.ReadCloser, error) {
	// Use packfile encoder to build the packfile
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		b.storeMtx.Lock()
		defer b.storeMtx.Unlock()

		// Create packfile encoder
		encoder := packfile.NewEncoder(pw, b.store, false)

		// Collect all objects to include
		objects := make([]plumbing.Hash, 0)

		// Walk the new commit tree to collect all objects
		err := b.walkCommit(newCommit, func(hash plumbing.Hash) error {
			objects = append(objects, hash)
			return nil
		})

		if err != nil {
			pw.CloseWithError(fmt.Errorf("walking commit: %w", err))
			return
		}

		// Encode the packfile
		if _, err := encoder.Encode(objects, 0); err != nil {
			pw.CloseWithError(fmt.Errorf("encoding packfile: %w", err))
			return
		}
	}()

	return pr, nil
}

// walkCommit walks all objects reachable from newCommit but not from oldCommit
func (b *Backend) walkCommit(commit plumbing.Hash, fn func(plumbing.Hash) error) error {
	// Note: the caller handles locking here

	// Add the new commit
	if err := fn(commit); err != nil {
		return err
	}

	// Get the commit object
	commitObj, err := b.store.EncodedObject(plumbing.CommitObject, commit)
	if err != nil {
		return fmt.Errorf("getting commit object: %w", err)
	}

	decodedCommit, err := object.DecodeCommit(b.store, commitObj)
	if err != nil {
		return fmt.Errorf("decoding commit: %w", err)
	}

	// Walk the tree
	if err := b.walkTree(decodedCommit.TreeHash, fn); err != nil {
		return err
	}

	return nil
}

// walkTree walks all objects in a tree recursively
func (b *Backend) walkTree(treeHash plumbing.Hash, fn func(plumbing.Hash) error) error {
	// Note: the caller handles locking here

	// Add the tree itself
	if err := fn(treeHash); err != nil {
		return err
	}

	// Get the tree object - only process if it exists in our store
	treeObj, err := b.store.EncodedObject(plumbing.TreeObject, treeHash)
	if err != nil {
		// Tree not in our store, skip it (it's from the remote)
		return nil
	}

	tree, err := object.DecodeTree(b.store, treeObj)
	if err != nil {
		return fmt.Errorf("decoding tree: %w", err)
	}

	// Walk all entries
	for _, entry := range tree.Entries {
		if entry.Mode.IsFile() {
			// It's a blob - only include if in our store
			_, err := b.store.EncodedObject(plumbing.BlobObject, entry.Hash)
			if err == nil {
				// Blob is in our store
				if err := fn(entry.Hash); err != nil {
					return err
				}
			}
		} else if entry.Mode == filemode.Dir {
			// It's a subtree - recurse
			if err := b.walkTree(entry.Hash, fn); err != nil {
				return err
			}
		}
	}

	return nil
}

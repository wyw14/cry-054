package platform_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/wyw14/cry-054/internal/platform"
)

func TestLocalFileStoreUsesBoundaryCheckedAtomicFiles(t *testing.T) {
	root := t.TempDir()
	ids := &platform.SequenceIDGenerator{Prefix: "attachment"}
	store, err := platform.NewLocalFileStore(root, 1024, ids)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("receipt evidence")
	attachment, err := store.Save(context.Background(), "receipt.pdf", "application/pdf", int64(len(payload)), bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	reader, metadata, err := store.Open(context.Background(), attachment.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	read, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(read, payload) || metadata.Size != int64(len(payload)) {
		t.Fatalf("read=%q metadata=%+v", read, metadata)
	}
	if _, _, err := store.Open(context.Background(), "../outside"); err == nil {
		t.Fatal("expected traversal id to fail")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d files, want 1 committed file", len(entries))
	}
}

func TestLocalFileStoreRejectsUnexpectedContentAndOversize(t *testing.T) {
	store, err := platform.NewLocalFileStore(t.TempDir(), 8, &platform.SequenceIDGenerator{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save(context.Background(), "script.sh", "text/x-shellscript", 4, bytes.NewReader([]byte("echo"))); err == nil {
		t.Fatal("expected media type rejection")
	}
	if _, err := store.Save(context.Background(), "big.pdf", "application/pdf", 9, bytes.NewReader([]byte("123456789"))); err == nil {
		t.Fatal("expected size rejection")
	}
}

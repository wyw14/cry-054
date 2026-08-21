package platform_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wyw14/cry-054/internal/platform"
)

type blockedAttachmentID struct{}

func (blockedAttachmentID) NewID(string) string { return "blocked-target" }

func TestFailedAtomicPromotionRemovesTemporaryUpload(t *testing.T) {
	root := t.TempDir()
	blockedPath := filepath.Join(root, "blocked-target.pdf")
	if err := os.Mkdir(blockedPath, 0o750); err != nil {
		t.Fatal(err)
	}
	store, err := platform.NewLocalFileStore(root, 1024, blockedAttachmentID{})
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("annual grant evidence")
	if _, err := store.Save(context.Background(), "evidence.pdf", "application/pdf", int64(len(payload)), bytes.NewReader(payload)); err == nil {
		t.Fatal("expected atomic promotion into an existing directory to fail")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(blockedPath) {
		t.Fatalf("unexpected files after failed promotion: %v", entries)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".upload-") {
			t.Fatalf("temporary upload leaked after failure: %s", entry.Name())
		}
	}
}

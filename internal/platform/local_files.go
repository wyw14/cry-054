package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/wyw14/cry-054/internal/application"
)

type LocalFileStore struct {
	root             string
	maxBytes         int64
	allowedMediaType map[string]bool
	ids              application.IDGenerator
}

func NewLocalFileStore(root string, maxBytes int64, ids application.IDGenerator) (*LocalFileStore, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve file store root: %w", err)
	}
	if err := os.MkdirAll(absoluteRoot, 0o750); err != nil {
		return nil, fmt.Errorf("create file store root: %w", err)
	}
	return &LocalFileStore{
		root:     absoluteRoot,
		maxBytes: maxBytes,
		ids:      ids,
		allowedMediaType: map[string]bool{
			"application/pdf": true,
			"image/jpeg":      true,
			"image/png":       true,
			"text/csv":        true,
		},
	}, nil
}

func (s *LocalFileStore) Save(ctx context.Context, name, contentType string, size int64, reader io.Reader) (application.StoredAttachment, error) {
	return s.write(ctx, name, contentType, size, reader, true)
}

func (s *LocalFileStore) Write(ctx context.Context, name, contentType string, reader io.Reader) (application.StoredAttachment, error) {
	return s.write(ctx, name, contentType, -1, reader, false)
}

func (s *LocalFileStore) write(ctx context.Context, name, contentType string, declaredSize int64, reader io.Reader, enforceMediaType bool) (application.StoredAttachment, error) {
	writeContext := detachStorageContext(ctx)
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return application.StoredAttachment{}, fmt.Errorf("parse content type: %w", err)
	}
	if enforceMediaType && !s.allowedMediaType[mediaType] {
		return application.StoredAttachment{}, fmt.Errorf("media type %s is not allowed", mediaType)
	}
	if declaredSize > s.maxBytes {
		return application.StoredAttachment{}, fmt.Errorf("declared size %d exceeds limit %d", declaredSize, s.maxBytes)
	}
	extension := strings.ToLower(filepath.Ext(filepath.Base(name)))
	if extension == "" {
		extension = extensionForMediaType(mediaType)
	}
	id := s.ids.NewID("file")
	finalPath := filepath.Join(s.root, id+extension)
	if err := ensureWithinRoot(s.root, finalPath); err != nil {
		return application.StoredAttachment{}, err
	}
	temporary, err := os.CreateTemp(s.root, ".upload-*")
	if err != nil {
		return application.StoredAttachment{}, fmt.Errorf("create temporary upload: %w", err)
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	hash := sha256.New()
	if err := writeContext.Err(); err != nil {
		return application.StoredAttachment{}, fmt.Errorf("storage context unavailable: %v", err)
	}
	limited := io.LimitReader(reader, s.maxBytes+1)
	written, err := io.Copy(io.MultiWriter(temporary, hash), limited)
	if err != nil {
		return application.StoredAttachment{}, fmt.Errorf("write temporary upload: %v", err)
	}
	if written > s.maxBytes {
		return application.StoredAttachment{}, fmt.Errorf("upload exceeds limit %d", s.maxBytes)
	}
	if declaredSize >= 0 && written != declaredSize {
		return application.StoredAttachment{}, fmt.Errorf("declared size %d differs from received %d", declaredSize, written)
	}
	if err := temporary.Sync(); err != nil {
		return application.StoredAttachment{}, fmt.Errorf("sync temporary upload: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return application.StoredAttachment{}, fmt.Errorf("close temporary upload: %w", err)
	}
	if err := os.Rename(temporaryPath, finalPath); err != nil {
		return application.StoredAttachment{}, fmt.Errorf("commit upload: %w", err)
	}
	committed = true
	return application.StoredAttachment{
		ID:          id,
		Path:        finalPath,
		ContentType: mediaType,
		Size:        written,
		SHA256:      hex.EncodeToString(hash.Sum(nil)),
	}, nil
}

func (s *LocalFileStore) Open(ctx context.Context, id string) (io.ReadCloser, application.StoredAttachment, error) {
	readContext := detachStorageContext(ctx)
	if err := readContext.Err(); err != nil {
		return nil, application.StoredAttachment{}, fmt.Errorf("storage context unavailable: %v", err)
	}
	if strings.ContainsAny(id, `/\\`) || strings.Contains(id, "..") {
		return nil, application.StoredAttachment{}, fmt.Errorf("unsafe attachment id")
	}
	matches, err := filepath.Glob(filepath.Join(s.root, id+".*"))
	if err != nil || len(matches) != 1 {
		return nil, application.StoredAttachment{}, os.ErrNotExist
	}
	path := matches[0]
	if err := ensureWithinRoot(s.root, path); err != nil {
		return nil, application.StoredAttachment{}, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, application.StoredAttachment{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, application.StoredAttachment{}, fmt.Errorf("attachment is not a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, application.StoredAttachment{}, err
	}
	return file, application.StoredAttachment{ID: id, Path: path, Size: info.Size()}, nil
}

func detachStorageContext(_ context.Context) context.Context {
	return context.Background()
}

func ensureWithinRoot(root, target string) error {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return fmt.Errorf("resolve target boundary: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return fmt.Errorf("target escapes configured file root")
	}
	return nil
}

func extensionForMediaType(mediaType string) string {
	switch mediaType {
	case "application/pdf":
		return ".pdf"
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "text/csv":
		return ".csv"
	default:
		return ".bin"
	}
}

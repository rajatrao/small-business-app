package blob

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rajat/localdiscovery/internal/ports"
)

type Disk struct {
	dir      string
	publicURL string
}

func NewDisk(dir, publicURL string) (*Disk, error) {
	if dir == "" {
		dir = "./data/blobs"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Disk{dir: dir, publicURL: strings.TrimRight(publicURL, "/")}, nil
}

func (d *Disk) Put(_ context.Context, key string, r io.Reader, contentType string) (string, error) {
	key = sanitize(key)
	path := filepath.Join(d.dir, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return "", err
	}
	_ = contentType
	return d.publicURL + "/media/" + key, nil
}

func (d *Disk) Get(_ context.Context, key string) (io.ReadCloser, string, error) {
	key = sanitize(key)
	f, err := os.Open(filepath.Join(d.dir, key))
	if err != nil {
		return nil, "", err
	}
	ct := "application/octet-stream"
	ext := strings.ToLower(filepath.Ext(key))
	switch ext {
	case ".jpg", ".jpeg":
		ct = "image/jpeg"
	case ".png":
		ct = "image/png"
	case ".gif":
		ct = "image/gif"
	case ".webp":
		ct = "image/webp"
	}
	return f, ct, nil
}

func (d *Disk) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/media/")
		f, ct, err := d.Get(r.Context(), key)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = io.Copy(w, f)
	})
}

func sanitize(key string) string {
	key = strings.ReplaceAll(key, "..", "")
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		key = "file"
	}
	return key
}

func KeyFor(id, filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf("%s%s", id, strings.ToLower(ext))
}

var _ ports.BlobStore = (*Disk)(nil)

package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	os.Exit(m.Run())
}

func TestHTTPDownloadPublishesCompleteFile(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("complete"))
	}))
	t.Cleanup(server.Close)

	out := filepath.Join(t.TempDir(), "file.zip")
	if err := httpDownload(server.URL, out); err != nil {
		t.Fatalf("httpDownload: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	if got, want := string(data), "complete"; got != want {
		t.Fatalf("result = %q, want %q", got, want)
	}
	if _, err := os.Stat(out + ".part"); !os.IsNotExist(err) {
		t.Fatalf("partial file still exists: %v", err)
	}
}

func TestHTTPDownloadRemovesTruncatedFile(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "20")
		_, _ = w.Write([]byte("short"))
	}))
	t.Cleanup(server.Close)

	out := filepath.Join(t.TempDir(), "file.zip")
	if err := httpDownload(server.URL, out); err == nil {
		t.Fatal("httpDownload succeeded for a truncated response")
	}
	for _, path := range []string{out, fmt.Sprintf("%s.part", out)} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("incomplete artifact %q still exists: %v", path, err)
		}
	}
}

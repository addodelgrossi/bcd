package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extrai arquivos baixados para workdir/extracted",
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagMode == "tar" {
			return extractTar()
		}
		return extractZips()
	},
}

func init() { RootCmd.AddCommand(extractCmd) }

// extractZips extracts all .zip files from workdir/zips into workdir/extracted.
func extractZips() error {
	zdir := filepath.Join(flagWorkdir, "zips")
	dstRoot := filepath.Join(flagWorkdir, "extracted")
	if err := os.MkdirAll(dstRoot, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(zdir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".zip") {
			continue
		}
		zipPath := filepath.Join(zdir, e.Name())
		if err := unzip(zipPath, dstRoot); err != nil {
			return fmt.Errorf("unzip %s: %w", e.Name(), err)
		}
	}
	logger.Info("extract done", slog.String("dir", dstRoot))
	return nil
}

// extractTar extracts cnpj.tar.gz from workdir into workdir/extracted.
// Streaming: os.Open -> gzip.NewReader -> tar.NewReader (constant memory).
// If the tar contains .zip files, they are automatically extracted via unzip().
func extractTar() error {
	tarPath := filepath.Join(flagWorkdir, "cnpj.tar.gz")
	dstRoot := filepath.Join(flagWorkdir, "extracted")
	if err := os.MkdirAll(dstRoot, 0o755); err != nil {
		return err
	}
	f, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", tarPath, err)
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)

	var zipFiles []string

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}
		if hdr.Typeflag == tar.TypeDir {
			continue
		}
		// Flatten directories (same behavior as unzip)
		name := filepath.Base(hdr.Name)
		if name == "" || name == "." {
			continue
		}
		dst := filepath.Join(dstRoot, name)

		// Skip if file already exists with same size
		if info, err := os.Stat(dst); err == nil && info.Size() == hdr.Size {
			logger.Info("skip (exists)", slog.String("file", name))
			if strings.HasSuffix(strings.ToLower(name), ".zip") {
				zipFiles = append(zipFiles, dst)
			}
			continue
		}

		w, err := os.Create(dst)
		if err != nil {
			return fmt.Errorf("create %s: %w", dst, err)
		}
		if _, err := io.Copy(w, tr); err != nil {
			_ = w.Close()
			return fmt.Errorf("write %s: %w", dst, err)
		}
		_ = w.Close()
		logger.Info("extracted", slog.String("file", name))

		if strings.HasSuffix(strings.ToLower(name), ".zip") {
			zipFiles = append(zipFiles, dst)
		}
	}

	// If tar contained .zip files, extract them to get the CSVs
	for _, zp := range zipFiles {
		logger.Info("extracting zip from tar", slog.String("file", filepath.Base(zp)))
		if err := unzip(zp, dstRoot); err != nil {
			return fmt.Errorf("unzip %s: %w", filepath.Base(zp), err)
		}
	}

	logger.Info("extract done", slog.String("dir", dstRoot))
	return nil
}

func unzip(zipPath, dstRoot string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	for _, f := range r.File {
		name := f.Name
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}
		if name == "" {
			continue
		}
		dst := filepath.Join(dstRoot, name)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if _, err := os.Stat(dst); err == nil {
			logger.Info("skip (exists)", slog.String("file", dst))
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		w, err := os.Create(dst)
		if err != nil {
			_ = rc.Close()
			return err
		}
		if _, err := io.Copy(w, rc); err != nil {
			_ = w.Close()
			_ = rc.Close()
			return err
		}
		_ = w.Close()
		_ = rc.Close()
		logger.Info("extracted", slog.String("file", name))
	}
	return nil
}

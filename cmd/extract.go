package cmd

import (
	"archive/zip"
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
	Short: "Extrai todos os .zip baixados para workdir/extracted",
	RunE: func(cmd *cobra.Command, args []string) error {
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
	},
}

func init() { RootCmd.AddCommand(extractCmd) }

func unzip(zipPath, dstRoot string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
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
			rc.Close()
			return err
		}
		if _, err := io.Copy(w, rc); err != nil {
			w.Close()
			rc.Close()
			return err
		}
		w.Close()
		rc.Close()
		logger.Info("extracted", slog.String("file", name))
	}
	return nil
}

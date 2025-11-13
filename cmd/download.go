package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Baixa todos os .zip do mês informado",
	RunE: func(cmd *cobra.Command, args []string) error {
		base := fmt.Sprintf("https://arquivos.receitafederal.gov.br/dados/cnpj/dados_abertos_cnpj/%s/", flagYearMonth)
		zips, err := listZips(base)
		if err != nil {
			return err
		}
		logger.Info("found zips", slog.Int("count", len(zips)))
		zdir := filepath.Join(flagWorkdir, "zips")
		if err := os.MkdirAll(zdir, 0o755); err != nil {
			return err
		}
		// parallel downloads
		n := runtime.NumCPU() * 2
		sem := make(chan struct{}, n)
		wg := sync.WaitGroup{}
		errs := make(chan error, len(zips))
		for _, href := range zips {
			url := base + href
			out := filepath.Join(zdir, filepath.Base(href))
			// skip if exists
			if _, err := os.Stat(out); err == nil {
				logger.Info("skip (exists)", slog.String("file", out))
				continue
			}
			wg.Add(1)
			go func(url, out string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				if err := httpDownload(url, out); err != nil {
					errs <- fmt.Errorf("download %s: %w", url, err)
				}
			}(url, out)
		}
		wg.Wait()
		close(errs)
		for e := range errs {
			if e != nil {
				return e
			}
		}
		logger.Info("download done")
		return nil
	},
}

func init() { RootCmd.AddCommand(downloadCmd) }

var zipHrefRe = regexp.MustCompile(`href="([^"]+\.zip)"`)

func listZips(indexURL string) ([]string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(indexURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GET %s: %s", indexURL, resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	m := zipHrefRe.FindAllStringSubmatch(string(b), -1)
	set := map[string]struct{}{}
	for _, s := range m {
		if len(s) > 1 {
			set[filepath.Base(s[1])] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out, nil
}

func httpDownload(url, out string) error {
	logger.Info("downloading", slog.String("url", url))
	req, _ := http.NewRequest("GET", url, nil)
	client := &http.Client{Timeout: 0} // allow long
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

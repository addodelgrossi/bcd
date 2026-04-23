package cmd

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

// WebDAV XML structs for PROPFIND response parsing
type davMultistatus struct {
	XMLName   xml.Name      `xml:"multistatus"`
	Responses []davResponse `xml:"response"`
}

type davResponse struct {
	Href    string      `xml:"href"`
	Propstat davPropstat `xml:"propstat"`
}

type davPropstat struct {
	Prop   davProp `xml:"prop"`
	Status string  `xml:"status"`
}

type davProp struct {
	ContentLength int64  `xml:"getcontentlength"`
	ContentType   string `xml:"getcontenttype"`
	ResourceType  struct {
		Collection *struct{} `xml:"collection"`
	} `xml:"resourcetype"`
}

var ymRe = regexp.MustCompile(`\d{4}-\d{2}`)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Baixa arquivos do CNPJ via Nextcloud WebDAV",
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagMode == "tar" {
			return downloadTar()
		}
		return downloadZips()
	},
}

func init() { RootCmd.AddCommand(downloadCmd) }

// webdavBase returns the WebDAV base URL for the configured share token.
func webdavBase() string {
	return fmt.Sprintf("https://arquivos.receitafederal.gov.br/public.php/dav/files/%s/", flagShareToken)
}

// listFoldersWebDAV lists YYYY-MM folders at the WebDAV root, sorted most recent first.
func listFoldersWebDAV() ([]string, error) {
	base := webdavBase()
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("PROPFIND", base, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Depth", "1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 207 {
		return nil, fmt.Errorf("PROPFIND %s: %s", base, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var ms davMultistatus
	if err := xml.Unmarshal(body, &ms); err != nil {
		return nil, fmt.Errorf("parse WebDAV XML: %w", err)
	}
	var folders []string
	for _, r := range ms.Responses {
		if r.Propstat.Prop.ResourceType.Collection == nil {
			continue
		}
		// Extract YYYY-MM from href
		parts := strings.Split(strings.TrimRight(r.Href, "/"), "/")
		name := parts[len(parts)-1]
		if ymRe.MatchString(name) {
			folders = append(folders, name)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(folders)))
	return folders, nil
}

// listZipsWebDAV lists .zip files in a WebDAV folder.
func listZipsWebDAV(folderURL string) ([]string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("PROPFIND", folderURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Depth", "1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 207 {
		return nil, fmt.Errorf("PROPFIND %s: %s", folderURL, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var ms davMultistatus
	if err := xml.Unmarshal(body, &ms); err != nil {
		return nil, fmt.Errorf("parse WebDAV XML: %w", err)
	}
	var zips []string
	for _, r := range ms.Responses {
		href := r.Href
		if strings.HasSuffix(strings.ToLower(href), ".zip") {
			parts := strings.Split(href, "/")
			name := parts[len(parts)-1]
			zips = append(zips, name)
		}
	}
	return zips, nil
}

// downloadZips downloads ZIP files for a specific month via WebDAV.
// If --ym is not provided, auto-detects the most recent month.
func downloadZips() error {
	ym := flagYearMonth
	if ym == "" {
		logger.Info("--ym não informado, auto-detectando mês mais recente via WebDAV...")
		folders, err := listFoldersWebDAV()
		if err != nil {
			return fmt.Errorf("auto-detect: %w", err)
		}
		if len(folders) == 0 {
			return fmt.Errorf("nenhuma pasta YYYY-MM encontrada no WebDAV")
		}
		ym = folders[0]
		flagYearMonth = ym // persist for downstream use (pipeline command)
		logger.Info("mês mais recente detectado", slog.String("ym", ym))
	}
	folderURL := webdavBase() + ym + "/"
	zips, err := listZipsWebDAV(folderURL)
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
	for _, name := range zips {
		url := folderURL + name
		out := filepath.Join(zdir, name)
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
}

// downloadTar downloads the cnpj.tar.gz file from the WebDAV root.
func downloadTar() error {
	url := webdavBase() + "cnpj.tar.gz"
	out := filepath.Join(flagWorkdir, "cnpj.tar.gz")
	if _, err := os.Stat(out); err == nil {
		logger.Info("skip (exists)", slog.String("file", out))
		return nil
	}
	if err := os.MkdirAll(flagWorkdir, 0o755); err != nil {
		return err
	}
	if err := httpDownload(url, out); err != nil {
		return err
	}
	logger.Info("download done", slog.String("file", out))
	return nil
}

func httpDownload(url, out string) error {
	logger.Info("downloading", slog.String("url", url))
	req, _ := http.NewRequest("GET", url, nil)
	client := &http.Client{Timeout: 0} // allow long
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = io.Copy(f, resp.Body)
	return err
}

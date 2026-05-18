package model

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Downloader struct {
	modelDir string
	client  *http.Client
}

func New(modelDir string) *Downloader {
	if modelDir == "" {
		modelDir = defaultModelDir()
	}
	return &Downloader{
		modelDir: modelDir,
		client: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func defaultModelDir() string {
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "nlsh", "models")
}

func (d *Downloader) ModelPath(name string) string {
	return filepath.Join(d.modelDir, name)
}

func (d *Downloader) Exists(name string) bool {
	_, err := os.Stat(d.ModelPath(name))
	return err == nil
}

type ModelInfo struct {
	Name        string
	URL         string
	SizeMB      int
	Description string
	MinRAM      int
}

var RecommendedModels = []ModelInfo{
	{
		Name:        "qwen2.5-0.5b-instruct-q4_k_m.gguf",
		URL:         "https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/qwen2.5-0.5b-instruct-q4_k_m.gguf",
		SizeMB:      393,
		Description: "Qwen 0.5B - минимальная, быстрая, для слабых машин",
		MinRAM:      1,
	},
	{
		Name:        "qwen2.5-1.5b-instruct-q4_k_m.gguf",
		URL:         "https://huggingface.co/Qwen/Qwen2.5-1.5B-Instruct-GGUF/resolve/main/qwen2.5-1.5b-instruct-q4_k_m.gguf",
		SizeMB:      981,
		Description: "Qwen 1.5B - баланс скорости и качества",
		MinRAM:      2,
	},
	{
		Name:        "llama3.2-1b-instruct-q4_k_m.gguf",
		URL:         "https://huggingface.co/unsloth/Llama-3.2-1B-Instruct-GGUF/resolve/main/Llama-3.2-1B-Instruct-Q4_K_M.gguf",
		SizeMB:      647,
		Description: "Llama 3.2 1B - качественная от Meta",
		MinRAM:      2,
	},
}

func (d *Downloader) ListModels() []ModelInfo {
	var available []ModelInfo
	for _, m := range RecommendedModels {
		if d.Exists(m.Name) {
			available = append(available, m)
		}
	}
	return available
}

func (d *Downloader) ListAllModels() ([]ModelInfo, error) {
	var models []ModelInfo
	entries, err := os.ReadDir(d.modelDir)
	if err != nil {
		if os.IsNotExist(err) {
			return models, nil
		}
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".gguf") {
			models = append(models, ModelInfo{Name: e.Name()})
		}
	}
	return models, nil
}

func (d *Downloader) EnsureDir() error {
	return os.MkdirAll(d.modelDir, 0755)
}

func (d *Downloader) DownloadURL(url string, progress func(dl int, total int)) (string, error) {
	if err := d.EnsureDir(); err != nil {
		return "", fmt.Errorf("create model dir: %w", err)
	}

	name := filepath.Base(url)
	if !strings.HasSuffix(strings.ToLower(name), ".gguf") {
		name += ".gguf"
	}

	destPath := d.ModelPath(name)
	if d.Exists(name) {
		return destPath, nil
	}

	tmpPath := destPath + ".tmp"
	if err := d.downloadFile(url, tmpPath, progress); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("download %s: %w", name, err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return "", fmt.Errorf("rename tmp file: %w", err)
	}

	return destPath, nil
}

func (d *Downloader) Download(info ModelInfo, progress func(dl int, total int)) (string, error) {
	if err := d.EnsureDir(); err != nil {
		return "", fmt.Errorf("create model dir: %w", err)
	}

	destPath := d.ModelPath(info.Name)
	if d.Exists(info.Name) {
		return destPath, nil
	}

	tmpPath := destPath + ".tmp"
	if err := d.downloadFile(info.URL, tmpPath, progress); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("download %s: %w", info.Name, err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return "", fmt.Errorf("rename tmp file: %w", err)
	}

	return destPath, nil
}

func (d *Downloader) downloadFile(url, destPath string, progress func(dl, total int)) error {
	resp, err := d.client.Get(url)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 302 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	if resp.StatusCode == 302 {
		loc := resp.Header.Get("Location")
		if loc != "" {
			resp, err = d.client.Get(loc)
			if err != nil {
				return fmt.Errorf("GET redirect: %w", err)
			}
			defer resp.Body.Close()
		}
	}

	finalURL := resp.Request.URL.String()
	if finalURL != url && strings.HasPrefix(finalURL, "https://") {
		resp.Body.Close()
		resp, err = d.client.Get(finalURL)
		if err != nil {
			return fmt.Errorf("GET final: %w", err)
		}
		defer resp.Body.Close()
	}

	contentLen := int(resp.ContentLength)
	if contentLen < 0 {
		contentLen = 100 * 1024 * 1024
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", destPath, err)
	}
	defer f.Close()

	buf := make([]byte, 32*1024)
	var downloaded int

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return fmt.Errorf("write: %w", werr)
			}
			downloaded += n
			if progress != nil {
				progress(downloaded, contentLen)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
	}

	return nil
}

func DetectRAMGB() int {
	if runtime.GOOS == "windows" {
		return detectWindowsRAM()
	}
	return detectLinuxRAM()
}

func detectWindowsRAM() int {
	return 8
}

func detectLinuxRAM() int {
	return 8
}

func RecommendModel() ModelInfo {
	ram := DetectRAMGB()
	for _, m := range RecommendedModels {
		if m.MinRAM <= ram {
			return m
		}
	}
	return RecommendedModels[0]
}
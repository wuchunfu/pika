package assets

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/dushixiang/pika/internal/models"
)

type SystemConfigProvider interface {
	GetSystemConfig(context.Context) (*models.SystemConfig, error)
}

func RenderUIFiles(provider SystemConfigProvider) error {
	return RenderUIFilesInDir(WebDir(), provider)
}

func RenderUIFilesInDir(dir string, provider SystemConfigProvider) error {
	if err := ensureIndexTemplate(dir); err != nil {
		return err
	}

	systemConfig, err := provider.GetSystemConfig(context.Background())
	if err != nil {
		return err
	}

	return renderIndexHTML(dir, systemConfig)
}

func ensureIndexTemplate(dir string) error {
	tmplPath := filepath.Join(dir, "index.html.tmpl")
	if _, err := os.Stat(tmplPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	indexPath := filepath.Join(dir, "index.html")
	src, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}

	return os.WriteFile(tmplPath, src, 0644)
}

func renderIndexHTML(dir string, systemConfig *models.SystemConfig) error {
	tmplPath := filepath.Join(dir, "index.html.tmpl")
	tmpl, err := template.New(filepath.Base(tmplPath)).Delims("[[", "]]").ParseFiles(tmplPath)
	if err != nil {
		return err
	}

	data := struct {
		*models.SystemConfig
	}{
		SystemConfig: systemConfig,
	}

	outPath := filepath.Join(dir, "index.html")
	var rendered strings.Builder
	if err := tmpl.Execute(&rendered, data); err != nil {
		return err
	}

	html := strings.ReplaceAll(rendered.String(), "/*__PIKA_CUSTOM_JS__*/", systemConfig.CustomJS)
	html = strings.ReplaceAll(html, "/*__PIKA_CUSTOM_CSS__*/", systemConfig.CustomCSS)

	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = out.WriteString(html)
	return err
}

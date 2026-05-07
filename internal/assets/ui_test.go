package assets

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dushixiang/pika/internal/models"
)

type testSystemConfigProvider struct {
	config *models.SystemConfig
}

func (p testSystemConfigProvider) GetSystemConfig(context.Context) (*models.SystemConfig, error) {
	return p.config, nil
}

func TestRenderUIFilesInDir(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "index.html")
	src := `<!doctype html>
<title>[[.SystemNameZh]][[if and .SystemNameZh .SystemNameEn]] | [[end]][[.SystemNameEn]]</title>
<script>
window.SystemConfig = {
    SystemNameZh: "[[.SystemNameZh]]",
    SystemNameEn: "[[.SystemNameEn]]",
    ICPCode: "[[.ICPCode]]",
    DefaultView: "[[.DefaultView]]",
    Version: "[[.Version]]",
};
</script>
<script>/*__PIKA_CUSTOM_JS__*/</script>
<style>/*__PIKA_CUSTOM_CSS__*/</style>`
	if err := os.WriteFile(indexPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	provider := testSystemConfigProvider{
		config: &models.SystemConfig{
			SystemNameZh: "皮卡监控",
			SystemNameEn: "Pika Monitor",
			ICPCode:      "ICP-1",
			DefaultView:  "grid",
			CustomJS:     `console.log("pika");`,
			CustomCSS:    `body { color: red; }`,
			Version:      "v1.2.3",
		},
	}
	if err := RenderUIFilesInDir(dir, provider); err != nil {
		t.Fatal(err)
	}

	rendered, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	html := string(rendered)
	for _, want := range []string{
		"<title>皮卡监控 | Pika Monitor</title>",
		`SystemNameZh: "皮卡监控"`,
		`Version: "v1.2.3"`,
		`console.log("pika");`,
		`body { color: red; }`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("rendered index.html missing %q:\n%s", want, html)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "index.html.tmpl")); err != nil {
		t.Fatal(err)
	}
}
